<script lang="ts">
  // "Cards" overview: one card per customer with progress ring, its projects
  // (status, progress, due date) and a footer with overdue/open items, the next
  // due date and the team. Customers with overdue items come first. Collapsible
  // (shared with the list view): collapsed, header and footer remain as summary.
  // Right click on a project or the customer opens the same menus as on the board.
  import { fmtDate, t } from '$lib/i18n.svelte';
  import { openCustomers, toggleCustomer } from '$lib/overviewOpen.svelte';
  import { barCls, customerCode, customerTone, statusDotCls, statusDotStyle, statusLabel } from '$lib/projectStatus';
  import { dueUrgent, pct, type OvCustomer, type OvProject } from '$lib/overview';
  import { setBoardHidden, sidebar } from '$lib/state.svelte';
  import type { SidebarCustomer } from '$lib/types';
  import TeamAvatars from './TeamAvatars.svelte';
  import ProgressRing from './ProgressRing.svelte';
  import ProjectMenu from './ProjectMenu.svelte';
  import CustomerMenu from './CustomerMenu.svelte';
  import Icon from './Icon.svelte';

  let {
    groups,
    today,
    canCreate,
    oncreate
  }: { groups: OvCustomer[]; today: string; canCreate: boolean; oncreate: (customerId: number) => void } = $props();

  // Context menus as on the board (right click; "…" button on the project row)
  let menu = $state<
    { kind: 'project'; project: OvProject; x: number; y: number } | { kind: 'customer'; customer: SidebarCustomer; x: number; y: number } | null
  >(null);
  function projectMenu(e: MouseEvent, project: OvProject) {
    e.preventDefault();
    e.stopPropagation();
    menu = { kind: 'project', project, x: e.clientX, y: e.clientY };
  }
  function menuButton(e: MouseEvent, project: OvProject) {
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = { kind: 'project', project, x: r.right - 288, y: r.bottom + 4 };
  }
  function customerMenu(e: MouseEvent, customer: SidebarCustomer) {
    e.preventDefault();
    menu = { kind: 'customer', customer, x: e.clientX, y: e.clientY };
  }
  // customer filter (shared with all overview views)
  const hidden = $derived(new Set(sidebar.boardHidden));
  function customerFilter(id: number, mode: 'only' | 'toggle' | 'all') {
    const all = sidebar.customers.map((c) => c.id);
    if (mode === 'all') setBoardHidden([]);
    else if (mode === 'only') setBoardHidden(all.filter((x) => x !== id));
    else setBoardHidden(hidden.has(id) ? [...hidden].filter((x) => x !== id) : [...hidden, id]);
  }

  const MAX_ROWS = 6; // further projects: "+N more" (link to the customer)
  const shortDate = (iso: string) => fmtDate(iso).slice(0, 5); // DD.MM

  const overdueOf = (p: OvProject) => Object.values(p.clExtra).reduce((n, c) => n + c.overdue, 0);

  // per customer: overdue and open items, next due date (open items or project), team
  const cards = $derived(
    groups
      .map((g) => {
        const overdue = g.projects.reduce((n, p) => n + overdueOf(p), 0);
        const open = g.projects.reduce((n, p) => n + (p.total - p.done), 0);
        const dates = g.projects
          .filter((p) => p.status !== 'done')
          .flatMap((p) => [...p.checklists.map((c) => c.next_due), p.due_date ?? ''])
          .filter((d) => d && d >= today)
          .sort();
        const team = new Map<number, { id: number; name: string }>();
        for (const p of g.projects) for (const m of p.team) team.set(m.id, m);
        return { ...g, overdue, open, next: dates[0] ?? '', team: [...team.values()] };
      })
      // overdue first, then customers with projects, then empty ones; otherwise the existing order (name)
      .sort((a, b) => Number(b.overdue > 0) - Number(a.overdue > 0) || Number(b.projects.length > 0) - Number(a.projects.length > 0))
  );
</script>

<div class="grid grid-cols-[repeat(auto-fill,minmax(21rem,1fr))] items-start gap-4">
  {#each cards as g (g.customer.id)}
    {@const open = openCustomers.has(g.customer.id)}
    <section
      class="flex flex-col rounded-2xl border bg-white p-4 shadow-xs dark:bg-zinc-900/40
        {g.overdue ? 'border-red-300 dark:border-red-500/50' : 'border-zinc-200 dark:border-zinc-800'}"
      aria-label={g.customer.name}>
      <!-- customer -->
      <header role="group" class="flex items-center gap-3" oncontextmenu={(e) => customerMenu(e, g.customer)}>
        <button
          type="button"
          class="-ml-1 rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200 {g.projects.length ? '' : 'invisible'}"
          disabled={!g.projects.length}
          onclick={() => toggleCustomer(g.customer.id)}
          aria-expanded={open}
          aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
          <Icon name="chevronDown" size={16} class="transition-transform {open ? '' : '-rotate-90'}" />
        </button>
        <a href="/customers/{g.customer.id}" class="flex size-11 shrink-0 items-center justify-center rounded-xl text-base font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</a>
        <div class="min-w-0 flex-1">
          <a href="/customers/{g.customer.id}" class="block truncate text-[15px] font-semibold hover:underline">{g.customer.name}</a>
          <p class="truncate text-xs text-zinc-500">
            {t('overview.nProjects', { n: g.projects.length })}{g.customer.location ? ` · ${g.customer.location}` : ''}
          </p>
        </div>
        {#if g.projects.length}<ProgressRing done={g.done} total={g.total} size={42} />{/if}
      </header>

      {#if g.projects.length}
        <!-- projects (hidden while collapsed: header and footer stay as summary) -->
        {#if open}
        <ul class="mt-3 divide-y divide-zinc-100 border-t border-zinc-100 dark:divide-zinc-800 dark:border-zinc-800">
          {#each g.projects.slice(0, MAX_ROWS) as p (p.id)}
            <li>
              <a href="/projects/{p.id}" class="group -mx-2 block rounded-lg px-2 py-2 hover:bg-zinc-50 dark:hover:bg-zinc-800/50" oncontextmenu={(e) => projectMenu(e, p)}>
                <!-- name over the full width, "…" on the right -->
                <span class="flex items-start gap-2">
                  <span class="line-clamp-2 min-w-0 flex-1 text-sm leading-snug font-semibold break-words" title={p.name}>{p.name}</span>
                  <!-- "…": on hover (devices with a mouse), always visible without a mouse -->
                  <button
                    type="button"
                    class="-mr-1 shrink-0 rounded-md p-0.5 text-zinc-400 group-hover:opacity-100 hover:bg-zinc-200/70 hover:text-zinc-700 focus:opacity-100 dark:hover:bg-zinc-700 dark:hover:text-zinc-200 [@media(hover:hover)]:opacity-0"
                    onclick={(e) => menuButton(e, p)}
                    title={t('menu.actions')}
                    aria-label="{t('menu.actions')}: {p.name}">
                    <Icon name="dots" size={14} />
                  </button>
                </span>
                <!-- below: status, progress bar, done/total, due date -->
                <span class="mt-1 flex items-center gap-2.5 text-xs text-zinc-500">
                  <span class="flex max-w-[45%] min-w-0 items-center gap-1.5">
                    <span class="size-2 shrink-0 rounded-full {statusDotCls(p.status)}" style={statusDotStyle(p.status)}></span>
                    <span class="truncate">{statusLabel(p.status)}</span>
                  </span>
                  <span class="block h-1 min-w-8 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800" title="{pct(p.done, p.total)}%">
                    <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
                  </span>
                  <span class="shrink-0 tabular-nums" title="{pct(p.done, p.total)}%">{p.done}/{p.total}</span>
                  <span class="w-10 shrink-0 text-right tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : ''}">
                    {p.due_date ? shortDate(p.due_date) : '–'}
                  </span>
                </span>
              </a>
            </li>
          {/each}
        </ul>
        {#if g.projects.length > MAX_ROWS}
          <a href="/customers/{g.customer.id}" class="mt-1 text-xs font-medium text-accent-600 hover:underline dark:text-accent-400">{t('cards.more', { n: g.projects.length - MAX_ROWS })}</a>
        {/if}
        {/if}

        <!-- footer: overdue/open items, next due date, team -->
        <footer class="flex min-h-[35px] items-center gap-3 border-t border-zinc-100 pt-3 text-xs whitespace-nowrap dark:border-zinc-800 {open && g.projects.length <= MAX_ROWS ? 'mt-auto' : 'mt-3'}">
          {#if g.overdue}
            <span class="flex items-center gap-1 font-semibold text-red-600 dark:text-red-400"><Icon name="alert" size={13} />{t('list.nOverdue', { n: g.overdue })}</span>
          {:else}
            <span class="text-zinc-500 tabular-nums">{t('cards.open', { n: g.open })}</span>
          {/if}
          {#if g.next}
            <span class="flex items-center gap-1 text-zinc-500 tabular-nums"><Icon name="calendar" size={12} />{t('list.next', { date: shortDate(g.next) })}</span>
          {/if}
          <span class="ml-auto"><TeamAvatars team={g.team} /></span>
        </footer>
      {:else}
        <!-- no projects yet: compact like a collapsed card -->
        <footer class="mt-3 flex min-h-[35px] items-center gap-3 border-t border-zinc-100 pt-3 text-xs whitespace-nowrap dark:border-zinc-800">
          <span class="text-zinc-400">{t('overview.noProjectsYet')}</span>
          {#if canCreate}
            <button type="button" class="ml-auto font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => oncreate(g.customer.id)}>
              + {t('cards.projectFromTemplate')}
            </button>
          {/if}
        </footer>
      {/if}
    </section>
  {/each}
</div>

{#if menu?.kind === 'project'}
  {@const m = menu}
  <ProjectMenu project={m.project} customer={m.project.customer} x={m.x} y={m.y} onclose={() => (menu = null)} />
{:else if menu?.kind === 'customer'}
  {@const m = menu}
  <CustomerMenu
    customer={m.customer}
    x={m.x}
    y={m.y}
    hidden={hidden.has(m.customer.id)}
    {canCreate}
    onclose={() => (menu = null)}
    oncreate={() => oncreate(m.customer.id)}
    onfilter={(mode) => customerFilter(m.customer.id, mode)} />
{/if}
