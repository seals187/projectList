<script lang="ts">
  // "Board" overview: columns per status (default), customer or person
  // responsible. Grouped by status, projects move via drag & drop between and
  // within columns (the order is shared by all users); admins edit a status'
  // name, color and WIP limit via the column title. Filters come from the page.
  import { isAdmin, setBoardHidden, sidebar } from '$lib/state.svelte';
  import { fmtDate, t } from '$lib/i18n.svelte';
  import { barCls, BOARD_STATUSES, statusDotCls, statusDotStyle, statusLabel } from '$lib/projectStatus';
  import { dueUrgent, pct, type OvCustomer, type OvProject } from '$lib/overview';
  import { filter, NO_OWNER } from '$lib/overviewFilter.svelte';
  import { tagsByIds } from '$lib/tags';
  import type { ProjectStatus, SidebarCustomer } from '$lib/types';
  import TeamAvatars from './TeamAvatars.svelte';
  import ProjectMenu from './ProjectMenu.svelte';
  import CustomerMenu from './CustomerMenu.svelte';
  import StatusMenu from './StatusMenu.svelte';
  import { projectCardColor, tintStyle } from '$lib/cardColors';
  import { myColors } from '$lib/myColors.svelte';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    all,
    groups,
    today,
    canCreate,
    oncreate,
    onmove,
    display = { tags: true, percent: true, due: true, owner: true }
  }: {
    all: OvCustomer[]; // all customers (column order incl. filtered-out projects)
    groups: OvCustomer[]; // visible customers with their matching projects
    today: string;
    canCreate: boolean;
    oncreate: (customerId: number | null, status: ProjectStatus) => void;
    onmove: (p: OvProject, status: ProjectStatus, ids: number[]) => void;
    display?: Record<string, boolean>; // header "Display" options
  } = $props();

  // ---- Columns ---------------------------------------------------------------------
  interface Column {
    key: string;
    title: string;
    status?: ProjectStatus; // grouped by status: drop target, WIP limit, status menu
    customer?: SidebarCustomer;
    owner?: { id: number; name: string };
    items: OvProject[];
  }

  // order within a status column: board position, then name (stable sort keeps the name order)
  const byPosition = (a: OvProject, b: OvProject) => a.board_position - b.board_position;
  // customer/person columns: projects in board order (status, then position)
  const statusIndex = (s: ProjectStatus) => (BOARD_STATUSES as readonly ProjectStatus[]).indexOf(s);
  const byStatus = (a: OvProject, b: OvProject) => statusIndex(a.status) - statusIndex(b.status) || byPosition(a, b);

  const everything = $derived(all.flatMap((g) => g.projects));
  const projects = $derived(groups.flatMap((g) => g.projects));
  const byStatusGroup = $derived(filter.group === 'status');

  const columns = $derived.by((): Column[] => {
    if (filter.group === 'customer')
      return groups.map((g) => ({ key: `c${g.customer.id}`, title: g.customer.name, customer: g.customer, items: [...g.projects].sort(byStatus) }));
    if (filter.group === 'owner') {
      const m = new Map<number, Column>();
      for (const p of projects) {
        const id = p.owner_id ?? NO_OWNER;
        if (!m.has(id)) m.set(id, { key: `o${id}`, title: id ? p.owner_name : t('project.noOwner'), owner: { id, name: p.owner_name }, items: [] });
        m.get(id)!.items.push(p);
      }
      // people alphabetically, "nobody" last
      return [...m.values()]
        .sort((a, b) => (a.owner!.id === NO_OWNER ? 1 : b.owner!.id === NO_OWNER ? -1 : a.title.localeCompare(b.title)))
        .map((c) => ({ ...c, items: c.items.sort(byStatus) }));
    }
    return BOARD_STATUSES.map((s) => ({ key: s, title: statusLabel(s), status: s, items: projects.filter((p) => p.status === s).sort(byPosition) }));
  });

  // the only visible customer is preselected when creating a project; new
  // projects always start in "planning", also from another status column
  const singleCustomer = $derived(groups.length === 1 ? groups[0].customer.id : null);
  function addTo(col: Column) {
    oncreate(col.status ? singleCustomer : (col.customer?.id ?? singleCustomer), 'planning');
  }

  // WIP limit (status columns): limit reached = red
  const wip = (col: Column) => (col.status ? (sidebar.statusWip[col.status] ?? 0) : 0);

  // ---- Menus (right click; "…" and quick actions on the card) ------------------------------
  type ProjectSection = 'due' | 'owner' | null;
  let menu = $state<
    | { kind: 'project'; project: OvProject; x: number; y: number; section: ProjectSection }
    | { kind: 'customer'; customer: SidebarCustomer; x: number; y: number }
    | { kind: 'status'; status: ProjectStatus; x: number; y: number }
    | null
  >(null);
  function projectMenu(e: MouseEvent, project: OvProject) {
    e.preventDefault();
    e.stopPropagation();
    menu = { kind: 'project', project, x: e.clientX, y: e.clientY, section: null };
  }
  function customerMenu(e: MouseEvent, customer: SidebarCustomer) {
    e.preventDefault();
    e.stopPropagation();
    menu = { kind: 'customer', customer, x: e.clientX, y: e.clientY };
  }
  // quick action buttons: open the menu below the button (optionally with a section open)
  function quick(e: MouseEvent, project: OvProject, section: ProjectSection) {
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = { kind: 'project', project, x: r.right - 288, y: r.bottom + 4, section };
  }
  function statusMenu(e: MouseEvent, status: ProjectStatus) {
    e.preventDefault();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = { kind: 'status', status, x: r.left, y: r.bottom + 4 };
  }

  // customer filter from the customer menu
  const hidden = $derived(new Set(sidebar.boardHidden));
  const allIds = $derived(all.map((g) => g.customer.id));
  function customerFilter(id: number, mode: 'only' | 'toggle' | 'all') {
    if (mode === 'all') setBoardHidden([]);
    else if (mode === 'only') setBoardHidden(allIds.filter((x) => x !== id));
    else setBoardHidden(hidden.has(id) ? [...hidden].filter((x) => x !== id) : [...hidden, id]);
  }

  // ---- Drag & Drop (grouped by status only) --------------------------------------------
  // target column and the card before which the project is inserted (null = at the end)
  let dragId = $state<number | null>(null);
  let over = $state<{ status: ProjectStatus; before: number | null } | null>(null);

  function dragOver(e: DragEvent, status: ProjectStatus) {
    if (dragId === null) return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
    // first card (except the dragged one) whose middle is below the pointer
    const cards = [...(e.currentTarget as HTMLElement).querySelectorAll<HTMLElement>('[data-card]')].filter((c) => Number(c.dataset.card) !== dragId);
    const next = cards.find((c) => {
      const r = c.getBoundingClientRect();
      return e.clientY < r.top + r.height / 2;
    });
    const before = next ? Number(next.dataset.card) : null;
    if (over?.status !== status || over.before !== before) over = { status, before };
  }

  function drop(e: DragEvent, status: ProjectStatus) {
    e.preventDefault();
    const p = everything.find((x) => x.id === dragId);
    const before = over?.status === status ? over.before : null;
    dragId = null;
    over = null;
    if (!p) return;
    // new order of the whole column, incl. filtered-out projects (they keep their place)
    const column = everything.filter((x) => x.status === status && x.id !== p.id).sort(byPosition);
    const visibleIds = new Set(projects.map((x) => x.id));
    const visible = column.filter((x) => visibleIds.has(x.id));
    let at = before === null ? -1 : column.findIndex((x) => x.id === before);
    if (at < 0) at = visible.length ? column.indexOf(visible[visible.length - 1]) + 1 : column.length;
    const ids = column.map((x) => x.id);
    ids.splice(at, 0, p.id);
    const old = everything.filter((x) => x.status === status).sort(byPosition).map((x) => x.id);
    if (p.status === status && old.join() === ids.join()) return; // nothing changed
    onmove(p, status, ids);
  }

  const quickCls = 'rounded-md p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200';
</script>

<!-- narrow: scroll horizontally; wide: equal-width columns (status: five) -->
<div class="flex min-h-[60vh] gap-3 overflow-x-auto pb-2 {byStatusGroup ? 'xl:grid xl:grid-cols-5 xl:overflow-visible' : ''}">
  {#each columns as col (col.key)}
    {@const limit = wip(col)}
    {@const full = limit > 0 && col.items.length >= limit}
    <section
      class="surface flex w-72 shrink-0 flex-col rounded-xl bg-zinc-100/70 p-2.5 transition-colors dark:bg-zinc-900/60
        {byStatusGroup ? 'xl:w-auto xl:min-w-0' : ''}
        {col.status && over?.status === col.status ? 'ring-2 ring-accent-400/60' : ''}"
      aria-label={col.title}
      ondragover={(e) => col.status && dragOver(e, col.status)}
      ondragleave={(e) => {
        if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) over = null;
      }}
      ondrop={(e) => col.status && drop(e, col.status)}>
      <!-- title takes the full width and wraps (long custom status names are never cut off); count right after it -->
      {#snippet count()}<span class="ml-1.5 text-sm font-normal text-zinc-400 tabular-nums">{col.items.length}</span>{/snippet}
      <header class="{filter.compact ? 'mb-2' : 'mb-3'} flex items-start gap-2 px-1">
        {#if col.status}
          {@const s = col.status}
          {#if isAdmin()}
            <h3 class="min-w-0 flex-1">
              <button
                type="button"
                class="-ml-1 flex max-w-full items-start gap-2 rounded-md px-1 py-0.5 text-left font-semibold hover:bg-zinc-200/70 dark:hover:bg-zinc-800"
                onclick={(e) => statusMenu(e, s)}
                oncontextmenu={(e) => statusMenu(e, s)}
                title={t('menu.statusSettings')}>
                <span class="mt-[0.4rem] size-2.5 shrink-0 rounded-full {statusDotCls(s)}" style={statusDotStyle(s)}></span>
                <span class="min-w-0 break-words">{col.title}{@render count()}</span>
              </button>
            </h3>
          {:else}
            <h3 class="flex min-w-0 flex-1 items-start gap-2 py-0.5 font-semibold">
              <span class="mt-[0.4rem] size-2.5 shrink-0 rounded-full {statusDotCls(s)}" style={statusDotStyle(s)}></span>
              <span class="min-w-0 break-words">{col.title}{@render count()}</span>
            </h3>
          {/if}
        {:else if col.customer}
          {@const c = col.customer}
          <h3 class="min-w-0 flex-1">
            <a href="/customers/{c.id}" class="flex items-start gap-2 py-0.5 font-semibold hover:underline" oncontextmenu={(e) => customerMenu(e, c)}>
              <EntityIcon type="customer" size="xs" class="mt-0.5" />
              <span class="min-w-0 break-words">{c.name}{@render count()}</span>
            </a>
          </h3>
        {:else if col.owner}
          <h3 class="flex min-w-0 flex-1 items-start gap-2 py-0.5 font-semibold">
            {#if col.owner.id}<Avatar name={col.owner.name} id={col.owner.id} size={20} />{:else}<span class="flex size-5 shrink-0 items-center justify-center text-zinc-400"><Icon name="user" size={14} /></span>{/if}
            <span class="min-w-0 break-words">{col.title}{@render count()}</span>
          </h3>
        {/if}
        {#if limit}
          <span class="mt-1 shrink-0 text-[11px] font-semibold tracking-wide whitespace-nowrap tabular-nums {full ? 'text-red-600 dark:text-red-400' : 'text-zinc-400'}" title={t('wip.title')}>
            WIP {col.items.length}/{limit}
          </span>
        {/if}
      </header>

      <div class="flex flex-1 flex-col {filter.compact ? 'gap-1.5' : 'gap-2.5'}">
        {#each col.items as p (p.id)}
          {@const tags = tagsByIds(p.tag_ids)}
          {@const tint = projectCardColor(p, p.customer, myColors.all)}
          {#if col.status && over?.status === col.status && over.before === p.id}<div class="-my-1.5 h-1 shrink-0 rounded-full bg-accent-500"></div>{/if}
          <a
            href="/projects/{p.id}"
            data-card={p.id}
            draggable={canCreate && byStatusGroup}
            oncontextmenu={(e) => projectMenu(e, p)}
            ondragstart={(e) => {
              dragId = p.id;
              e.dataTransfer?.setData('text/plain', String(p.id));
              if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
            }}
            ondragend={() => {
              dragId = null;
              over = null;
            }}
            class="group relative block rounded-xl border bg-white shadow-xs transition-shadow hover:border-accent-300 hover:shadow-md dark:bg-zinc-900 dark:hover:border-accent-500/50
              {filter.compact ? 'px-3 py-2' : 'p-3'}
              {p.status === 'blocked' ? 'border-red-300 dark:border-red-500/50' : 'border-zinc-200 dark:border-zinc-800'}
              {dragId === p.id ? 'opacity-50' : ''}
              {tint ? 'card-tint' : ''}"
            style={tintStyle(tint)}>
            <!-- customer (in customer columns: the status instead) -->
            {#if col.customer}
              <p class="flex items-center gap-1.5 pr-6 text-xs text-zinc-500">
                <span class="size-2 shrink-0 rounded-full {statusDotCls(p.status)}" style={statusDotStyle(p.status)}></span>
                <span class="truncate">{statusLabel(p.status)}</span>
              </p>
            {:else}
              <p class="flex items-center gap-1.5 pr-6 text-xs text-zinc-500" oncontextmenu={(e) => customerMenu(e, p.customer)}>
                <EntityIcon type="customer" size="xs" />
                <span class="truncate">{p.customer.name}</span>
              </p>
            {/if}

            <!-- quick actions on hover (devices with a mouse); without a mouse only "…" (always visible) -->
            <span
              class="absolute top-1.5 right-1.5 flex items-center gap-0.5 rounded-lg bg-white/95 p-0.5 group-hover:opacity-100 focus-within:opacity-100 dark:bg-zinc-900/95 [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:shadow-sm [@media(hover:hover)]:ring-1 [@media(hover:hover)]:ring-zinc-200 dark:[@media(hover:hover)]:ring-zinc-700">
              {#if canCreate}
                <button type="button" class="{quickCls} hidden [@media(hover:hover)]:block" onclick={(e) => quick(e, p, 'owner')} title={t('project.owner')} aria-label="{t('project.owner')}: {p.name}">
                  <Icon name="user" size={13} />
                </button>
                <button type="button" class="{quickCls} hidden [@media(hover:hover)]:block" onclick={(e) => quick(e, p, 'due')} title={t('project.schedule')} aria-label="{t('project.schedule')}: {p.name}">
                  <Icon name="calendar" size={13} />
                </button>
              {/if}
              <button type="button" class={quickCls} onclick={(e) => quick(e, p, null)} title={t('menu.actions')} aria-label="{t('menu.actions')}: {p.name}">
                <Icon name="dots" size={13} />
              </button>
            </span>

            <p class="flex items-start gap-1.5 leading-snug font-semibold break-words {filter.compact ? 'mt-1 text-[13px]' : 'mt-1.5 text-[13px]'}" title={p.name}><EntityIcon type="project" size="xs" class="mt-px" /><span class="line-clamp-2 min-w-0">{p.name}</span></p>

            {#if !filter.compact && display.tags && tags.length}
              <!-- labels as small dots, max. two, then "+N" -->
              <p class="mt-1.5 flex min-w-0 items-center gap-2.5 text-xs text-zinc-600 dark:text-zinc-400" title={tags.map((x) => x.name).join(', ')}>
                {#each tags.slice(0, 2) as tag (tag.id)}
                  <span class="flex min-w-0 items-center gap-1"><span class="size-2 shrink-0 rounded-full" style:background-color={tag.color}></span><span class="truncate">{tag.name}</span></span>
                {/each}
                {#if tags.length > 2}<span class="shrink-0 text-zinc-400">+{tags.length - 2}</span>{/if}
              </p>
            {/if}

            <!-- progress -->
            <div class="flex items-center gap-2 {filter.compact ? 'mt-1.5' : 'mt-2.5'}">
              <span class="block h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
              </span>
              {#if display.percent}<span class="shrink-0 text-xs text-zinc-500 tabular-nums">{pct(p.done, p.total)}%</span>{/if}
            </div>

            {#if !filter.compact && (p.due_date || p.checklists.length || p.team.length || p.owner_id)}
              <div class="mt-2.5 flex items-center gap-3 text-xs text-zinc-500">
                {#if p.due_date && display.due}
                  <span class="flex items-center gap-1 tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : ''}">
                    <Icon name="calendar" size={12} />{fmtDate(p.due_date)}
                  </span>
                {/if}
                {#if p.checklists.length}
                  <span class="flex items-center gap-1 tabular-nums" title={t('project.checklists')}><Icon name="checkSquare" size={12} />{p.checklists.length}</span>
                {/if}
                <span class="ml-auto flex items-center gap-1.5">
                  {#if display.owner}
                    {#if p.owner_id}
                      <span title="{t('project.owner')}: {p.owner_name}"><Avatar name={p.owner_name} id={p.owner_id} size={20} /></span>
                    {/if}
                    <TeamAvatars team={p.team} />
                  {/if}
                </span>
              </div>
            {/if}
          </a>
        {/each}
        {#if col.status && over?.status === col.status && over.before === null && col.items.some((x) => x.id !== dragId)}
          <div class="-my-1.5 h-1 shrink-0 rounded-full bg-accent-500"></div>
        {/if}
        {#if !col.items.length && col.status && dragId !== null}
          <p class="rounded-xl border border-dashed border-zinc-300 py-6 text-center text-sm text-zinc-400 dark:border-zinc-700">{t('overview.dropHere')}</p>
        {/if}
        {#if canCreate && !col.owner}
          <button
            type="button"
            class="flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-left text-sm text-zinc-400 hover:bg-zinc-200/60 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
            onclick={() => addTo(col)}>
            <Icon name="plus" size={14} />{t('overview.addProjectColumn')}
          </button>
        {/if}
      </div>
    </section>
  {/each}
</div>

{#if menu?.kind === 'status'}
  {@const m = menu}
  <StatusMenu status={m.status} x={m.x} y={m.y} onclose={() => (menu = null)} />
{:else if menu?.kind === 'project'}
  {@const m = menu}
  <ProjectMenu project={m.project} customer={m.project.customer} x={m.x} y={m.y} initial={m.section} boardColor onclose={() => (menu = null)} />
{:else if menu?.kind === 'customer'}
  {@const m = menu}
  <CustomerMenu
    customer={m.customer}
    x={m.x}
    y={m.y}
    hidden={hidden.has(m.customer.id)}
    {canCreate}
    onclose={() => (menu = null)}
    oncreate={() => oncreate(m.customer.id, 'planning')}
    onfilter={(mode) => customerFilter(m.customer.id, mode)} />
{/if}
