<script lang="ts">
  // "List" overview: key figures at the top, below a table grouped by customer.
  import { openCustomers, toggleCustomer } from '$lib/overviewOpen.svelte';
  import { fmtDate, t, withoutText } from '$lib/i18n.svelte';
  import { barCls, customerCode, customerTone, daysUntil, STATUS_BADGE } from '$lib/projectStatus';
  import { dueUrgent, pct, type OvCustomer } from '$lib/overview';
  import TeamAvatars from './TeamAvatars.svelte';
  import Icon from './Icon.svelte';

  let {
    groups,
    today,
    canCreate,
    oncreate
  }: { groups: OvCustomer[]; today: string; canCreate: boolean; oncreate: (customerId: number) => void } = $props();

  // Projects per customer: collapsed by default (shared with the cards view)

  const all = $derived(groups.flatMap((g) => g.projects));
  const running = $derived(all.filter((p) => p.status === 'active' || p.status === 'review' || p.status === 'blocked'));
  const openTasks = $derived(all.reduce((s, p) => s + (p.total - p.done), 0));
  const dueSoon = $derived(
    all
      .filter((p) => p.due_date && p.status !== 'done' && daysUntil(p.due_date, today) <= 14)
      .sort((a, b) => a.due_date!.localeCompare(b.due_date!))
  );
  const blocked = $derived(all.filter((p) => p.status === 'blocked'));
  const shortDate = (iso: string) => fmtDate(iso).slice(0, 5); // TT.MM / DD/MM

  // Columns: Project | Status | Checklists | Done | Due | Team | Next step
  const cols = 'grid grid-cols-[minmax(10rem,1.3fr)_7rem_minmax(12rem,1.6fr)_6.5rem_7.5rem_5.5rem_minmax(10rem,1.4fr)] items-center gap-x-4';
</script>

<div class="mb-5 grid grid-cols-2 gap-3 lg:grid-cols-4">
  <div class="card px-4 py-3">
    <p class="text-sm text-zinc-500">{t('overview.activeProjects')}</p>
    <p class="mt-0.5 text-2xl font-bold tabular-nums">{running.length} <span class="text-sm font-normal text-zinc-400">{t('overview.ofN', { n: all.length })}</span></p>
  </div>
  <div class="card px-4 py-3">
    <p class="text-sm text-zinc-500">{t('overview.openTasks')}</p>
    <p class="mt-0.5 text-2xl font-bold tabular-nums">{openTasks}</p>
  </div>
  <div class="card px-4 py-3">
    <p class="text-sm text-zinc-500">{t('overview.dueSoon')}</p>
    <p class="mt-0.5 truncate text-2xl font-bold tabular-nums">
      {dueSoon.length}
      <span class="text-sm font-normal text-zinc-400">{dueSoon.slice(0, 3).map((p) => shortDate(p.due_date!)).join(' · ')}</span>
    </p>
  </div>
  <div class="card px-4 py-3">
    <p class="text-sm text-zinc-500">{t('overview.blocked')}</p>
    <p class="mt-0.5 truncate text-2xl font-bold tabular-nums {blocked.length ? 'text-red-600 dark:text-red-400' : ''}">
      {blocked.length}
      <span class="text-sm font-normal text-zinc-400">{blocked.map((p) => p.name).join(', ')}</span>
    </p>
  </div>
</div>

<div class="overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
  <div class="min-w-[64rem]">
    <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
      <span class="pl-9">{t('overview.col.project')}</span>
      <span>{t('overview.col.status')}</span>
      <span>{t('overview.col.checklists')}</span>
      <span>{t('overview.col.done')}</span>
      <span>{t('overview.col.due')}</span>
      <span>{t('overview.col.team')}</span>
      <span>{t('overview.col.next')}</span>
    </div>

    {#each groups as g (g.customer.id)}
      {@const open = openCustomers.has(g.customer.id)}
      <div class="flex items-center gap-3 border-b border-zinc-200 px-4 py-2.5 dark:border-zinc-800">
        <button type="button" class="rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200" onclick={() => toggleCustomer(g.customer.id)} aria-expanded={open} aria-label="{t(open ? 'overview.hideProjects' : 'overview.showProjects')}: {g.customer.name}">
          <Icon name="chevronDown" size={15} class="transition-transform {open ? '' : '-rotate-90'}" />
        </button>
        <span class="flex size-7 shrink-0 items-center justify-center rounded-lg text-[11px] font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</span>
        <a href="/customers/{g.customer.id}" class="truncate font-semibold hover:underline">{g.customer.name}</a>
        <span class="shrink-0 text-sm text-zinc-400">
          {t('overview.nProjects', { n: g.projects.length })}{g.projects.length ? ` · ${t('overview.overall', { pct: pct(g.done, g.total) })}` : ''}{g.excluded ? ` · ${withoutText(g.excluded)}` : ''}
        </span>
        {#if canCreate}
          <button type="button" class="ml-auto shrink-0 text-sm font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => oncreate(g.customer.id)}>
            + {t('overview.addProject')}
          </button>
        {/if}
      </div>

      {#if open}
        {#each g.projects as p (p.id)}
          <a
            href="/projects/{p.id}"
            class="{cols} border-b border-zinc-100 px-4 py-2.5 text-sm last:border-b-0 hover:bg-zinc-50 dark:border-zinc-800/70 dark:hover:bg-zinc-800/40
              {p.status === 'blocked' ? 'bg-red-50/60 dark:bg-red-950/20' : ''}">
            <span class="truncate pl-9 font-semibold">{p.name}</span>
            <span><span class="badge {STATUS_BADGE[p.status]}">{t(`status.${p.status}`)}</span></span>
            <span class="flex min-w-0 gap-2">
              {#each p.checklists.slice(0, 3) as cl (cl.id)}
                <span class="min-w-0 flex-1" title="{cl.title}: {cl.done}/{cl.total}">
                  <span class="block truncate text-xs text-zinc-500">{cl.title}</span>
                  <span class="mt-0.5 block h-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                    <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                  </span>
                </span>
              {:else}
                <span class="text-zinc-400">–</span>
              {/each}
              {#if p.checklists.length > 3}<span class="self-end text-xs text-zinc-400">+{p.checklists.length - 3}</span>{/if}
            </span>
            <span class="tabular-nums"><span class="font-semibold">{pct(p.done, p.total)}%</span> <span class="text-xs text-zinc-400">{p.done}/{p.total}</span></span>
            <span class="flex items-center gap-1 tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-600 dark:text-zinc-300'}">
              {#if p.due_date}<Icon name="calendar" size={13} />{fmtDate(p.due_date)}{:else}<span class="text-zinc-400">–</span>{/if}
            </span>
            <span><TeamAvatars team={p.team} /></span>
            <span class="truncate">{#if p.next && p.status !== 'done'}{p.next}{:else}<span class="text-zinc-400">—</span>{/if}</span>
          </a>
        {:else}
          <p class="border-b border-zinc-100 px-4 py-2.5 pl-[3.25rem] text-sm text-zinc-400 last:border-b-0 dark:border-zinc-800/70">
            {t('overview.noProjectsYet')}{#if canCreate}{' – '}<button type="button" class="font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => oncreate(g.customer.id)}>{t('overview.createProject')}</button>{/if}
          </p>
        {/each}
      {/if}
    {/each}
  </div>
</div>
