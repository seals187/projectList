<script lang="ts">
  // Overview: customers and projects as cards, status board, grouped list,
  // calendar or timeline. Filters (customer, label, person responsible) apply
  // to all views and can be saved as views (sidebar).
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
  import { buildOverview, type OvProject } from '$lib/overview';
  import { openCustomers, setAllCustomers } from '$lib/overviewOpen.svelte';
  import { activeFilters, addSavedView, BOARD_GROUPS, filter, matchesFilter, NO_OWNER, setFilter, type BoardGroup } from '$lib/overviewFilter.svelte';
  import { customerCode, customerTone, statusLabel } from '$lib/projectStatus';
  import type { OverviewExtra, ProjectStatus } from '$lib/types';
  import OverviewCards from '$lib/components/OverviewCards.svelte';
  import OverviewList from '$lib/components/OverviewList.svelte';
  import OverviewBoard from '$lib/components/OverviewBoard.svelte';
  import OverviewCalendar from '$lib/components/OverviewCalendar.svelte';
  import OverviewTimeline from '$lib/components/OverviewTimeline.svelte';
  import FilterDropdown, { type FilterItem } from '$lib/components/FilterDropdown.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import MainHeader from '$lib/components/MainHeader.svelte';

  const today = todayISO();
  let extra = $state<Record<string, OverviewExtra>>({});

  $effect(() => {
    refreshSidebar(0); // keep progress/status up to date
    api.get<Record<string, OverviewExtra>>(`/overview?today=${today}`).then(
      (r) => (extra = r),
      () => {}
    );
  });

  const groups = $derived(buildOverview(sidebar.customers, extra));
  const projects = $derived(groups.flatMap((g) => g.projects));
  const blocked = $derived(projects.filter((p) => p.status === 'blocked').length);
  const canCreate = $derived(canWrite() && sidebar.customers.length > 0);
  const dateLine = $derived(new Date().toLocaleDateString(locale(), { weekday: 'long', day: '2-digit', month: '2-digit', year: 'numeric' }));

  // ---- Filters (all views) -----------------------------------------------------
  // hidden customers; IDs of deleted customers are ignored
  const hidden = $derived(new Set(sidebar.boardHidden.filter((id) => groups.some((g) => g.customer.id === id))));
  const projectFilter = $derived(filter.tags.length > 0 || filter.owners.length > 0);
  // visible customers with their matching projects; with a label/person filter,
  // customers without a matching project are left out
  const shown = $derived(
    groups
      .filter((g) => !hidden.has(g.customer.id))
      .map((g) => ({ ...g, projects: g.projects.filter(matchesFilter) }))
      .filter((g) => !projectFilter || g.projects.length > 0)
  );

  const customerItems = $derived<FilterItem[]>(groups.map((g) => ({ id: g.customer.id, label: g.customer.name, code: { text: customerCode(g.customer.name), cls: customerTone(g.customer.id) } })));
  const tagItems = $derived<FilterItem[]>(sidebar.tags.map((tag) => ({ id: tag.id, label: tag.name, color: tag.color })));
  // people who are responsible for at least one project, plus "nobody"
  const ownerItems = $derived.by<FilterItem[]>(() => {
    const m = new Map<number, string>();
    for (const p of projects) if (p.owner_id) m.set(p.owner_id, p.owner_name);
    const list = [...m].map(([id, name]) => ({ id, label: name, avatar: true })).sort((a, b) => a.label.localeCompare(b.label));
    return [...list, { id: NO_OWNER, label: t('project.noOwner') }];
  });

  const customerValue = $derived(hidden.size ? groups.filter((g) => !hidden.has(g.customer.id)).map((g) => g.customer.id) : null);
  const setCustomers = (v: number[] | null) => setBoardHidden(v === null ? [] : groups.map((g) => g.customer.id).filter((id) => !v.includes(id)));
  function resetFilters() {
    setBoardHidden([]);
    setFilter({ tags: [], owners: [] });
  }

  // "Save view": name popover below the button
  let saveAt = $state<{ x: number; y: number } | null>(null);
  let viewName = $state('');
  async function saveView(e: SubmitEvent) {
    e.preventDefault();
    if (!viewName.trim()) return;
    await addSavedView(viewName, overviewView.value);
    viewName = '';
    saveAt = null;
  }

  // ---- Expand/collapse all (cards, list) -------------------------------------------
  const allOpen = $derived(shown.length > 0 && shown.every((g) => openCustomers.has(g.customer.id)));
  const toggleAll = () => setAllCustomers(shown.map((g) => g.customer.id), !allOpen);

  // "New project" dialog (global in the layout)
  const create = (customerId: number | null = null, status: ProjectStatus = 'planning') => openNewProject(customerId, status);

  // Board: drag & drop changes the column (status) and/or the order within it
  // (show immediately, revert on error). ids = new order of the target column.
  async function move(p: OvProject, status: ProjectStatus, ids: number[]) {
    const all = sidebar.customers.flatMap((c) => c.projects);
    const touched = all.filter((x) => ids.includes(x.id));
    const prev = touched.map((x) => ({ x, status: x.status, pos: x.board_position }));
    const moved = p.status !== status;
    for (const x of touched) {
      x.status = status;
      x.board_position = ids.indexOf(x.id);
    }
    try {
      await api.put('/board/order', { status, ids });
      if (moved) toast(t('overview.moved', { name: p.name, status: statusLabel(status) }), 'info');
    } catch (e) {
      for (const o of prev) {
        o.x.status = o.status;
        o.x.board_position = o.pos;
      }
      toast(errorText(e));
    }
  }

  const VIEW_ICON: Record<OverviewView, 'grid' | 'columns' | 'list' | 'calendar' | 'timeline'> = {
    cards: 'grid',
    board: 'columns',
    list: 'list',
    calendar: 'calendar',
    timeline: 'timeline'
  };
  const segCls = (on: boolean) =>
    `inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-sm ${on ? 'seg-on' : 'sel-off font-medium'}`;
  const view = $derived(overviewView.value);
</script>

<svelte:head><title>{t('nav.overview')} · ProjectList</title></svelte:head>

<MainHeader title={t('overview.hello', { name: session.user?.name ?? '' })}>
  {#snippet subtitle()}
    <span class="capitalize">{dateLine}</span>
    · {t('overview.nCustomers', { n: sidebar.customers.length })}
    · {t('overview.nProjects', { n: projects.length })}
    {#if blocked}· <span class="font-semibold text-red-600 dark:text-red-400">{t('overview.nBlocked', { n: blocked })}</span>{/if}
  {/snippet}
  {#snippet actions()}
    <div class="inline-flex gap-0.5 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70" role="group" aria-label={t('overview.view')}>
      {#each OVERVIEW_VIEWS as v (v)}
        <button type="button" class={segCls(view === v)} aria-pressed={view === v} onclick={() => setOverviewView(v)} title={t(`overview.view.${v}`)}>
          <Icon name={VIEW_ICON[v]} size={14} /><span class="hidden lg:inline">{t(`overview.view.${v}`)}</span>
        </button>
      {/each}
    </div>
    {#if canCreate}
      <button type="button" class="btn-primary" onclick={() => create()}><Icon name="plus" size={15} /> {t('overview.newProject')}</button>
    {/if}
  {/snippet}
  {#snippet toolbar()}
    {#if sidebar.customers.length}
      <!-- Filter bar: applies to all views -->
      <div class="mb-5 flex flex-wrap items-center gap-2">
        <Icon name="filter" size={15} class="text-zinc-400" />
        <FilterDropdown label={t('filter.customer')} items={customerItems} value={customerValue} onchange={setCustomers} none />
        <FilterDropdown label={t('filter.label')} items={tagItems} value={filter.tags.length ? filter.tags : null} onchange={(v) => setFilter({ tags: v ?? [] })} />
        <FilterDropdown label={t('filter.owner')} items={ownerItems} value={filter.owners.length ? filter.owners : null} onchange={(v) => setFilter({ owners: v ?? [] })} />
        {#if view === 'board'}
          <label class="relative">
            <select
              class="appearance-none rounded-lg border-0 bg-accent-50 py-1 pr-7 pl-2.5 text-sm font-medium text-accent-700 outline-none focus:ring-2 focus:ring-accent-500/30 dark:bg-accent-950/50 dark:text-accent-300"
              value={filter.group}
              onchange={(e) => setFilter({ group: e.currentTarget.value as BoardGroup })}
              aria-label={t('filter.groupBy')}>
              {#each BOARD_GROUPS as g (g)}<option value={g}>{t('filter.groupBy')}: {t(`filter.group.${g}`)}</option>{/each}
            </select>
            <Icon name="chevronDown" size={13} class="pointer-events-none absolute top-1/2 right-2 -translate-y-1/2 text-accent-500" />
          </label>
        {/if}
        {#if activeFilters()}
          <button type="button" class="rounded-lg px-2 py-1 text-sm text-zinc-500 hover:bg-zinc-100 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-200" onclick={resetFilters}>
            {t('filter.reset')}
          </button>
        {/if}
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-lg border border-dashed border-zinc-300 px-2.5 py-1 text-sm text-zinc-500 hover:border-zinc-400 hover:text-zinc-800 dark:border-zinc-700 dark:hover:text-zinc-200"
          onclick={(e) => {
            const r = e.currentTarget.getBoundingClientRect();
            saveAt = saveAt ? null : { x: r.left, y: r.bottom + 4 };
          }}>
          <Icon name="plus" size={13} />{t('views.save')}
        </button>

        <span class="ml-auto flex items-center gap-3">
          {#if view === 'board'}
            <!-- board: collapsed = compact cards (name and progress only) -->
            <button type="button" class="btn-ghost py-1!" onclick={() => setFilter({ compact: !filter.compact })}>
              <Icon name="chevronDown" size={15} class={filter.compact ? '' : 'rotate-180'} />
              {t(filter.compact ? 'editor.expandAll' : 'editor.collapseAll')}
            </button>
          {:else if view === 'cards' || view === 'list'}
            <button type="button" class="btn-ghost py-1!" onclick={toggleAll}>
              <Icon name="chevronDown" size={15} class={allOpen ? 'rotate-180' : ''} />
              {t(allOpen ? 'editor.collapseAll' : 'editor.expandAll')}
            </button>
          {/if}
        </span>
      </div>
    {/if}
  {/snippet}
</MainHeader>

{#if sidebar.customers.length === 0}
  <p class="py-2 text-sm text-zinc-400">
    {#if isAdmin()}<a href="/customers" class="text-accent-600 hover:underline">{t('overview.noCustomersAdmin')}</a>{:else}{t('overview.noCustomers')}{/if}
  </p>
{:else}

  {#if !shown.length}
    <p class="rounded-xl border border-dashed border-zinc-300 px-4 py-6 text-center text-sm text-zinc-500 dark:border-zinc-700">
      {t('filter.noMatch')}
      <button type="button" class="ml-1 font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={resetFilters}>{t('filter.reset')}</button>
    </p>
  {:else if view === 'list'}
    <OverviewList groups={shown} {today} {canCreate} oncreate={(id) => create(id)} />
  {:else if view === 'board'}
    <OverviewBoard all={groups} groups={shown} {today} {canCreate} oncreate={create} onmove={move} />
  {:else if view === 'calendar'}
    <OverviewCalendar groups={shown} {today} />
  {:else if view === 'timeline'}
    <OverviewTimeline groups={shown} {today} />
  {:else}
    <OverviewCards groups={shown} {today} {canCreate} oncreate={(id) => create(id)} />
  {/if}
{/if}

{#if saveAt}
  <ContextMenu x={saveAt.x} y={saveAt.y} label={t('views.save')} onclose={() => (saveAt = null)}>
    <form onsubmit={saveView} class="space-y-2 p-1.5">
      <p class="text-xs text-zinc-500">{t('views.saveHint')}</p>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input py-1.5! text-sm!" maxlength="60" placeholder={t('views.namePlaceholder')} bind:value={viewName} autofocus aria-label={t('views.name')} />
      <button type="submit" class="btn-primary w-full justify-center" disabled={!viewName.trim()}>{t('common.save')}</button>
    </form>
  </ContextMenu>
{/if}
