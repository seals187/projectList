<script lang="ts">
  // Reporting, view "Dashboard": key figures, completed vs. created items per
  // week (line chart with hover), projects by status, due in the next 14 days.
  import { t } from '$lib/i18n.svelte';
  import { ALL_STATUSES, statusDotCls, statusDotStyle, statusLabel } from '$lib/projectStatus';
  import { dayMonth, linePoints, niceMax, pctChange, signed, type Report } from '$lib/reporting';
  import type { ProjectStatus } from '$lib/types';
  import KpiCard from './KpiCard.svelte';

  let { report, today }: { report: Report; today: string } = $props();

  const d = $derived(report.dashboard);
  const completedChange = $derived(pctChange(d.completed, d.completed_prev));
  const openChange = $derived(pctChange(d.open, d.open_prev));

  // ---- chart (viewBox units)
  const W = 600;
  const H = 170;
  const max = $derived(niceMax(Math.max(1, ...d.weeks.map((w) => Math.max(w.completed, w.created)))));
  const path = (vals: number[]) =>
    linePoints(vals, W, H, max)
      .map(([x, y], i) => `${i ? 'L' : 'M'}${x.toFixed(1)},${y.toFixed(1)}`)
      .join(' ');
  const completedPath = $derived(path(d.weeks.map((w) => w.completed)));
  const createdPath = $derived(path(d.weeks.map((w) => w.created)));
  let hover = $state<number | null>(null);
  const step = $derived(d.weeks.length > 1 ? W / (d.weeks.length - 1) : W);

  const statuses = $derived(ALL_STATUSES.filter((s) => s !== 'archived' && (d.status[s] ?? 0) > 0) as ProjectStatus[]);
  const statusTotal = $derived(statuses.reduce((s, x) => s + (d.status[x] ?? 0), 0));
</script>

<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <KpiCard
    label={t('rep.kpi.completed')}
    value={d.completed}
    note={completedChange === null ? '' : t('rep.vsPrev', { v: signed(completedChange, ' %') })}
    tone={completedChange !== null && completedChange < 0 ? 'bad' : 'good'}
    testid="kpi-completed" />
  <KpiCard label={t('rep.kpi.open')} value={d.open} note={openChange === null ? '' : signed(openChange, ' %')} tone={openChange !== null && openChange > 0 ? 'bad' : 'good'} />
  <KpiCard
    label={t('rep.kpi.overdue')}
    value={d.overdue}
    valueTone={d.overdue ? 'bad' : null}
    note={d.overdue !== d.overdue_prev ? signed(d.overdue - d.overdue_prev) : ''}
    tone={d.overdue > d.overdue_prev ? 'bad' : 'good'} />
  <KpiCard label={t('rep.kpi.projectsDone')} value={d.projects_done} note={t('rep.thisPeriod')} tone="good" />
</div>

<div class="mt-4 grid items-start gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(0,0.9fr)_minmax(0,1.2fr)]">
  <!-- completed vs. created per week -->
  <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-chart">
    <h2 class="font-semibold">{t('rep.chart.title')} <span class="text-sm font-normal text-zinc-400">{t('rep.chart.perWeek')}</span></h2>
    <div class="relative mt-3" role="img" aria-label={t('rep.chart.title')}>
      <svg viewBox="0 0 {W} {H + 26}" class="h-auto w-full overflow-visible" role="presentation" onmouseleave={() => (hover = null)}>
        {#each [0, 0.25, 0.5, 0.75, 1] as g (g)}
          <line x1="0" x2={W} y1={H * g} y2={H * g} class="stroke-zinc-200 dark:stroke-zinc-800" stroke-width="1" />
        {/each}
        {#if hover !== null}
          <line x1={hover * step} x2={hover * step} y1="0" y2={H} class="stroke-zinc-300 dark:stroke-zinc-600" stroke-width="1" />
        {/if}
        <path d={createdPath} fill="none" class="stroke-zinc-400" stroke-width="2" stroke-dasharray="6 5" stroke-linejoin="round" />
        <path d={completedPath} fill="none" stroke="var(--accent-primary)" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />
        {#each d.weeks as w, i (w.start)}
          {#if i % 2 === 0 || i === d.weeks.length - 1}
            <text x={i * step} y={H + 21} text-anchor={i === 0 ? 'start' : i === d.weeks.length - 1 ? 'end' : 'middle'} class="fill-zinc-400 text-[14px]">{w.label}</text>
          {/if}
          <!-- hover target: the whole column of the week -->
          <rect x={i * step - step / 2} y="0" width={step} height={H} fill="transparent" role="presentation" onmouseenter={() => (hover = i)} />
        {/each}
      </svg>
      {#if hover !== null}
        {@const w = d.weeks[hover]}
        <div
          class="pointer-events-none absolute top-0 z-10 rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-xs shadow-md dark:border-zinc-700 dark:bg-zinc-900"
          style:left="{Math.min(Math.max((hover / Math.max(1, d.weeks.length - 1)) * 100, 12), 82)}%"
          style:transform="translateX(-50%)">
          <p class="font-semibold">{w.label} · {dayMonth(w.start)}</p>
          <p class="flex items-center gap-1.5"><span class="h-0.5 w-3 rounded bg-[var(--accent-primary)]"></span>{t('rep.chart.completed')}: <b class="tabular-nums">{w.completed}</b></p>
          <p class="flex items-center gap-1.5"><span class="h-0 w-3 border-t-2 border-dashed border-zinc-400"></span>{t('rep.chart.created')}: <b class="tabular-nums">{w.created}</b></p>
        </div>
      {/if}
    </div>
    <div class="mt-2 flex gap-4 text-xs text-zinc-500">
      <span class="flex items-center gap-1.5"><span class="h-0.5 w-4 rounded bg-[var(--accent-primary)]"></span>{t('rep.chart.completed')}</span>
      <span class="flex items-center gap-1.5"><span class="h-0 w-4 border-t-2 border-dashed border-zinc-400"></span>{t('rep.chart.created')}</span>
    </div>
  </section>

  <!-- projects by status -->
  <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-status">
    <h2 class="font-semibold">{t('rep.status.title')}</h2>
    {#if statusTotal}
      <div class="mt-3 flex h-2.5 gap-0.5 overflow-hidden rounded-full">
        {#each statuses as s (s)}
          <span class="h-full {statusDotCls(s)}" style={statusDotStyle(s)} style:flex-grow={d.status[s]} title="{statusLabel(s)}: {d.status[s]}"></span>
        {/each}
      </div>
      <ul class="mt-3 space-y-1.5 text-sm">
        {#each statuses as s (s)}
          <li class="flex items-center gap-2">
            <span class="size-2.5 rounded-full {statusDotCls(s)}" style={statusDotStyle(s)}></span>
            <span class="flex-1">{statusLabel(s)}</span>
            <span class="font-semibold tabular-nums">{d.status[s]}</span>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="mt-3 text-sm text-zinc-400">{t('rep.none')}</p>
    {/if}
  </section>

  <!-- due in the next 14 days -->
  <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-due">
    <h2 class="font-semibold">{t('rep.due.title')}</h2>
    <ul class="mt-2 divide-y divide-zinc-100 text-sm dark:divide-zinc-800">
      {#each d.due ?? [] as x (x.kind + x.id)}
        <li>
          <a href={x.kind === 'project' ? `/projects/${x.id}` : `/checklists/${x.id}`} class="flex items-center gap-3 py-2 hover:underline">
            <span class="w-12 shrink-0 font-semibold tabular-nums {x.date < today ? 'text-red-600 dark:text-red-400' : ''}">{dayMonth(x.date)}</span>
            <span class="min-w-0 flex-1 truncate">{x.title}</span>
            <span class="max-w-[40%] shrink-0 truncate text-zinc-400">{x.customer}</span>
          </a>
        </li>
      {:else}
        <li class="py-2 text-zinc-400">{t('rep.due.none')}</li>
      {/each}
    </ul>
  </section>
</div>
