<script lang="ts">
  // Reporting, view "Customers": progress, open / overdue items and next due
  // date per customer; click on a row shows the preview of its status report
  // (PDF = print view /reports/print/{id}).
  import { session } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { dayMonth, percent, swissDate, type Report } from '$lib/reporting';
  import { barCls } from '$lib/projectStatus';
  import EntityIcon from './EntityIcon.svelte';
  import Icon from './Icon.svelte';

  let { report, today }: { report: Report; today: string } = $props();

  let selected = $state<number | null>(null);
  const sel = $derived(report.customers.find((c) => c.id === selected) ?? report.customers[0]);
  const pdfHref = (id: number) => `/reports/print/${id}?period=${report.days}`;
  const cols = 'grid grid-cols-[minmax(9rem,1.5fr)_3.5rem_minmax(7rem,1.2fr)_3rem_4.5rem_5rem_4.5rem] items-center gap-x-3';
</script>

<div class="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(17rem,1fr)]">
  <div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
    <div class="min-w-[40rem]">
      <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
        <span>{t('rep.col.customer')}</span><span>{t('rep.col.projects')}</span><span>{t('rep.col.progress')}</span>
        <span>{t('rep.col.open')}</span><span>{t('rep.col.overdue')}</span><span>{t('rep.col.nextDue')}</span><span></span>
      </div>
      {#each report.customers as c (c.id)}
        {@const pc = percent(c.done, c.total)}
        <div
          class="{cols} cursor-pointer border-b border-zinc-100 px-4 py-2.5 text-sm last:border-b-0 hover:bg-zinc-50 dark:border-zinc-800/70 dark:hover:bg-zinc-800/40 {sel?.id === c.id ? 'bg-accent-50/50 dark:bg-accent-950/20' : ''}"
          role="button"
          tabindex="0"
          onclick={() => (selected = c.id)}
          onkeydown={(e) => e.key === 'Enter' && (selected = c.id)}
          data-testid="rep-customer-row">
          <span class="flex min-w-0 items-center gap-2.5"><EntityIcon type="customer" size="sm" /><span class="truncate font-semibold">{c.name}</span></span>
          <span class="tabular-nums">{c.projects.length}</span>
          <span class="flex items-center gap-2.5">
            <span class="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800"><span class="block h-full rounded-full {barCls(c.done, c.total)}" style:width="{pc}%"></span></span>
            <span class="w-10 text-right font-semibold tabular-nums">{pc}%</span>
          </span>
          <span class="tabular-nums">{c.open}</span>
          <span class="tabular-nums {c.overdue ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-400'}">{c.overdue}</span>
          <span class="tabular-nums {c.next_due && c.next_due < today ? 'text-red-600 dark:text-red-400' : 'text-zinc-500'}">{c.next_due ? dayMonth(c.next_due) : '–'}</span>
          <a href={pdfHref(c.id)} target="_blank" class="btn-outline py-1!" onclick={(e) => e.stopPropagation()} data-testid="rep-pdf"><Icon name="download" size={14} />PDF</a>
        </div>
      {:else}
        <p class="px-4 py-4 text-sm text-zinc-400">{t('rep.none')}</p>
      {/each}
    </div>
  </div>

  {#if sel}
    {@const overdueTotal = sel.overdue}
    <section class="surface self-start rounded-xl border border-zinc-200 p-4 dark:border-zinc-800" data-testid="rep-preview">
      <h2 class="font-semibold">{t('rep.preview.title')}</h2>
      <div class="mt-3 rounded-lg border border-zinc-200 p-4 dark:border-zinc-700">
        <p class="text-[17px] font-bold">{t('rep.preview.heading', { name: sel.name })}</p>
        <p class="text-xs text-zinc-500">{swissDate(report.from)} – {swissDate(report.to)} · {t('rep.preview.by', { name: session.user?.name ?? '' })}</p>
        <p class="mt-3 border-t border-zinc-100 pt-3 text-sm dark:border-zinc-800">
          <b>{t('overview.nProjects', { n: sel.projects.length })}</b> · {t('rep.preview.done', { n: percent(sel.done, sel.total) })} · {t('rep.preview.overdue', { n: overdueTotal })}
        </p>
        <ul class="mt-2 space-y-1.5 text-sm">
          {#each sel.projects as p (p.id)}
            {@const pp = percent(p.done, p.total)}
            <li class="grid grid-cols-[minmax(0,1fr)_minmax(5rem,1fr)_2.5rem] items-center gap-3">
              <span class="truncate">{p.name}</span>
              <span class="h-1.5 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800"><span class="block h-full rounded-full {barCls(p.done, p.total)}" style:width="{pp}%"></span></span>
              <span class="text-right tabular-nums">{pp}%</span>
            </li>
          {/each}
        </ul>
        <p class="mt-3 border-t border-zinc-100 pt-2 text-xs text-zinc-400 dark:border-zinc-800">{t('rep.preview.more')}</p>
      </div>
      <a href={pdfHref(sel.id)} target="_blank" class="btn-primary mt-3"><Icon name="download" size={15} />{t('rep.preview.pdf')}</a>
    </section>
  {/if}
</div>
