<script lang="ts">
  // Reporting, view "Templates": applications, average duration of the checklists
  // (creation → completed), planned duration (→ due date), punctuality; checklists
  // that took longest; automations in the period. Durations need the creation
  // time, which is recorded since version 0.1.7 (older checklists: from templates only).
  import { t } from '$lib/i18n.svelte';
  import { categoryHex } from '$lib/templateLibrary';
  import { signed, type Report } from '$lib/reporting';
  import type { TemplateCategory } from '$lib/types';
  import KpiCard from './KpiCard.svelte';
  import Icon from './Icon.svelte';

  let { report, categories }: { report: Report; categories: TemplateCategory[] } = $props();

  const tp = $derived(report.templates);
  const days = (v: number | null) => (v === null ? '–' : t('rep.days', { n: v.toLocaleString(undefined, { maximumFractionDigits: 1 }) }));
  const diff = (cur: number | null, prev: number | null, unit: string) => (cur === null || prev === null ? '' : signed(Math.round((cur - prev) * 10) / 10, unit));
  const mostUsed = $derived(tp.rows.find((r) => r.used > 0));
  const cols = 'grid grid-cols-[minmax(10rem,1.8fr)_5.5rem_6rem_5.5rem_minmax(8rem,1.2fr)] items-center gap-x-4';
</script>

<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <KpiCard label={t('rep.kpi.checklistsCreated')} value={tp.created} note={t('rep.fromTemplates')} tone="good" />
  <KpiCard label={t('rep.kpi.avgDuration')} value={days(tp.avg_days)} note={diff(tp.avg_days, tp.avg_days_prev, ' d')} tone={tp.avg_days !== null && tp.avg_days_prev !== null && tp.avg_days > tp.avg_days_prev ? 'bad' : 'good'} />
  <KpiCard label={t('rep.kpi.onTime')} value={tp.on_time === null ? '–' : `${Math.round(tp.on_time)} %`} note={diff(tp.on_time, tp.on_time_prev, ' %')} tone={tp.on_time !== null && tp.on_time_prev !== null && tp.on_time < tp.on_time_prev ? 'bad' : 'good'} />
  <KpiCard label={t('rep.kpi.mostUsed')} value={mostUsed?.name ?? '–'} note={mostUsed ? `${mostUsed.used}×` : ''} />
</div>

<div class="mt-4 grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">
  <div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
    <div class="min-w-[40rem]">
      <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
        <span>{t('lib.col.template')}</span><span>{t('lib.col.used')}</span><span>{t('rep.col.avgDuration')}</span><span>{t('rep.col.planned')}</span><span>{t('rep.col.onTime')}</span>
      </div>
      {#each tp.rows as r (r.id)}
        <div class="{cols} border-b border-zinc-100 px-4 py-2.5 text-sm last:border-b-0 dark:border-zinc-800/70" data-testid="rep-template-row">
          <a href="/templates/{r.id}" class="flex min-w-0 items-center gap-3 hover:underline">
            <span class="tpl-icon" style:--c={categoryHex(categories.find((c) => c.id === r.category_id))}><Icon name="fileText" size={16} /></span>
            <span class="truncate font-semibold">{r.name}</span>
          </a>
          <span class="font-semibold tabular-nums">{r.used}×</span>
          <span class="tabular-nums">{days(r.avg_days)}</span>
          <span class="tabular-nums {r.planned === null || r.avg_days === null ? '' : r.avg_days <= r.planned ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'}">{days(r.planned)}</span>
          <span class="flex items-center gap-2.5">
            {#if r.on_time !== null}
              <span class="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800"><span class="block h-full rounded-full {r.on_time >= 90 ? 'bg-emerald-500' : 'bg-[var(--accent-primary)]'}" style:width="{r.on_time}%"></span></span>
              <span class="w-10 text-right font-semibold tabular-nums">{Math.round(r.on_time)}%</span>
            {:else}
              <span class="text-zinc-400">–</span>
            {/if}
          </span>
        </div>
      {:else}
        <p class="px-4 py-4 text-sm text-zinc-400">{t('rep.tpl.none')}</p>
      {/each}
    </div>
  </div>

  <div class="space-y-4 self-start">
    <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-longest">
      <h2 class="font-semibold">{t('rep.longest.title')} <span class="text-sm font-normal text-zinc-400">{t('rep.longest.sub')}</span></h2>
      <ul class="mt-2 divide-y divide-zinc-100 text-sm dark:divide-zinc-800">
        {#each tp.longest as l (l.id)}
          <li>
            <a href="/checklists/{l.id}" class="flex items-center gap-3 py-2 hover:underline">
              <span class="min-w-0 flex-1">
                <span class="block truncate">{l.title}</span>
                <span class="block truncate text-xs text-zinc-400">{l.project} · {l.customer}</span>
              </span>
              <span class="font-semibold tabular-nums">{days(l.days ?? null)}</span>
            </a>
          </li>
        {:else}
          <li class="py-2 text-zinc-400">{t('rep.longest.none')}</li>
        {/each}
      </ul>
    </section>
    <section class="surface rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-automation">
      <h2 class="font-semibold">{t('rep.auto.title')}</h2>
      <div class="mt-2 grid grid-cols-2 gap-4">
        <div><p class="text-[28px] leading-tight font-bold tabular-nums">{tp.auto_created}</p><p class="text-sm text-zinc-500">{t('rep.auto.created')}</p></div>
        <div><p class="text-[28px] leading-tight font-bold tabular-nums {tp.auto_warned ? 'text-amber-600' : ''}">{tp.auto_warned}</p><p class="text-sm text-zinc-500">{t('rep.auto.warned')}</p></div>
      </div>
    </section>
  </div>
</div>
<p class="mt-3 text-xs text-zinc-500 in-[.has-bg]:text-white/80">{t('rep.tpl.hint')}</p>
