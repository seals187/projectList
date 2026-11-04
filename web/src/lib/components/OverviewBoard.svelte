<script lang="ts">
  // Übersicht „Board“: Spalten je Status, Projekte per Drag & Drop verschieben,
  // oben Filter nach Kunden.
  import { SvelteSet } from 'svelte/reactivity';
  import { fmtDate, t } from '$lib/i18n.svelte';
  import { barCls, BOARD_STATUSES, customerCode, customerTone, STATUS_DOT } from '$lib/projectStatus';
  import { dueUrgent, pct, type OvCustomer, type OvProject } from '$lib/overview';
  import type { ProjectStatus } from '$lib/types';
  import TeamAvatars from './TeamAvatars.svelte';
  import ProgressRing from './ProgressRing.svelte';
  import Icon from './Icon.svelte';

  let {
    groups,
    today,
    canCreate,
    oncreate,
    onmove
  }: {
    groups: OvCustomer[];
    today: string;
    canCreate: boolean;
    oncreate: (customerId: number | null, status: ProjectStatus) => void;
    onmove: (p: OvProject, status: ProjectStatus) => void;
  } = $props();

  // ausgeblendete Kunden (leer = alle sichtbar)
  const hidden = new SvelteSet<number>();
  const toggle = (id: number) => (hidden.has(id) ? hidden.delete(id) : hidden.add(id));

  const projects = $derived(groups.filter((g) => !hidden.has(g.customer.id)).flatMap((g) => g.projects));
  const columns = $derived(BOARD_STATUSES.map((s) => ({ status: s, items: projects.filter((p) => p.status === s) })));
  const shown = $derived(groups.filter((g) => !hidden.has(g.customer.id)));

  // Drag & Drop
  let dragId = $state<number | null>(null);
  let over = $state<ProjectStatus | null>(null);

  function drop(e: DragEvent, status: ProjectStatus) {
    e.preventDefault();
    const p = projects.find((x) => x.id === dragId);
    if (p && p.status !== status) onmove(p, status);
    dragId = null;
    over = null;
  }
</script>

<div class="mb-5 flex flex-wrap items-center gap-2 text-sm">
  <span class="text-zinc-500">{t('overview.customerFilter')}</span>
  {#each groups as g (g.customer.id)}
    {@const on = !hidden.has(g.customer.id)}
    <button
      type="button"
      class="inline-flex items-center gap-2 rounded-full border py-1 pr-3 pl-1.5 transition-colors
        {on ? 'border-accent-400/60 bg-accent-50 dark:border-accent-500/40 dark:bg-accent-950/40' : 'border-zinc-200 bg-white text-zinc-500 hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900'}"
      aria-pressed={on}
      onclick={() => toggle(g.customer.id)}>
      <span class="flex size-5 items-center justify-center rounded-full text-[8px] font-bold {customerTone(g.customer.id)}">{customerCode(g.customer.name)}</span>
      {g.customer.name}
    </button>
  {/each}
</div>

<!-- schmal: horizontal scrollen; breit: fünf gleich breite Spalten -->
<div class="flex min-h-[60vh] gap-3 overflow-x-auto pb-2 xl:grid xl:grid-cols-5 xl:overflow-visible">
  {#each columns as col (col.status)}
    <section
      class="flex w-72 shrink-0 flex-col rounded-2xl bg-zinc-100/70 p-2.5 transition-colors xl:w-auto xl:min-w-0 dark:bg-zinc-900/60
        {over === col.status ? 'ring-2 ring-accent-400/60' : ''}"
      aria-label={t(`status.${col.status}`)}
      ondragover={(e) => {
        if (dragId === null) return;
        e.preventDefault();
        over = col.status;
      }}
      ondragleave={(e) => {
        if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) over = null;
      }}
      ondrop={(e) => drop(e, col.status)}>
      <header class="mb-3 flex items-center gap-2 px-1">
        <span class="size-2.5 rounded-full {STATUS_DOT[col.status]}"></span>
        <h3 class="font-semibold">{t(`status.${col.status}`)}</h3>
        <span class="text-sm text-zinc-400 tabular-nums">{col.items.length}</span>
        {#if canCreate}
          <button
            type="button"
            class="ml-auto rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
            onclick={() => oncreate(shown.length === 1 ? shown[0].customer.id : null, col.status)}
            title={t('overview.newProject')}
            aria-label="{t('overview.newProject')} ({t(`status.${col.status}`)})">
            <Icon name="plus" size={16} />
          </button>
        {/if}
      </header>

      <div class="flex flex-1 flex-col gap-3">
        {#each col.items as p (p.id)}
          <a
            href="/projects/{p.id}"
            draggable={canCreate}
            ondragstart={(e) => {
              dragId = p.id;
              e.dataTransfer?.setData('text/plain', String(p.id));
              if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
            }}
            ondragend={() => {
              dragId = null;
              over = null;
            }}
            class="block rounded-xl border bg-white p-3.5 shadow-xs transition-shadow hover:shadow-md dark:bg-zinc-900
              {p.status === 'blocked' ? 'border-red-300 dark:border-red-500/50' : 'border-zinc-200 dark:border-zinc-800'}
              {dragId === p.id ? 'opacity-50' : ''}">
            <p class="flex items-center gap-2 text-sm text-zinc-500">
              <span class="flex size-5 shrink-0 items-center justify-center rounded text-[8px] font-bold {customerTone(p.customer.id)}">{customerCode(p.customer.name)}</span>
              <span class="truncate">{p.customer.name}</span>
            </p>
            <div class="mt-2 flex items-start gap-2">
              <span class="line-clamp-2 min-w-0 flex-1 text-[15px] leading-snug font-semibold break-words" title={p.name}>{p.name}</span>
              <ProgressRing done={p.done} total={p.total} />
            </div>
            {#if p.checklists.length}
              <div class="mt-2.5 space-y-2">
                {#each p.checklists as cl (cl.id)}
                  <div class="text-sm" title={cl.title}>
                    <div class="flex items-baseline gap-2">
                      <span class="min-w-0 flex-1 truncate">{cl.title}</span>
                      <span class="shrink-0 text-xs text-zinc-400 tabular-nums">{cl.done}/{cl.total}</span>
                    </div>
                    <span class="mt-1 block h-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                      <span class="block h-full rounded-full {barCls(cl.done, cl.total)}" style:width="{pct(cl.done, cl.total)}%"></span>
                    </span>
                  </div>
                {/each}
              </div>
            {/if}
            {#if p.due_date || p.team.length}
              <div class="mt-3 flex items-center justify-between border-t border-zinc-100 pt-2.5 text-sm dark:border-zinc-800">
                <span class="flex items-center gap-1 tabular-nums {dueUrgent(p, today) ? 'font-semibold text-red-600 dark:text-red-400' : 'text-zinc-500'}">
                  {#if p.due_date}<Icon name="calendar" size={13} />{fmtDate(p.due_date)}{/if}
                </span>
                <TeamAvatars team={p.team} />
              </div>
            {/if}
          </a>
        {:else}
          {#if dragId !== null}
            <p class="rounded-xl border border-dashed border-zinc-300 py-6 text-center text-sm text-zinc-400 dark:border-zinc-700">{t('overview.dropHere')}</p>
          {/if}
        {/each}
      </div>
    </section>
  {/each}
</div>
