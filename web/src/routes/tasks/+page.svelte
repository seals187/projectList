<script lang="ts">
  // My tasks:
  //  - "Customers": customers → projects I am responsible for or in which something
  //    (checklist or item) is assigned to me → their checklists; with customer
  //    progress and a toggle per project for whether it counts toward the customer progress.
  //  - "Personal": items assigned to me to check off.
  //  Header (PageHeader): search, filters customer / due date, chips open / today / done,
  //  sorting, expand/collapse; state per user in localStorage and in the URL.
  import { api, errorText } from '$lib/api';
  import { canWrite, openNewProject, refreshSidebar, session, sidebar, toast } from '$lib/state.svelte';
  import { locale, t, todayISO, withoutText } from '$lib/i18n.svelte';
  import { barCls, customerLabel, customerProgress, daysUntil, statusBadgeCls, statusBadgeStyle, statusLabel } from '$lib/projectStatus';
  import { pct } from '$lib/overview';
  import { addPageView } from '$lib/overviewFilter.svelte';
  import { allCollapsed } from '$lib/pageStateCore';
  import { pageState } from '$lib/pageState.svelte';
  import { displayLabel, displayOf, setDisplay } from '$lib/display.svelte';
  import type { OverviewExtra, SidebarChecklist, SidebarCustomer, SidebarProject, Task } from '$lib/types';
  import PageHeader, { headerDate, type HeaderFilter } from '$lib/components/PageHeader.svelte';
  import type { FilterId } from '$lib/components/FilterDropdown.svelte';
  import TaskList from '$lib/components/TaskList.svelte';
  import ProgressToggle from '$lib/components/ProgressToggle.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import EntityIcon from '$lib/components/EntityIcon.svelte';
  import { customerWide, projectColor, tintStyle } from '$lib/cardColors';
  import { myColors } from '$lib/myColors.svelte';
  import CustomerMenu from '$lib/components/CustomerMenu.svelte';
  import ProjectMenu from '$lib/components/ProjectMenu.svelte';
  import ChecklistMenu from '$lib/components/ChecklistMenu.svelte';
  import { page } from '$app/state';

  type Tab = 'customers' | 'personal';
  type Filter = 'open' | 'today' | 'done';
  const FILTERS: Filter[] = ['open', 'today', 'done'];
  type Due = 'overdue' | 'today' | 'week' | 'later' | 'none';
  const DUES: Due[] = ['overdue', 'today', 'week', 'later', 'none'];
  type Sort = 'due' | 'name' | 'progress';
  const SORTS: Sort[] = ['due', 'name', 'progress'];

  const today = todayISO();
  const ps = pageState('tasks', {
    view: { def: 'customers' as string, url: 'str' },
    chip: { def: 'open' as string, url: 'str' },
    q: { def: '', url: 'str' },
    cust: { def: null as FilterId[] | null, url: 'ids' },
    due: { def: null as FilterId[] | null, url: 'strs' },
    sort: { def: 'due' as string, url: 'str' },
    closed: { def: [] as number[] }, // collapsed customers (default: expanded)
    openP: { def: [] as number[] } // expanded projects (default: collapsed)
  });
  const tab = $derived<Tab>(ps.view === 'personal' ? 'personal' : 'customers');
  const filter = $derived<Filter>(FILTERS.includes(ps.chip as Filter) ? (ps.chip as Filter) : 'open');
  const sort = $derived<Sort>(SORTS.includes(ps.sort as Sort) ? (ps.sort as Sort) : 'due');
  const closedSet = $derived(new Set(ps.closed));
  const openSet = $derived(new Set(ps.openP));
  const toggleIn = (list: number[], id: number) => (list.includes(id) ? list.filter((x) => x !== id) : [...list, id]);
  const toggleClosed = (id: number) => (ps.closed = toggleIn(ps.closed, id));
  const toggleProject = (id: number) => (ps.openP = toggleIn(ps.openP, id));

  let extra = $state<Record<string, OverviewExtra>>({});
  let openTasks = $state<Task[] | null>(null);
  let doneTasks = $state<Task[] | null>(null);

  $effect(() => {
    refreshSidebar(0);
    api.get<Record<string, OverviewExtra>>('/overview').then(
      (r) => (extra = r),
      () => {}
    );
    api.get<Task[]>('/my-tasks').then(
      (r) => (openTasks = r),
      (e) => toast(errorText(e))
    );
  });
  $effect(() => {
    if (filter === 'done' && doneTasks === null)
      api.get<Task[]>('/my-tasks?done=1').then(
        (r) => (doneTasks = r),
        (e) => toast(errorText(e))
      );
  });

  // ---- Due date buckets (filter "Due") -----------------------------------------------
  function bucket(d: string | null | undefined): Due {
    if (!d) return 'none';
    const n = daysUntil(d, today);
    return n < 0 ? 'overdue' : n === 0 ? 'today' : n <= 6 ? 'week' : 'later';
  }
  /** Earliest due date of a project: its open checklists or the project itself. */
  const projectDue = (p: SidebarProject) =>
    [...p.checklists.filter((c) => clOpen(c) && c.next_due).map((c) => c.next_due), p.status !== 'done' ? (p.due_date ?? '') : ''].filter(Boolean).sort()[0] ?? '';
  const dueOk = (d: string | null | undefined) => ps.due === null || ps.due.includes(bucket(d));
  const custOk = (id: number) => ps.cust === null || ps.cust.includes(id);
  const hit = (...s: (string | undefined | null)[]) => {
    const q = ps.q.trim().toLowerCase();
    return !q || s.some((x) => x?.toLowerCase().includes(q));
  };

  // ---- Customer view -----------------------------------------------------------
  // mine: I am responsible for the project, or something in it is assigned to me
  const isOwner = (p: SidebarProject) => !!session.user && p.owner_id === session.user.id;
  const isMine = (p: SidebarProject) => isOwner(p) || !!extra[p.id]?.team.some((m) => m.id === session.user?.id);
  const clOpen = (cl: SidebarChecklist) => cl.total === 0 || cl.done < cl.total;
  const clDueToday = (cl: SidebarChecklist) => !!cl.next_due && cl.next_due <= today;
  function matches(p: SidebarProject): boolean {
    if (filter === 'done') return p.status === 'done' || (p.total > 0 && p.done === p.total);
    if (p.status === 'done') return false;
    if (filter === 'today') return p.checklists.some(clDueToday);
    return p.total === 0 || p.done < p.total || p.checklists.some(clOpen);
  }

  interface Group {
    customer: SidebarCustomer;
    projects: SidebarProject[];
    checklists: number;
  }
  const byName = (a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: 'base' });
  const ratio = (p: { done: number; total: number }) => (p.total ? p.done / p.total : -1);
  const projCmp: Record<Sort, (a: SidebarProject, b: SidebarProject) => number> = {
    due: (a, b) => (projectDue(a) || '9999').localeCompare(projectDue(b) || '9999') || byName(a.name, b.name),
    name: (a, b) => byName(a.name, b.name),
    progress: (a, b) => ratio(b) - ratio(a) || byName(a.name, b.name)
  };
  // customers with my projects (before the header filters: customer filter entries)
  const mine = $derived(
    sidebar.customers
      .map((c) => ({ customer: c, projects: c.projects.filter((p) => p.status !== 'archived' && isMine(p) && matches(p)) }))
      .filter((g) => g.projects.length > 0)
  );
  const groups = $derived.by((): Group[] => {
    const list = mine
      .filter((g) => custOk(g.customer.id))
      .map((g) => {
        const projects = g.projects
          .filter((p) => dueOk(projectDue(p)) && (hit(g.customer.name, p.name) || p.checklists.some((c) => hit(c.title))))
          .sort(projCmp[sort]);
        return { customer: g.customer, projects, checklists: projects.reduce((s, p) => s + p.checklists.length, 0) };
      })
      .filter((g) => g.projects.length > 0);
    const first = (g: Group) => g.projects[0];
    return list.sort((a, b) => projCmp[sort](first(a), first(b)) || byName(a.customer.name, b.customer.name));
  });

  // ---- Personal ------------------------------------------------------------------
  let personal = $state<Task[]>([]);
  const personalSrc = $derived.by(() => {
    const src = filter === 'done' ? (doneTasks ?? []) : (openTasks ?? []);
    return filter === 'today' ? src.filter((x) => x.due_date && x.due_date <= today) : src;
  });
  $effect(() => {
    const list = personalSrc.filter(
      (x) => custOk(x.customer_id) && dueOk(x.due_date) && hit(x.title, x.project_name, x.customer_name, x.checklist_title)
    );
    if (sort === 'due') list.sort((a, b) => (a.due_date ?? '9999').localeCompare(b.due_date ?? '9999'));
    if (sort === 'name') list.sort((a, b) => byName(a.title, b.title));
    personal = list;
  });
  const personalCount = $derived(personalSrc.length);

  // ---- Header ----------------------------------------------------------------------
  const filters = $derived<HeaderFilter[]>([
    {
      key: 'cust',
      label: t('filter.customer'),
      items: (tab === 'personal' ? sidebar.customers.filter((c) => personalSrc.some((x) => x.customer_id === c.id)) : mine.map((g) => g.customer)).map((c) => ({
        id: c.id,
        label: c.name,
        entity: 'customer' as const
      })),
      value: ps.cust,
      onchange: (v: FilterId[] | null) => (ps.cust = v)
    },
    { key: 'due', label: t('filter.due'), items: DUES.map((d) => ({ id: d, label: t(`filter.due.${d}`) })), value: ps.due, onchange: (v: FilterId[] | null) => (ps.due = v) }
  ]);
  const assigned = $derived(groups.reduce((s, g) => s + g.checklists, 0));
  // "Display" options (per user and view)
  const DISPLAY = { percent: true, due: true };
  const disp = $derived(displayOf(`tasks.${tab}`, DISPLAY));
  const collapsed = $derived(tab === 'customers' && allCollapsed(groups.map((g) => !closedSet.has(g.customer.id))));
  function toggleCollapse() {
    if (collapsed) {
      ps.closed = [];
      ps.openP = groups.flatMap((g) => g.projects.map((p) => p.id));
    } else {
      ps.closed = groups.map((g) => g.customer.id);
      ps.openP = [];
    }
  }

  // ---- Display ---------------------------------------------------------------------
  const shortDate = (iso: string) => new Date(iso + 'T00:00:00').toLocaleDateString(locale(), { day: '2-digit', month: '2-digit' });
  function dueLabel(iso: string): { text: string; cls: string } {
    const d = daysUntil(iso, today);
    if (d < -1) return { text: shortDate(iso), cls: 'font-semibold text-red-600 dark:text-red-400' };
    if (d === -1) return { text: t('due.yesterday'), cls: 'font-semibold text-red-600 dark:text-red-400' };
    if (d === 0) return { text: t('due.today'), cls: 'font-semibold text-accent-600 dark:text-accent-400' };
    if (d === 1) return { text: t('due.tomorrow'), cls: 'text-zinc-600 dark:text-zinc-300' };
    return { text: shortDate(iso), cls: 'text-zinc-500' };
  }

  // ---- Context menus (customer, project, checklist) ------------------------------------
  type Menu =
    | { t: 'customer'; customer: SidebarCustomer }
    | { t: 'project'; customer: SidebarCustomer; project: SidebarProject }
    | { t: 'checklist'; project: SidebarProject; checklist: SidebarChecklist };
  let menu = $state<(Menu & { x: number; y: number }) | null>(null);
  function openMenu(e: MouseEvent, m: Menu) {
    e.preventDefault();
    e.stopPropagation();
    menu = { ...m, x: e.clientX, y: e.clientY };
  }

  // Columns of a project/checklist row: arrow | name | progress | due | toggle | link
  const rowCols = 'grid grid-cols-[1.5rem_minmax(0,1fr)_11rem_4rem_2rem_1.25rem] items-center gap-x-3';
</script>

<svelte:head><title>{t('nav.myTasks')} · ProjectList</title></svelte:head>

<div>
  <PageHeader
    icon="listChecks"
    title={t('nav.myTasks')}
    subtitle="{headerDate(new Date(), locale())} · {t('tasks.assigned', { n: assigned })}"
    views={[
      { key: 'customers', label: t('tasks.tab.customers'), icon: 'users', count: groups.length },
      { key: 'personal', label: t('tasks.tab.personal'), icon: 'user', count: personalCount }
    ]}
    activeView={tab}
    onViewChange={(v) => (ps.view = v)}
    searchPlaceholder={t('ph.search.tasks')}
    searchValue={ps.q}
    onSearch={(q) => (ps.q = q)}
    {filters}
    quickFilters={FILTERS.map((f) => ({ key: f, label: t(`tasks.filter.${f}`), active: filter === f }))}
    onQuickFilter={(k) => (ps.chip = k)}
    onSaveView={(name) => addPageView(name, 'tasks', page.url.search.slice(1))}
    sortOptions={SORTS.map((s) => ({ value: s, label: t(`sort.${s}`) }))}
    activeSort={sort}
    onSortChange={(s) => (ps.sort = s)}
    {collapsed}
    onToggleCollapse={toggleCollapse}
    collapseDisabled={tab !== 'customers' || !groups.length}
    displayOptions={Object.keys(DISPLAY).map((k) => ({ key: k, label: displayLabel(k), value: disp[k] }))}
    onDisplayChange={(k, v) => setDisplay(`tasks.${tab}`, k, v)} />

  {#if tab === 'customers'}
    <div class="no-glass space-y-4 rounded-2xl bg-zinc-100/70 p-3 in-[.has-bg]:bg-transparent in-[.has-bg]:p-0 dark:bg-zinc-900/50">
      {#each groups as g (g.customer.id)}
        {@const cust = customerProgress(g.customer.projects)}
        {@const open = !closedSet.has(g.customer.id)}
        {@const tint = customerWide(g.customer)}
        <section class="rounded-xl border border-zinc-200 bg-white px-4 py-3 shadow-xs dark:border-zinc-800 dark:bg-zinc-900 {tint ? 'card-tint' : ''}" style={tintStyle(tint)}>
          <header class="flex items-center gap-3" role="group" aria-label={g.customer.name} oncontextmenu={(e) => openMenu(e, { t: 'customer', customer: g.customer })}>
            <EntityIcon type="customer" size="lg" />
            <div class="min-w-0 flex-1">
              <a href="/customers/{g.customer.id}" class="block truncate text-[15px] font-semibold hover:underline">{customerLabel(g.customer.name)}</a>
              <p class="text-xs text-zinc-500">{t('tasks.counts', { p: g.projects.length, c: g.checklists })}</p>
            </div>
            <div class="hidden w-40 text-right sm:block">
              <div class="flex items-center justify-end gap-3">
                <span class="h-1.5 w-24 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                  <span class="block h-full rounded-full {barCls(cust.done, cust.total)}" style:width="{pct(cust.done, cust.total)}%"></span>
                </span>
                <span class="w-10 text-sm font-semibold tabular-nums">{pct(cust.done, cust.total)}%</span>
              </div>
              {#if cust.excluded}<p class="mt-0.5 text-[11px] text-zinc-400">{withoutText(cust.excluded)}</p>{/if}
            </div>
            <button
              type="button"
              class="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
              onclick={() => toggleClosed(g.customer.id)}
              aria-expanded={open}
              aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
              <Icon name="chevronDown" size={16} class="transition-transform {open ? '' : '-rotate-90'}" />
            </button>
          </header>

          {#if open}
            <ul class="mt-2 divide-y divide-zinc-100 dark:divide-zinc-800">
              {#each g.projects as p (p.id)}
                {@const pOpen = openSet.has(p.id)}
                {@const muted = !p.include_in_customer_progress}
                <li class="py-1">
                  <!-- project row: click expands the checklists -->
                  <div
                    class="{rowCols} cursor-pointer rounded-lg px-1 py-1.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    class:card-tint={!!projectColor(p.id, myColors.all)}
            style={tintStyle(projectColor(p.id, myColors.all))}
                    role="button"
                    tabindex="0"
                    aria-expanded={pOpen}
                    aria-label="{t('tasks.showChecklists')}: {p.name}"
                    onclick={() => toggleProject(p.id)}
                    oncontextmenu={(e) => openMenu(e, { t: 'project', customer: g.customer, project: p })}
                    data-testid="task-project-row"
                    onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), toggleProject(p.id))}>
                    <span class="flex size-6 items-center justify-center rounded text-zinc-400 {pOpen ? 'bg-zinc-100 dark:bg-zinc-800' : ''}">
                      <Icon name="chevron" size={13} class="transition-transform {pOpen ? 'rotate-90' : ''}" />
                    </span>
                    <span class="flex min-w-0 items-center gap-2">
                      <EntityIcon type="project" size="sm" />
                      <span class="truncate font-semibold {muted ? 'text-zinc-400 dark:text-zinc-500' : ''}">{p.name}</span>
                      <span class="badge shrink-0 {statusBadgeCls(p.status)}" style={statusBadgeStyle(p.status)}>{statusLabel(p.status)}</span>
                      {#if isOwner(p)}
                        <span class="inline-flex shrink-0 items-center gap-1 rounded-full bg-accent-50 px-2 py-px text-[11px] font-medium text-accent-700 dark:bg-accent-950/60 dark:text-accent-400" title={t('tasks.ownerHint')}>
                          <Icon name="user" size={11} />{t('project.owner')}
                        </span>
                      {/if}
                      <span class="hidden shrink-0 text-xs text-zinc-400 md:inline">{t('overview.nChecklists', { n: p.checklists.length })}</span>
                    </span>
                    <span class="flex items-center gap-2.5 {muted ? 'opacity-50 grayscale' : ''}">
                      <span class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                        <span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pct(p.done, p.total)}%"></span>
                      </span>
                      <span class="w-9 text-right text-xs text-zinc-500 tabular-nums">{disp.percent ? `${pct(p.done, p.total)}%` : ''}</span>
                    </span>
                    <span class="text-right text-xs text-zinc-500 tabular-nums">{disp.due && p.due_date ? shortDate(p.due_date) : ''}</span>
                    <ProgressToggle project={p} customerName={g.customer.name} />
                    <a href="/projects/{p.id}" class="text-zinc-300 hover:text-zinc-700 dark:text-zinc-600 dark:hover:text-zinc-200" onclick={(e) => e.stopPropagation()} aria-label="{t('tasks.openProject')}: {p.name}" title={t('tasks.openProject')}>
                      <Icon name="chevron" size={14} />
                    </a>
                  </div>

                  {#if pOpen}
                    <ul class="mb-1 ml-9 space-y-0.5">
                      {#each p.checklists as cl (cl.id)}
                        {@const due = cl.next_due && clOpen(cl) ? dueLabel(cl.next_due) : null}
                        <li>
                          <a href="/checklists/{cl.id}" oncontextmenu={(e) => openMenu(e, { t: 'checklist', project: p, checklist: cl })} data-testid="task-checklist-row" class="{rowCols} rounded-lg px-1 py-1.5 text-sm hover:bg-accent-50/60 dark:hover:bg-accent-950/30">
                            <EntityIcon type="checklist" size="xs" class="justify-self-center" />
                            <span class="truncate">{cl.title}</span>
                            <span class="flex items-center gap-2.5">
                              <span class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                                <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                              </span>
                              <span class="w-9 text-right text-xs text-zinc-500 tabular-nums">{cl.done}/{cl.total}</span>
                            </span>
                            <span class="text-right text-xs tabular-nums {due?.cls ?? ''}">{disp.due ? (due?.text ?? '') : ''}</span>
                            <span></span>
                            <span class="text-zinc-300 dark:text-zinc-600"><Icon name="chevron" size={14} /></span>
                          </a>
                        </li>
                      {/each}
                    </ul>
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}
        </section>
      {:else}
        <p class="px-2 py-6 text-center text-sm text-zinc-400">{t('tasks.noProjects')}</p>
      {/each}
    </div>
  {:else}
    <div class="rounded-2xl border border-zinc-200 bg-white px-4 py-3 dark:border-zinc-800 dark:bg-zinc-900">
      {#if (filter === 'done' ? doneTasks : openTasks) === null}
        <p class="py-4 text-sm text-zinc-400">{t('common.loading')}</p>
      {:else if personal.length === 0}
        <p class="py-6 text-center text-sm text-zinc-400">{t('tasks.noPersonal')}</p>
      {:else}
        <TaskList bind:tasks={personal} />
      {/if}
    </div>
  {/if}
</div>

{#if menu}
  {@const m = menu}
  {#if m.t === 'customer'}
    <CustomerMenu customer={m.customer} x={m.x} y={m.y} canCreate={canWrite()} onclose={() => (menu = null)} oncreate={() => openNewProject(m.customer.id)} />
  {:else if m.t === 'project'}
    <ProjectMenu project={m.project} customer={m.customer} x={m.x} y={m.y} onclose={() => (menu = null)} />
  {:else}
    <ChecklistMenu checklist={m.checklist} project={m.project} x={m.x} y={m.y} onclose={() => (menu = null)} />
  {/if}
{/if}
