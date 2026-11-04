<script lang="ts">
  // Kundenübersicht: Liste (aufklappbar mit Projekten) oder Karten, Suche,
  // Filter (alle / mit aktiven Projekten / kritisch / ohne Projekt), Sortierung.
  import { SvelteSet } from 'svelte/reactivity';
  import { api } from '$lib/api';
  import { canWrite, isAdmin, openNewProject, prefs, refreshSidebar, setPref, sidebar } from '$lib/state.svelte';
  import { fmtDate, t, todayISO, withoutText } from '$lib/i18n.svelte';
  import { barCls, customerCode, customerTone, STATUS_BADGE } from '$lib/projectStatus';
  import { dueUrgent, pct, segGrow } from '$lib/overview';
  import {
    buildCustomerRows,
    CUSTOMER_FILTERS,
    CUSTOMER_SORTS,
    CUSTOMER_STATUS_BADGE,
    matchesFilter,
    sortRows,
    type CustomerFilter,
    type CustomerRow
  } from '$lib/customers';
  import type { CustomerListItem } from '$lib/types';
  import NewCustomerDialog from '$lib/components/NewCustomerDialog.svelte';
  import ProgressToggle from '$lib/components/ProgressToggle.svelte';
  import Icon from '$lib/components/Icon.svelte';

  const today = todayISO();
  let infos = $state<CustomerListItem[]>([]);
  let query = $state('');
  let filter = $state<CustomerFilter>('all');
  let adding = $state(false);
  let menuFor = $state<number | null>(null);
  const expanded = new SvelteSet<number>();

  $effect(() => {
    refreshSidebar(0);
    api.get<CustomerListItem[]>('/customers').then(
      (r) => (infos = r),
      () => {}
    );
  });

  const rows = $derived(buildCustomerRows(sidebar.customers, infos, today));
  const counts = $derived(Object.fromEntries(CUSTOMER_FILTERS.map((f) => [f, rows.filter((r) => matchesFilter(r, f)).length])) as Record<CustomerFilter, number>);
  const visible = $derived.by(() => {
    const q = query.trim().toLowerCase();
    const hit = (r: CustomerRow) =>
      !q || [r.customer.name, r.info?.location, r.info?.contact_name, r.info?.contact_email].some((s) => s?.toLowerCase().includes(q));
    return sortRows(rows.filter((r) => matchesFilter(r, filter) && hit(r)), prefs.customersSort);
  });
  const totals = $derived({
    customers: rows.length,
    projects: rows.reduce((s, r) => s + r.projects.length, 0),
    open: rows.reduce((s, r) => s + r.open, 0)
  });

  const toggle = (id: number) => (expanded.has(id) ? expanded.delete(id) : expanded.add(id));
  const segCls = (on: boolean) =>
    `inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-sm ${on ? 'bg-white font-medium shadow-xs dark:bg-zinc-700' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}`;
  const chipCls = (on: boolean) =>
    `rounded-md border px-2.5 py-1 text-xs font-medium ${on ? 'border-zinc-900 bg-zinc-900 text-white dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900' : 'border-zinc-200 bg-white text-zinc-600 hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300'}`;

  // Spalten: Kunde | Ansprechpartner | Projekte | Offen | Fortschritt | Nächste Fälligkeit | Status | Menü
  const cols = 'grid grid-cols-[minmax(14rem,1.6fr)_minmax(11rem,1.3fr)_5.5rem_4.5rem_9.5rem_8.5rem_7.5rem_2rem] items-center gap-x-4';
</script>

<svelte:window onclick={() => (menuFor = null)} />
<svelte:head><title>{t('nav.customers')} · ProjectList</title></svelte:head>

<header class="mb-5 flex flex-wrap items-start justify-between gap-4">
  <div>
    <h1 class="font-serif text-[28px] leading-tight font-bold tracking-tight">{t('nav.customers')}</h1>
    <p class="mt-1 text-sm text-zinc-500">
      {t('overview.nCustomers', { n: totals.customers })} · {t('overview.nProjects', { n: totals.projects })} · {t('customers.nOpen', { n: totals.open })}
    </p>
  </div>
  <div class="flex items-center gap-3">
    <div class="inline-flex gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70" role="group" aria-label={t('overview.view')}>
      <button type="button" class={segCls(prefs.customersView === 'cards')} aria-pressed={prefs.customersView === 'cards'} onclick={() => setPref('customersView', 'cards')}>
        <Icon name="grid" size={14} />{t('overview.view.cards')}
      </button>
      <button type="button" class={segCls(prefs.customersView === 'list')} aria-pressed={prefs.customersView === 'list'} onclick={() => setPref('customersView', 'list')}>
        <Icon name="list" size={14} />{t('overview.view.list')}
      </button>
    </div>
    {#if isAdmin()}
      <button type="button" class="btn-primary" onclick={() => (adding = true)}><Icon name="plus" size={15} /> {t('customers.add')}</button>
    {/if}
  </div>
</header>

<!-- Suche, Filter, Sortierung -->
<div class="mb-4 flex flex-wrap items-center gap-2">
  <label class="relative w-full sm:w-72">
    <Icon name="search" size={14} class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-zinc-400" />
    <input class="input pl-8!" type="search" placeholder={t('customers.search')} bind:value={query} aria-label={t('customers.search')} />
  </label>
  {#each CUSTOMER_FILTERS as f (f)}
    <button type="button" class={chipCls(filter === f)} aria-pressed={filter === f} onclick={() => (filter = f)}>
      {t(`customers.filter.${f}`)} <span class="ml-0.5 tabular-nums opacity-70">{counts[f]}</span>
    </button>
  {/each}
  <label class="relative ml-auto">
    <select
      class="appearance-none rounded-md border border-zinc-200 bg-white py-1 pr-7 pl-2.5 text-xs text-zinc-600 outline-none hover:border-zinc-300 focus:border-accent-500 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300"
      value={prefs.customersSort}
      onchange={(e) => setPref('customersSort', e.currentTarget.value as (typeof CUSTOMER_SORTS)[number])}
      aria-label={t('customers.sortBy', { label: '' })}>
      {#each CUSTOMER_SORTS as s (s)}<option value={s}>{t('customers.sortBy', { label: t(`customers.sort.${s}`) })}</option>{/each}
    </select>
    <Icon name="chevronDown" size={13} class="pointer-events-none absolute top-1/2 right-2 -translate-y-1/2 text-zinc-400" />
  </label>
</div>

{#snippet statusBadge(r: CustomerRow)}
  <span class="badge inline-flex items-center gap-1.5 {CUSTOMER_STATUS_BADGE[r.status].badge}">
    <span class="size-1.5 rounded-full {CUSTOMER_STATUS_BADGE[r.status].dot}"></span>{t(`customers.status.${r.status}`)}
  </span>
{/snippet}

{#snippet progressBar(done: number, total: number, width = 'w-[4.5rem]')}
  {#if total}
    <span class="flex items-center gap-2">
      <span class="h-1.5 {width} overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
        <span class="block h-full rounded-full {barCls(done, total)}" style:width="{pct(done, total)}%"></span>
      </span>
      <span class="text-xs text-zinc-500 tabular-nums">{pct(done, total)}%</span>
    </span>
  {:else}
    <span class="text-zinc-400">–</span>
  {/if}
{/snippet}

{#snippet dueDate(iso: string | null, urgent: boolean)}
  {#if iso}
    <span class="flex items-center gap-1 text-[13px] tabular-nums {urgent ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-600 dark:text-zinc-300'}">
      <Icon name="calendar" size={13} />{fmtDate(iso)}
    </span>
  {:else}
    <span class="text-zinc-400">–</span>
  {/if}
{/snippet}

{#snippet rowMenu(r: CustomerRow)}
  <div class="relative">
    <button
      type="button"
      class="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
      onclick={(e) => {
        e.stopPropagation();
        menuFor = menuFor === r.customer.id ? null : r.customer.id;
      }}
      aria-label="{t('customers.actions')}: {r.customer.name}"
      aria-expanded={menuFor === r.customer.id}>
      <Icon name="dots" size={16} />
    </button>
    {#if menuFor === r.customer.id}
      <div class="absolute top-full right-0 z-20 mt-1 w-44 rounded-lg border border-zinc-200 bg-white p-1 text-sm shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
        <a href="/customers/{r.customer.id}" class="block rounded px-2.5 py-1.5 hover:bg-zinc-100 dark:hover:bg-zinc-800">{t('customers.open')}</a>
        {#if canWrite()}
          <button type="button" class="block w-full rounded px-2.5 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800" onclick={() => openNewProject(r.customer.id)}>
            {t('overview.newProject')}
          </button>
        {/if}
      </div>
    {/if}
  </div>
{/snippet}

{#if rows.length === 0}
  <p class="py-4 text-sm text-zinc-400">{isAdmin() ? t('overview.noCustomersAdmin') : t('overview.noCustomers')}</p>
{:else if visible.length === 0}
  <p class="py-4 text-sm text-zinc-400">{t('customers.noMatch')}</p>
{:else if prefs.customersView === 'cards'}
  <!-- Karten -->
  <div class="grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] gap-4">
    {#each visible as r (r.customer.id)}
      <a href="/customers/{r.customer.id}" class="card block p-4 transition-shadow hover:shadow-md">
        <div class="flex items-start gap-3">
          <span class="flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-bold {customerTone(r.customer.id)}">{customerCode(r.customer.name)}</span>
          <div class="min-w-0 flex-1">
            <p class="line-clamp-2 leading-snug font-semibold break-words" title={r.customer.name}>{r.customer.name}</p>
            <p class="truncate text-xs text-zinc-500">{r.info?.location || '–'}</p>
          </div>
          {@render statusBadge(r)}
        </div>
        {#if r.info?.contact_name || r.info?.contact_email}
          <p class="mt-3 truncate text-sm">{r.info?.contact_name}</p>
          <p class="truncate text-xs text-zinc-400">{r.info?.contact_email}</p>
        {/if}
        <div class="mt-3 flex items-center gap-4 text-sm">
          <span><span class="font-semibold tabular-nums">{r.projects.length}</span> <span class="text-zinc-500">{t('customers.col.projects')}</span></span>
          <span><span class="font-semibold tabular-nums">{r.open}</span> <span class="text-zinc-500">{t('customers.col.open')}</span></span>
        </div>
        <div class="mt-3 flex items-center justify-between gap-2">
          <div>
            {@render progressBar(r.done, r.total, 'w-24')}
            {#if r.excluded}<span class="block text-[11px] text-zinc-400">{withoutText(r.excluded)}</span>{/if}
          </div>
          {@render dueDate(r.nextDue, r.nextDueUrgent)}
        </div>
      </a>
    {/each}
  </div>
{:else}
  <!-- Liste -->
  <div class="overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
    <div class="min-w-[60rem]">
      <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
        <span class="pl-7">{t('customers.col.customer')}</span>
        <span>{t('customers.col.contact')}</span>
        <span>{t('customers.col.projects')}</span>
        <span>{t('customers.col.open')}</span>
        <span>{t('customers.col.progress')}</span>
        <span>{t('customers.col.due')}</span>
        <span>{t('customers.col.status')}</span>
        <span></span>
      </div>

      {#each visible as r (r.customer.id)}
        {@const open = expanded.has(r.customer.id)}
        <div class="border-b border-zinc-100 last:border-b-0 dark:border-zinc-800/70 {open ? 'bg-zinc-50/80 dark:bg-zinc-900/50' : ''}">
          <div class="{cols} px-4 py-3 text-sm {open ? '' : 'hover:bg-zinc-50 dark:hover:bg-zinc-900/40'}">
            <div class="flex min-w-0 items-center gap-2.5">
              <button
                type="button"
                class="rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
                onclick={() => toggle(r.customer.id)}
                aria-expanded={open}
                aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {r.customer.name}">
                <Icon name="chevron" size={14} class="transition-transform {open ? 'rotate-90' : ''}" />
              </button>
              <span class="flex size-8 shrink-0 items-center justify-center rounded-lg text-[11px] font-bold {customerTone(r.customer.id)}">{customerCode(r.customer.name)}</span>
              <a href="/customers/{r.customer.id}" class="min-w-0">
                <span class="block truncate font-semibold hover:underline">{r.customer.name}</span>
                <span class="block truncate text-xs text-zinc-500">{r.info?.location || ''}</span>
              </a>
            </div>
            <div class="min-w-0">
              <span class="block truncate">{r.info?.contact_name || ''}</span>
              {#if r.info?.contact_email}
                <a href="mailto:{r.info.contact_email}" class="block truncate text-xs text-zinc-500 hover:underline">{r.info.contact_email}</a>
              {/if}
              {#if !r.info?.contact_name && !r.info?.contact_email}<span class="text-zinc-400">–</span>{/if}
            </div>
            <span class="font-semibold tabular-nums">{r.projects.length}</span>
            <span class="font-semibold tabular-nums">{r.open}</span>
            <div>
              {@render progressBar(r.done, r.total)}
              {#if r.excluded}<span class="block text-[11px] text-zinc-400">{withoutText(r.excluded)}</span>{/if}
            </div>
            {@render dueDate(r.nextDue, r.nextDueUrgent)}
            <span>{@render statusBadge(r)}</span>
            {@render rowMenu(r)}
          </div>

          {#if open}
            <div class="pb-2">
              {#each r.projects as p (p.id)}
                <a href="/projects/{p.id}" class="{cols} px-4 py-1.5 text-sm hover:bg-zinc-100/70 dark:hover:bg-zinc-800/50">
                  <span class="flex min-w-0 items-center gap-2 pl-[3.25rem]">
                    <Icon name="folder" size={14} class="text-zinc-400" />
                    <span class="truncate font-medium {p.include_in_customer_progress ? '' : 'text-zinc-400 dark:text-zinc-500'}">{p.name}</span>
                  </span>
                  <span class="text-xs text-zinc-500">{t('overview.nChecklists', { n: p.checklists.length })}</span>
                  <span><span class="badge {STATUS_BADGE[p.status]}">{t(`status.${p.status}`)}</span></span>
                  <span class="tabular-nums">{p.total - p.done}</span>
                  <span class="flex gap-1">
                    {#each p.checklists as cl (cl.id)}
                      <span class="h-1.5 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800" style:flex-grow={segGrow(cl)} style:flex-basis="0" title="{cl.title}: {cl.done}/{cl.total}">
                        <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                      </span>
                    {:else}
                      <span class="h-1.5 flex-1 rounded-full bg-zinc-200 dark:bg-zinc-800"></span>
                    {/each}
                  </span>
                  {@render dueDate(p.due_date, dueUrgent(p, today))}
                  <span class="text-xs text-zinc-500 tabular-nums">{pct(p.done, p.total)}%</span>
                  <span class="-my-1 justify-self-end"><ProgressToggle project={p} customerName={r.customer.name} /></span>
                </a>
              {/each}
              {#if canWrite()}
                <button
                  type="button"
                  class="ml-4 py-1.5 pl-[3.25rem] text-sm font-medium text-accent-600 hover:underline dark:text-accent-400"
                  onclick={() => openNewProject(r.customer.id)}>
                  + {t('customers.addProject')}
                </button>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  </div>
{/if}

{#if adding}
  <NewCustomerDialog onclose={() => (adding = false)} />
{/if}
