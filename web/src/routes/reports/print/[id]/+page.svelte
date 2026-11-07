<script lang="ts">
  // Status report of a customer as a print page (without sidebar): the browser's
  // print dialog opens automatically → "Save as PDF". Always light, A4.
  import { page } from '$app/state';
  import { tick } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { session } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { percent, swissDate, type ClBrief, type CustomerReport } from '$lib/reporting';

  let rep = $state<CustomerReport | null>(null);
  let error = $state('');
  const id = $derived(Number(page.params.id));
  const period = $derived(Number(page.url.searchParams.get('period')) || 30);

  $effect(() => {
    api.get<CustomerReport>(`/reports/customers/${id}?period=${period}`).then(
      async (r) => {
        rep = r;
        await tick();
        if (!page.url.searchParams.has('noprint')) setTimeout(() => window.print(), 300);
      },
      (e) => (error = errorText(e))
    );
  });
  const now = new Date();
  const created = `${String(now.getDate()).padStart(2, '0')}.${String(now.getMonth() + 1).padStart(2, '0')}.${now.getFullYear()}`;
</script>

<svelte:head><title>{rep ? t('rep.preview.heading', { name: rep.customer.name }) : t('nav.reports')} · ProjectList</title></svelte:head>

{#snippet lines(title: string, list: ClBrief[], tone = '', progress = false)}
  <section class="mt-6 break-inside-avoid">
    <h2 class="border-b border-zinc-300 pb-1 text-[13px] font-bold tracking-wider text-zinc-500 uppercase">{title} <span class="font-normal">({list.length})</span></h2>
    {#if list.length}
      <table class="mt-1 w-full text-[12.5px]">
        <tbody>
          {#each list as l, i (i)}
            <tr class="border-b border-zinc-100 align-top">
              <td class="w-20 py-1 pr-3 tabular-nums {tone}">{l.date ? swissDate(l.date) : '–'}</td>
              <td class="py-1 pr-3 font-medium">{l.title}</td>
              <td class="py-1 pr-3 text-zinc-500">{l.project}</td>
              {#if progress}<td class="w-16 py-1 text-right tabular-nums">{percent(l.done, l.total)}%</td>{/if}
            </tr>
          {/each}
        </tbody>
      </table>
    {:else}
      <p class="mt-1 text-[12.5px] text-zinc-400">{t('rep.none')}</p>
    {/if}
  </section>
{/snippet}

<div class="min-h-dvh bg-zinc-100 text-zinc-900 print:bg-white">
  <div class="mx-auto flex max-w-[210mm] justify-end gap-2 px-4 pt-4 print:hidden">
    <button type="button" class="btn-ghost text-zinc-600!" onclick={() => history.back()}>{t('nav.back')}</button>
    <button type="button" class="btn-primary" onclick={() => window.print()} data-testid="print-btn">{t('rep.print')}</button>
  </div>
  <article class="mx-auto my-4 max-w-[210mm] bg-white px-[15mm] py-[14mm] shadow-sm print:my-0 print:max-w-none print:p-0 print:shadow-none" data-testid="print-report">
    {#if error}
      <p class="text-red-600">{error}</p>
    {:else if !rep}
      <p class="text-zinc-400">{t('common.loading')}</p>
    {:else}
      <header class="border-b-2 border-zinc-900 pb-3">
        <p class="text-[11px] font-semibold tracking-widest text-zinc-500 uppercase">ProjectList · {t('rep.print.kind')}</p>
        <h1 class="mt-1 text-[26px] font-bold">{t('rep.preview.heading', { name: rep.customer.name })}</h1>
        <p class="text-[13px] text-zinc-600">
          {swissDate(rep.from)} – {swissDate(rep.to)} · {t('rep.print.created', { date: created, name: session.user?.name ?? '' })}
        </p>
        {#if rep.customer.location || rep.customer.contact}
          <p class="text-[13px] text-zinc-600">{[rep.customer.location, rep.customer.contact].filter(Boolean).join(' · ')}</p>
        {/if}
      </header>

      <p class="mt-4 text-[14px]">
        <b>{t('overview.nProjects', { n: rep.projects.length })}</b> · {t('rep.preview.done', { n: percent(rep.done, rep.total) })} ·
        {t('rep.print.open', { n: rep.open.length })} · {t('rep.preview.overdue', { n: rep.overdue.length })}
      </p>

      <section class="mt-4 break-inside-avoid">
        <h2 class="border-b border-zinc-300 pb-1 text-[13px] font-bold tracking-wider text-zinc-500 uppercase">{t('rep.col.projects')}</h2>
        <table class="mt-1 w-full text-[13px]">
          <tbody>
            {#each rep.projects as p (p.id)}
              {@const pc = percent(p.done, p.total)}
              <tr class="border-b border-zinc-100">
                <td class="py-1.5 pr-3 font-medium">{p.name}</td>
                <td class="w-[45%] py-1.5 pr-3">
                  <div class="h-2 overflow-hidden rounded-full bg-zinc-200 print:border print:border-zinc-300"><div class="h-full rounded-full bg-zinc-800" style:width="{pc}%"></div></div>
                </td>
                <td class="w-40 py-1.5 text-right whitespace-nowrap tabular-nums">{pc}% · {t('rep.nLists', { done: p.done, total: p.total })}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>

      {@render lines(t('rep.print.overdue'), rep.overdue, 'font-semibold text-red-700', true)}
      {@render lines(t('rep.print.upcoming'), rep.upcoming, '', true)}
      {@render lines(t('rep.print.openLists'), rep.open, '', true)}
      {@render lines(t('rep.print.completed'), rep.completed)}

      {#if rep.customer.notes}
        <section class="mt-6 break-inside-avoid">
          <h2 class="border-b border-zinc-300 pb-1 text-[13px] font-bold tracking-wider text-zinc-500 uppercase">{t('rep.print.notes')}</h2>
          <p class="mt-1 text-[12.5px] whitespace-pre-line">{rep.customer.notes}</p>
        </section>
      {/if}
    {/if}
  </article>
</div>
