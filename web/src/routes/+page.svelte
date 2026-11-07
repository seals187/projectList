<script lang="ts">
  // Overview: customers and projects as cards, status board, grouped list,
  // calendar or timeline. Filters (customer, label, person responsible) apply
  // to all views and can be saved as views (sidebar). Header: PageHeader.
  import { api, errorText } from '$lib/api';
  import {
    canWrite,
    isAdmin,
    openNewProject,
    OVERVIEW_VIEWS,
    overviewView,
    refreshSidebar,
    session,
    setBoardHidden,
    setOverviewView,
    sidebar,
    toast,
    type OverviewView
  } from '$lib/state.svelte';
  import { locale, t, todayISO } from '$lib/i18n.svelte';
  import { buildOverview, OVERVIEW_SORTS, searchOverview, sortOverview, type OverviewSort, type OvProject } from '$lib/overview';
  import { openCustomers, setAllCustomers } from '$lib/overviewOpen.svelte';
  import { addSavedView, BOARD_GROUPS, filter, matchesFilter, NO_OWNER, resetFilters, setFilter, type BoardGroup } from '$lib/overviewFilter.svelte';
  import { allCollapsed } from '$lib/pageStateCore';
  import { pageState } from '$lib/pageState.svelte';
  import { displayLabel, displayOf, setDisplay } from '$lib/display.svelte';
  import { statusLabel } from '$lib/projectStatus';
  import type { OverviewExtra, ProjectStatus } from '$lib/types';
  import OverviewCards from '$lib/components/OverviewCards.svelte';
  import OverviewList from '$lib/components/OverviewList.svelte';
  import OverviewBoard from '$lib/components/OverviewBoard.svelte';
  import OverviewCalendar from '$lib/components/OverviewCalendar.svelte';
  import OverviewTimeline from '$lib/components/OverviewTimeline.svelte';
  import PageHeader, { headerDate, type HeaderFilter } from '$lib/components/PageHeader.svelte';
  import ZoomArea from '$lib/components/ZoomArea.svelte';
  import type { FilterId, FilterItem } from '$lib/components/FilterDropdown.svelte';

  const today = todayISO();
  let extra = $state<Record<string, OverviewExtra>>({});

  $effect(() => {
    refreshSidebar(0); // keep progress/status up to date
    api.get<Record<string, OverviewExtra>>(`/overview?today=${today}`).then(
      (r) => (extra = r),
      () => {}
    );
  });

  // Header state: search and sorting per user and browser; view and filters as
  // before (filters synced in the database). Everything is mirrored into the URL.
  const ids = (v: unknown) => (Array.isArray(v) ? v.map(Number).filter(Number.isInteger) : []);
  const ps = pageState(
    'overview',
    { q: { def: '', url: 'str' }, sort: { def: 'activity' as string, url: 'str' } },
    {
      view: {
        def: 'cards',
        url: 'str',
        keep: true,
        get: () => overviewView.value,
        set: (v: string) => OVERVIEW_VIEWS.includes(v as OverviewView) && setOverviewView(v as OverviewView)
      },
      tags: { def: [], url: 'ids', get: () => filter.tags, set: (v: number[]) => setFilter({ tags: ids(v) }) },
      owners: { def: [], url: 'ids', get: () => filter.owners, set: (v: number[]) => setFilter({ owners: ids(v) }) },
      hide: { def: [], url: 'ids', get: () => sidebar.boardHidden, set: (v: number[]) => setBoardHidden(ids(v)) },
      group: { def: 'status', url: 'str', get: () => filter.group, set: (v: string) => BOARD_GROUPS.includes(v as BoardGroup) && setFilter({ group: v as BoardGroup }) }
    }
  );
  const sort = $derived<OverviewSort>(OVERVIEW_SORTS.includes(ps.sort as OverviewSort) ? (ps.sort as OverviewSort) : 'activity');

  const groups = $derived(buildOverview(sidebar.customers, extra));
  const projects = $derived(groups.flatMap((g) => g.projects));
  const blocked = $derived(projects.filter((p) => p.status === 'blocked').length);
  const canCreate = $derived(canWrite() && sidebar.customers.length > 0);

  // ---- Filters (all views) -----------------------------------------------------
  // hidden customers; IDs of deleted customers are ignored
  const hidden = $derived(new Set(sidebar.boardHidden.filter((id) => groups.some((g) => g.customer.id === id))));
  const projectFilter = $derived(filter.tags.length > 0 || filter.owners.length > 0);
  // visible customers with their matching projects; with a label/person filter,
  // customers without a matching project are left out; then search and sorting
  const shown = $derived(
    sortOverview(
      searchOverview(
        groups
          .filter((g) => !hidden.has(g.customer.id))
          .map((g) => ({ ...g, projects: g.projects.filter(matchesFilter) }))
          .filter((g) => !projectFilter || g.projects.length > 0),
        ps.q
      ),
      sort
    )
  );

  const customerItems = $derived<FilterItem[]>(
    groups.map((g) => ({ id: g.customer.id, label: g.customer.name, entity: 'customer' as const }))
  );
  const tagItems = $derived<FilterItem[]>(sidebar.tags.map((tag) => ({ id: tag.id, label: tag.name, color: tag.color })));
  // people who are responsible for at least one project, plus "nobody"
  const ownerItems = $derived.by<FilterItem[]>(() => {
    const m = new Map<number, string>();
    for (const p of projects) if (p.owner_id) m.set(p.owner_id, p.owner_name);
    const list = [...m].map(([id, name]) => ({ id, label: name, avatar: true })).sort((a, b) => a.label.localeCompare(b.label));
    return [...list, { id: NO_OWNER, label: t('project.noOwner') }];
  });

  const customerValue = $derived(hidden.size ? groups.filter((g) => !hidden.has(g.customer.id)).map((g) => g.customer.id) : null);
  const setCustomers = (v: FilterId[] | null) => setBoardHidden(v === null ? [] : groups.map((g) => g.customer.id).filter((id) => !v.includes(id)));

  const view = $derived(overviewView.value);
  const filters = $derived.by<HeaderFilter[]>(() => {
    if (!sidebar.customers.length) return [];
    const list: HeaderFilter[] = [
      { key: 'customer', label: t('filter.customer'), items: customerItems, value: customerValue, onchange: setCustomers, none: true },
      { key: 'label', label: t('filter.label'), items: tagItems, value: filter.tags.length ? filter.tags : null, onchange: (v: FilterId[] | null) => setFilter({ tags: ids(v ?? []) }) },
      { key: 'owner', label: t('filter.owner'), items: ownerItems, value: filter.owners.length ? filter.owners : null, onchange: (v: FilterId[] | null) => setFilter({ owners: ids(v ?? []) }) }
    ];
    if (view === 'board')
      list.push({
        key: 'group',
        label: t('filter.groupBy'),
        options: BOARD_GROUPS.map((g) => ({ value: g, label: t(`filter.group.${g}`) })),
        value: filter.group,
        defaultValue: 'status',
        onchange: (g: string) => setFilter({ group: g as BoardGroup })
      });
    return list;
  });

  // "Save view": filters as before plus search/sorting
  function saveView(name: string) {
    const q = new URLSearchParams();
    if (ps.q) q.set('q', ps.q);
    if (sort !== 'activity') q.set('sort', sort);
    return addSavedView(name, overviewView.value, q.toString());
  }
  function resetAll() {
    resetFilters();
    ps.q = '';
  }

  // ---- Expand/collapse all: customers (cards, list), compact cards (board) -----------
  const collapsible = $derived(view === 'cards' || view === 'list' || view === 'board');
  const collapsed = $derived(view === 'board' ? filter.compact : allCollapsed(shown.map((g) => openCustomers.has(g.customer.id))));
  function toggleCollapse() {
    if (view === 'board') setFilter({ compact: !filter.compact });
    else setAllCustomers(shown.map((g) => g.customer.id), collapsed);
  }

  // "New project" dialog (global in the layout)
  const create = (customerId: number | null = null, status: ProjectStatus = 'planning') => openNewProject(customerId, status);

  // Board: drag & drop changes the column (status) and/or the order within it
  // (show immediately, revert on error). ids = new order of the target column.
  async function move(p: OvProject, status: ProjectStatus, order: number[]) {
    const all = sidebar.customers.flatMap((c) => c.projects);
    const touched = all.filter((x) => order.includes(x.id));
    const prev = touched.map((x) => ({ x, status: x.status, pos: x.board_position }));
    const moved = p.status !== status;
    for (const x of touched) {
      x.status = status;
      x.board_position = order.indexOf(x.id);
    }
    try {
      await api.put('/board/order', { status, ids: order });
      if (moved) toast(t('overview.moved', { name: p.name, status: statusLabel(status) }), 'info');
    } catch (e) {
      for (const o of prev) {
        o.x.status = o.status;
        o.x.board_position = o.pos;
      }
      toast(errorText(e));
    }
  }

  // ---- "Display" options per view (per user: server, fallback localStorage) ---------
  const DISPLAY: Record<OverviewView, Record<string, boolean>> = {
    list: { steps: true, percent: true, allTags: false, due: true, owner: true },
    board: { tags: true, percent: true, due: true, owner: true },
    cards: { tags: true, due: true, owner: true },
    calendar: {},
    timeline: {}
  };
  const disp = $derived(displayOf(`overview.${view}`, DISPLAY[view]));
  const allSteps = $derived(shown.reduce((s, g) => ({ done: s.done + g.projects.reduce((n, p) => n + p.done, 0), total: s.total + g.projects.reduce((n, p) => n + p.total, 0) }), { done: 0, total: 0 }));
  const displayOptions = $derived(
    Object.keys(DISPLAY[view]).map((k) => ({
      key: k,
      label: k === 'steps' ? `${t('disp.steps')} (${allSteps.done}/${allSteps.total})` : displayLabel(k),
      value: disp[k]
    }))
  );

  const VIEW_ICON: Record<OverviewView, 'grid' | 'columns' | 'list' | 'calendar' | 'timeline'> = {
    cards: 'grid',
    board: 'columns',
    list: 'list',
    calendar: 'calendar',
    timeline: 'timeline'
  };
</script>

<svelte:head><title>{t('nav.overview')} · ProjectList</title></svelte:head>

<PageHeader
  zoomPage="overview"
  icon="home"
  title={t('overview.hello', { name: session.user?.name ?? '' })}
  views={OVERVIEW_VIEWS.map((v) => ({ key: v, label: t(`overview.view.${v}`), icon: VIEW_ICON[v] }))}
  activeView={view}
  onViewChange={(v) => setOverviewView(v as OverviewView)}
  primaryAction={canCreate ? { label: t('ph.addProject'), onClick: () => create() } : undefined}
  searchPlaceholder={t('ph.search.overview')}
  searchValue={ps.q}
  onSearch={(q) => (ps.q = q)}
  {filters}
  onSaveView={saveView}
  sortOptions={OVERVIEW_SORTS.map((s) => ({ value: s, label: t(`sort.${s}`) }))}
  activeSort={sort}
  onSortChange={(s) => (ps.sort = s)}
  {collapsed}
  onToggleCollapse={toggleCollapse}
  collapseDisabled={!collapsible}
  {displayOptions}
  onDisplayChange={(k, v) => setDisplay(`overview.${view}`, k, v)}>
  {#snippet subtitle()}
    {headerDate(new Date(), locale())}
    · {t('overview.nCustomers', { n: sidebar.customers.length })}
    · {t('overview.nProjects', { n: projects.length })}
    {#if blocked}· <span class="font-semibold text-red-600 dark:text-red-400">{t('overview.nBlocked', { n: blocked })}</span>{/if}
  {/snippet}
</PageHeader>

<ZoomArea page="overview">
{#if sidebar.customers.length === 0}
  <p class="py-2 text-sm text-zinc-400">
    {#if isAdmin()}<a href="/customers" class="text-accent-600 hover:underline">{t('overview.noCustomersAdmin')}</a>{:else}{t('overview.noCustomers')}{/if}
  </p>
{:else if !shown.length}
  <p class="rounded-xl border border-dashed border-zinc-300 px-4 py-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
    {t('filter.noMatch')}
    <button type="button" class="ml-1 font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={resetAll}>{t('filter.reset')}</button>
  </p>
{:else if view === 'list'}
  <OverviewList groups={shown} {today} {canCreate} oncreate={(id) => create(id)} display={disp} />
{:else if view === 'board'}
  <OverviewBoard all={groups} groups={shown} {today} {canCreate} oncreate={create} onmove={move} display={disp} />
{:else if view === 'calendar'}
  <OverviewCalendar groups={shown} {today} />
{:else if view === 'timeline'}
  <OverviewTimeline groups={shown} {today} />
{:else}
  <OverviewCards groups={shown} {today} {canCreate} oncreate={(id) => create(id)} display={disp} />
{/if}
</ZoomArea>
