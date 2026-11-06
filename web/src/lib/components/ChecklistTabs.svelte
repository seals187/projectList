<script lang="ts">
  // Open checklists as tabs above the checklist page: type icon, title,
  // progress (done/total), red dot for overdue items, ✕ (or middle click) to
  // close. Opening a checklist adds its tab next to the active one; tabs can be
  // reordered by drag & drop; "+" searches all checklists to open another one.
  import { goto } from '$app/navigation';
  import { untrack } from 'svelte';
  import { t, todayISO } from '$lib/i18n.svelte';
  import { sidebar } from '$lib/state.svelte';
  import { closeTab, moveTab, openTab, pruneTabs } from '$lib/checklistTabs';
  import { checklistTabs, loadTabs, setTabs } from '$lib/checklistTabs.svelte';
  import ContextMenu from './ContextMenu.svelte';
  import EntityIcon from './EntityIcon.svelte';
  import Icon from './Icon.svelte';

  let { activeId, onImage = false }: { activeId: number; onImage?: boolean } = $props();

  // checklists by id (title, progress, project) from the sidebar data
  const info = $derived.by(() => {
    const m = new Map<number, { title: string; done: number; total: number; nextDue: string; projectId: number; project: string; customer: string }>();
    for (const c of sidebar.customers)
      for (const p of c.projects)
        for (const cl of p.checklists) m.set(cl.id, { title: cl.title, done: cl.done, total: cl.total, nextDue: cl.next_due, projectId: p.id, project: p.name, customer: c.name });
    return m;
  });
  const today = todayISO();

  // open the active checklist as a tab (after the one shown before)
  let prev: number | null = null;
  $effect(() => {
    loadTabs();
    const id = activeId;
    // only depends on the active checklist (closing tabs must not re-open it)
    untrack(() => setTabs(openTab(checklistTabs.ids, id, prev)));
    prev = id;
  });
  // deleted / no longer visible checklists disappear (once the data is there)
  $effect(() => {
    if (!sidebar.loaded) return;
    const kept = pruneTabs(checklistTabs.ids, (id) => id === activeId || info.has(id));
    if (kept.length !== checklistTabs.ids.length) setTabs(kept);
  });

  function close(e: Event, id: number) {
    e.preventDefault();
    e.stopPropagation();
    const r = closeTab(checklistTabs.ids, id);
    setTabs(r.ids);
    if (id !== activeId) return;
    if (r.next != null) goto(`/checklists/${r.next}`);
    else {
      const pid = info.get(id)?.projectId;
      goto(pid ? `/projects/${pid}` : '/');
    }
  }

  // ---- drag & drop ------------------------------------------------------------------
  let drag = $state<number | null>(null);
  let over = $state<{ id: number | null } | null>(null);
  function dragOver(e: DragEvent, id: number | null) {
    if (drag == null) return;
    e.preventDefault();
    if (id == null) return void (over = { id: null });
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    if (e.clientX < r.left + r.width / 2) over = { id };
    else {
      const i = checklistTabs.ids.indexOf(id);
      over = { id: checklistTabs.ids[i + 1] ?? null };
    }
  }
  function dropTab(e: DragEvent) {
    e.preventDefault();
    if (drag != null && over) setTabs(moveTab(checklistTabs.ids, drag, over.id));
    drag = null;
    over = null;
  }

  // ---- "+": open another checklist ----------------------------------------------------
  let picker = $state<{ x: number; y: number } | null>(null);
  let query = $state('');
  const matches = $derived.by(() => {
    const q = query.trim().toLocaleLowerCase();
    return [...info.entries()]
      .filter(([id, c]) => !checklistTabs.ids.includes(id) && (!q || `${c.title} ${c.project} ${c.customer}`.toLocaleLowerCase().includes(q)))
      .slice(0, 12);
  });
  // after the menu has been moved to <body> (moving a focused element drops the focus)
  const focus = (el: HTMLInputElement) => void requestAnimationFrame(() => el.focus());

  const tabCls = (on: boolean) =>
    on
      ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-900 dark:text-zinc-100'
      : onImage
        ? 'bg-white/55 text-zinc-700 hover:bg-white/80 dark:bg-zinc-900/55 dark:text-zinc-200 dark:hover:bg-zinc-900/80'
        : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200/70 dark:bg-zinc-800/60 dark:text-zinc-300 dark:hover:bg-zinc-800';
</script>

<div class="flex items-center gap-1.5 overflow-x-auto pb-1" role="tablist" aria-label={t('tabs.label')} data-testid="checklist-tabs">
  {#each checklistTabs.ids as id (id)}
    {@const c = info.get(id)}
    {@const on = id === activeId}
    {#if c || on}
      <a
        href="/checklists/{id}"
        role="tab"
        aria-selected={on}
        draggable="true"
        class="group relative flex h-9 max-w-64 min-w-36 shrink-0 items-center gap-2 rounded-[10px] pr-1.5 pl-2.5 text-[13px] transition-colors {tabCls(on)} {drag === id ? 'opacity-40' : ''}"
        title={c ? `${c.title} · ${c.customer} › ${c.project}` : ''}
        ondragstart={(e) => {
          drag = id;
          e.dataTransfer?.setData('text/plain', String(id));
        }}
        ondragover={(e) => dragOver(e, id)}
        ondrop={dropTab}
        ondragend={() => ((drag = null), (over = null))}
        onauxclick={(e) => e.button === 1 && close(e, id)}
        data-testid="checklist-tab">
        {#if over && over.id === id && drag !== id}<span class="absolute inset-y-1 -left-1 w-0.5 rounded bg-accent-500"></span>{/if}
        <EntityIcon type="checklist" size="xs" />
        <span class="min-w-0 flex-1 truncate {on ? 'font-semibold' : ''}">{c?.title ?? '…'}</span>
        {#if c}
          <span class="shrink-0 text-[11px] text-zinc-400 tabular-nums">{c.done}/{c.total}</span>
          {#if c.nextDue && c.nextDue < today && c.done < c.total}
            <span class="size-1.5 shrink-0 rounded-full bg-red-500" title={t('tabs.overdue')}></span>
          {/if}
        {/if}
        <button
          type="button"
          class="flex size-5 shrink-0 items-center justify-center rounded text-zinc-400 hover:bg-black/8 hover:text-zinc-700 dark:hover:bg-white/10 dark:hover:text-zinc-100"
          onclick={(e) => close(e, id)}
          title={t('tabs.close')}
          aria-label="{t('tabs.close')}: {c?.title ?? ''}">
          <Icon name="x" size={12} />
        </button>
      </a>
    {/if}
  {/each}
  <!-- drop zone at the end + "+" -->
  <button
    type="button"
    class="relative flex size-9 shrink-0 items-center justify-center rounded-[10px] {tabCls(false)}"
    ondragover={(e) => dragOver(e, null)}
    ondrop={dropTab}
    onclick={(e) => {
      const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
      query = '';
      picker = { x: r.left, y: r.bottom + 6 };
    }}
    title={t('tabs.open')}
    aria-label={t('tabs.open')}>
    {#if over && over.id === null}<span class="absolute h-6 w-0.5 -translate-x-5 rounded bg-accent-500"></span>{/if}
    <Icon name="plus" size={15} />
  </button>
</div>

{#if picker}
  <ContextMenu x={picker.x} y={picker.y} label={t('tabs.open')} onclose={() => (picker = null)}>
    <input class="input mb-1.5 py-1! text-sm!" placeholder={t('tabs.search')} aria-label={t('tabs.search')} bind:value={query} use:focus
      onkeydown={(e) => {
        if (e.key === 'Enter' && matches[0]) {
          picker = null;
          goto(`/checklists/${matches[0][0]}`);
        }
      }} />
    {#each matches as [id, c] (id)}
      <button
        type="button"
        role="menuitem"
        class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800"
        onclick={() => ((picker = null), goto(`/checklists/${id}`))}>
        <EntityIcon type="checklist" size="sm" />
        <span class="min-w-0 flex-1 leading-tight">
          <span class="block truncate">{c.title}</span>
          <span class="block truncate text-[11px] text-zinc-400">{c.customer} › {c.project}</span>
        </span>
        <span class="shrink-0 text-[11px] text-zinc-400 tabular-nums">{c.done}/{c.total}</span>
      </button>
    {:else}
      <p class="px-2 py-2 text-sm text-zinc-400">{t('tabs.none')}</p>
    {/each}
  </ContextMenu>
{/if}
