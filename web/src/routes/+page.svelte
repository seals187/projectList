<script lang="ts">
  // Overview: customers and projects as cards, grouped list or status board.
  import { api, errorText } from '$lib/api';
  import { canWrite, isAdmin, openNewProject, OVERVIEW_VIEWS, overviewView, refreshSidebar, session, setOverviewView, sidebar, toast } from '$lib/state.svelte';
  import { locale, t, todayISO } from '$lib/i18n.svelte';
  import { buildOverview, type OvProject } from '$lib/overview';
  import { openCustomers, setAllCustomers } from '$lib/overviewOpen.svelte';
  import type { OverviewExtra, ProjectStatus } from '$lib/types';
  import OverviewCards from '$lib/components/OverviewCards.svelte';
  import OverviewList from '$lib/components/OverviewList.svelte';
  import OverviewBoard from '$lib/components/OverviewBoard.svelte';
  import Icon from '$lib/components/Icon.svelte';

  const today = todayISO();
  let extra = $state<Record<string, OverviewExtra>>({});

  $effect(() => {
    refreshSidebar(0); // keep progress/status up to date
    api.get<Record<string, OverviewExtra>>('/overview').then(
      (r) => (extra = r),
      () => {}
    );
  });

  const groups = $derived(buildOverview(sidebar.customers, extra));
  const projects = $derived(groups.flatMap((g) => g.projects));
  const blocked = $derived(projects.filter((p) => p.status === 'blocked').length);
  const allOpen = $derived(groups.length > 0 && groups.every((g) => openCustomers.has(g.customer.id)));
  const canCreate = $derived(canWrite() && sidebar.customers.length > 0);
  const dateLine = $derived(new Date().toLocaleDateString(locale(), { weekday: 'long', day: '2-digit', month: '2-digit', year: 'numeric' }));

  // "New project" dialog (global in the layout)
  const create = (customerId: number | null = null, status: ProjectStatus = 'planning') => openNewProject(customerId, status);

  // Board: change status via drag & drop (show immediately, revert on error)
  async function move(p: OvProject, status: ProjectStatus) {
    const target = sidebar.customers.flatMap((c) => c.projects).find((x) => x.id === p.id);
    if (!target) return;
    const prev = target.status;
    target.status = status;
    try {
      await api.patch(`/projects/${p.id}`, { status });
      toast(t('overview.moved', { name: p.name, status: t(`status.${status}`) }), 'info');
    } catch (e) {
      target.status = prev;
      toast(errorText(e));
    }
  }

  const segCls = (on: boolean) =>
    `rounded-md px-3 py-1 text-sm ${on ? 'bg-white font-medium shadow-xs dark:bg-zinc-700' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}`;
</script>

<svelte:head><title>{t('nav.overview')} · ProjectList</title></svelte:head>

<header class="mb-6 flex flex-wrap items-start justify-between gap-4">
  <div class="min-w-0">
    <h1 class="font-serif text-[28px] leading-tight font-bold tracking-tight">{t('overview.hello', { name: session.user?.name ?? '' })}</h1>
    <p class="mt-1 text-sm text-zinc-500">
      <span class="capitalize">{dateLine}</span>
      · {t('overview.nCustomers', { n: sidebar.customers.length })}
      · {t('overview.nProjects', { n: projects.length })}
      {#if blocked}· <span class="font-semibold text-red-600 dark:text-red-400">{t('overview.nBlocked', { n: blocked })}</span>{/if}
    </p>
  </div>
  <div class="flex flex-wrap items-center gap-3">
    {#if overviewView.value !== 'board' && groups.length}
      <button type="button" class="btn-ghost" onclick={() => setAllCustomers(groups.map((g) => g.customer.id), !allOpen)}>
        <Icon name="chevronDown" size={15} class={allOpen ? 'rotate-180' : ''} />
        {t(allOpen ? 'editor.collapseAll' : 'editor.expandAll')}
      </button>
    {/if}
    <div class="inline-flex gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70" role="group" aria-label={t('overview.view')}>
      {#each OVERVIEW_VIEWS as v (v)}
        <button type="button" class={segCls(overviewView.value === v)} aria-pressed={overviewView.value === v} onclick={() => setOverviewView(v)}>{t(`overview.view.${v}`)}</button>
      {/each}
    </div>
    {#if canCreate}
      <button type="button" class="btn-primary" onclick={() => create()}><Icon name="plus" size={15} /> {t('overview.newProject')}</button>
    {/if}
  </div>
</header>

{#if sidebar.customers.length === 0}
  <p class="py-2 text-sm text-zinc-400">
    {#if isAdmin()}<a href="/customers" class="text-accent-600 hover:underline">{t('overview.noCustomersAdmin')}</a>{:else}{t('overview.noCustomers')}{/if}
  </p>
{:else if overviewView.value === 'list'}
  <OverviewList {groups} {today} {canCreate} oncreate={(id) => create(id)} />
{:else if overviewView.value === 'board'}
  <OverviewBoard {groups} {today} {canCreate} oncreate={create} onmove={move} />
{:else}
  <OverviewCards {groups} {today} {canCreate} oncreate={(id) => create(id)} />
{/if}
