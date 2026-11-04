<script lang="ts">
  // "Cards" overview: one card per customer with its projects.
  import { fmtDate, t, withoutText } from '$lib/i18n.svelte';
  import { ALL_STATUSES, barCls, customerCode, customerTone, STATUS_BADGE, STATUS_DOT } from '$lib/projectStatus';
  import { dueUrgent, pct, segGrow, type OvCustomer } from '$lib/overview';
  import { openCustomers, toggleCustomer } from '$lib/overviewOpen.svelte';
  import TeamAvatars from './TeamAvatars.svelte';
  import Icon from './Icon.svelte';

  let {
    groups,
    today,
    canCreate,
    oncreate
  }: { groups: OvCustomer[]; today: string; canCreate: boolean; oncreate: (customerId: number) => void } = $props();
</script>

<!-- as many columns as fit (min. 16rem each): 4–6 on wide screens -->
<div class="grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] items-start gap-4">
  {#each groups as g (g.customer.id)}
    {@const open = openCustomers.has(g.customer.id)}
    <section class="rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900/40">
      <header class="flex items-center gap-3 {open ? 'mb-3' : ''}">
        <button
          type="button"
          class="-mr-1 rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
          onclick={() => toggleCustomer(g.customer.id)}
          aria-expanded={open}
          aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
          <Icon name="chevronDown" size={16} class="transition-transform {open ? '' : '-rotate-90'}" />
        </button>
        <a href="/customers/{g.customer.id}" class="flex size-10 shrink-0 items-center justify-center rounded-xl text-sm font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</a>
        <div class="min-w-0 flex-1">
          <a href="/customers/{g.customer.id}" class="block truncate text-base font-semibold hover:underline">{g.customer.name}</a>
          <p class="text-sm text-zinc-500">{t('overview.nProjects', { n: g.projects.length })}</p>
        </div>
        {#if canCreate}
          <button type="button" class="btn-ghost px-1.5!" onclick={() => oncreate(g.customer.id)} title={t('overview.newProject')} aria-label={t('overview.newProject')}>
            <Icon name="plus" size={16} />
          </button>
        {/if}
      </header>

      {#if !open}
        <!-- collapsed: overall progress and number of projects per status -->
        {#if g.projects.length}
          <button type="button" class="mt-3 block w-full text-left" onclick={() => toggleCustomer(g.customer.id)} aria-label="{t('overview.showProjects')}: {g.customer.name}">
            <span class="block h-1.5 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
              <span class="block h-full rounded-full {barCls(g.done, g.total)}" style:width="{pct(g.done, g.total)}%"></span>
            </span>
            <span class="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-zinc-500">
              <span class="tabular-nums">{t('overview.overall', { pct: pct(g.done, g.total) })} · {g.done}/{g.total}</span>
              {#if g.excluded}<span>{withoutText(g.excluded)}</span>{/if}
              {#each ALL_STATUSES as s (s)}
                {@const n = g.projects.filter((p) => p.status === s).length}
                {#if n}<span class="flex items-center gap-1"><span class="size-1.5 rounded-full {STATUS_DOT[s]}"></span>{n} {t(`status.${s}`)}</span>{/if}
              {/each}
            </span>
          </button>
        {/if}
      {:else}
      <div class="space-y-3">
        {#each g.projects as p (p.id)}
          <a href="/projects/{p.id}" class="block rounded-xl border border-zinc-200 bg-white p-3.5 transition-shadow hover:shadow-md dark:border-zinc-800 dark:bg-zinc-900">
            <div class="flex items-start gap-2">
              <span class="line-clamp-2 min-w-0 leading-snug font-semibold break-words" title={p.name}>{p.name}</span>
              <span class="badge mt-px shrink-0 {STATUS_BADGE[p.status]}">{t(`status.${p.status}`)}</span>
              <span class="ml-auto shrink-0"><TeamAvatars team={p.team} /></span>
            </div>
            <!-- one segment per checklist, width by number of items -->
            <div class="mt-3 flex gap-1">
              {#each p.checklists as cl (cl.id)}
                <span class="h-1.5 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800" style:flex-grow={segGrow(cl)} style:flex-basis="0" title="{cl.title}: {cl.done}/{cl.total}">
                  <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{cl.total ? (cl.done / cl.total) * 100 : 0}%"></span>
                </span>
              {:else}
                <span class="h-1.5 flex-1 rounded-full bg-zinc-200 dark:bg-zinc-800"></span>
              {/each}
            </div>
            <div class="mt-2.5 flex items-center justify-between gap-2 text-sm text-zinc-500">
              <span class="tabular-nums">{t('overview.nChecklists', { n: p.checklists.length })} · {p.done}/{p.total}</span>
              {#if p.due_date}
                <span class="flex items-center gap-1 tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : ''}">
                  <Icon name="calendar" size={13} />{fmtDate(p.due_date)}
                </span>
              {/if}
            </div>
            {#if p.next && p.status !== 'done'}
              <p class="mt-1.5 truncate text-sm"><span class="text-zinc-400">{t('overview.next')}</span> {p.next}</p>
            {/if}
          </a>
        {:else}
          <div class="rounded-xl border border-dashed border-zinc-300 px-4 py-5 text-center text-sm dark:border-zinc-700">
            <p class="text-zinc-400">{t('overview.noProjectsYet')}</p>
            {#if canCreate}
              <button type="button" class="mt-0.5 font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => oncreate(g.customer.id)}>
                + {t('overview.createProject')}
              </button>
            {/if}
          </div>
        {/each}
      </div>
      {/if}
    </section>
  {/each}
</div>
