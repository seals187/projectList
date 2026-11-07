<script lang="ts">
  // Reporting, view "Workload": open checklists per person (bar: open / due this
  // week / overdue), overdue and done in the period; open checklists without a person.
  import { t } from '$lib/i18n.svelte';
  import { dayMonth, pctChange, signed, type Report } from '$lib/reporting';
  import KpiCard from './KpiCard.svelte';
  import Avatar from './Avatar.svelte';

  let { report, today }: { report: Report; today: string } = $props();

  const people = $derived(report.workload.people.filter((p) => p.open > 0 || p.done > 0));
  const openAssigned = $derived(people.reduce((s, p) => s + p.open, 0));
  const overdueTotal = $derived(people.reduce((s, p) => s + p.overdue, 0));
  const overdueNames = $derived(people.filter((p) => p.overdue > 0).map((p) => p.name).join(', '));
  const doneTotal = $derived(people.reduce((s, p) => s + p.done, 0));
  const doneChange = $derived(pctChange(doneTotal, report.workload.done_prev));
  const withOpen = $derived(people.filter((p) => p.open > 0).length);
  const maxOpen = $derived(Math.max(1, ...people.map((p) => p.open)));
  const un = $derived(report.workload.unassigned);
  const cols = 'grid grid-cols-[minmax(9rem,1.2fr)_minmax(8rem,1.6fr)_3.5rem_6rem_6rem] items-center gap-x-4';
</script>

<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <KpiCard label={t('rep.kpi.openAssigned')} value={openAssigned} note={t('rep.people', { n: withOpen })} tone="good" />
  <KpiCard label={t('rep.kpi.overdue')} value={overdueTotal} valueTone={overdueTotal ? 'bad' : null} note={overdueNames} tone="bad" />
  <KpiCard label={t('rep.kpi.donePeriod', { n: report.days })} value={doneTotal} note={doneChange === null ? '' : signed(doneChange, ' %')} tone={doneChange !== null && doneChange < 0 ? 'bad' : 'good'} />
  <KpiCard label={t('rep.kpi.avgPerson')} value={withOpen ? Math.round(openAssigned / withOpen) : 0} note={t('rep.openItems')} />
</div>

<div class="mt-4 grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
  <div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
    <div class="min-w-[40rem]">
      <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
        <span>{t('rep.col.assignee')}</span><span>{t('rep.col.openItems')}</span><span>{t('rep.col.open')}</span><span>{t('rep.col.overdue')}</span>
        <span>{t('rep.col.done', { n: report.days })}</span>
      </div>
      {#each people as p (p.id)}
        {@const rest = p.open - p.due_week - p.overdue}
        <div class="{cols} border-b border-zinc-100 px-4 py-2.5 text-sm dark:border-zinc-800/70" data-testid="rep-person-row">
          <span class="flex min-w-0 items-center gap-2.5"><Avatar name={p.name} id={p.id} size={28} /><span class="truncate font-semibold">{p.name}</span></span>
          <span class="flex h-2.5" style:width="{(p.open / maxOpen) * 100}%" title="{t('rep.legend.open')}: {rest} · {t('rep.legend.week')}: {p.due_week} · {t('rep.legend.overdue')}: {p.overdue}">
            {#if rest}<span class="h-full rounded-full border-r-2 border-white bg-[var(--accent-primary)] dark:border-zinc-900" style:flex-grow={rest}></span>{/if}
            {#if p.due_week}<span class="h-full rounded-full border-r-2 border-white bg-amber-500 dark:border-zinc-900" style:flex-grow={p.due_week}></span>{/if}
            {#if p.overdue}<span class="h-full rounded-full bg-red-600" style:flex-grow={p.overdue}></span>{/if}
          </span>
          <span class="font-semibold tabular-nums">{p.open}</span>
          <span class={p.overdue ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-400'}>{t('rep.nOverdue', { n: p.overdue })}</span>
          <span class="text-emerald-700 dark:text-emerald-400">{t('rep.nDone', { n: p.done })}</span>
        </div>
      {:else}
        <p class="px-4 py-4 text-sm text-zinc-400">{t('rep.none')}</p>
      {/each}
      <div class="flex gap-4 px-4 py-2.5 text-xs text-zinc-500">
        <span class="flex items-center gap-1.5"><span class="size-2.5 rounded-sm bg-[var(--accent-primary)]"></span>{t('rep.legend.open')}</span>
        <span class="flex items-center gap-1.5"><span class="size-2.5 rounded-sm bg-amber-500"></span>{t('rep.legend.week')}</span>
        <span class="flex items-center gap-1.5"><span class="size-2.5 rounded-sm bg-red-600"></span>{t('rep.legend.overdue')}</span>
      </div>
    </div>
  </div>

  <section class="surface self-start rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-unassigned">
    <h2 class="font-semibold">{t('rep.unassigned.title')}</h2>
    <p class="mt-1 text-[32px] leading-tight font-bold tabular-nums">{un.total}</p>
    <p class="text-sm text-zinc-500">{t('rep.unassigned.hint', { n: un.checklists.length })}</p>
    <ul class="mt-2 divide-y divide-zinc-100 text-sm dark:divide-zinc-800">
      {#each un.checklists.slice(0, 8) as c (c.id)}
        <li>
          <a href="/checklists/{c.id}" class="flex items-center gap-3 py-2 hover:underline">
            <span class="min-w-0 flex-1">
              <span class="block truncate">{c.title}</span>
              <span class="block truncate text-xs text-zinc-400">{c.project} · {c.customer}</span>
            </span>
            <span class="shrink-0 text-xs tabular-nums {c.date && c.date < today ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-500'}">{c.date ? dayMonth(c.date) : '–'}</span>
          </a>
        </li>
      {/each}
    </ul>
  </section>
</div>
