<script lang="ts">
  import { goto } from '$app/navigation';
  import { tick } from 'svelte';
  import { api } from '$lib/api';
  import { palette } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { SearchResult } from '$lib/types';
  import { entitySubtitle, type EntityType } from '$lib/entityTypes';
  import EntityIcon from './EntityIcon.svelte';

  interface Entry {
    label: string;
    hint: string;
    type?: EntityType; // object type (search results): type icon
    href: string;
    group: string;
  }

  let query = $state('');
  let results = $state<Entry[]>([]);
  let active = $state(0);
  let input = $state<HTMLInputElement>();
  let timer: ReturnType<typeof setTimeout>;
  let seq = 0;

  const staticEntries = $derived<Entry[]>([
    { label: t('nav.overview'), hint: '', href: '/', group: t('search.navigation') },
    { label: t('nav.myTasks'), hint: '', href: '/tasks', group: t('search.navigation') },
    { label: t('nav.customers'), hint: '', href: '/customers', group: t('search.navigation') },
    { label: t('nav.templates'), hint: '', href: '/templates', group: t('search.navigation') },
    { label: t('nav.settings'), hint: '', href: '/settings', group: t('search.navigation') }
  ]);

  const entries = $derived(query.trim() ? results : staticEntries);

  function onGlobalKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      palette.open = !palette.open;
    }
  }

  $effect(() => {
    if (palette.open) {
      query = '';
      results = [];
      active = 0;
      tick().then(() => input?.focus());
    }
  });

  function search() {
    clearTimeout(timer);
    active = 0;
    const q = query.trim();
    if (!q) return;
    timer = setTimeout(async () => {
      const my = ++seq;
      try {
        const r = await api.get<SearchResult>('/search?q=' + encodeURIComponent(q));
        if (my !== seq) return; // stale response
        results = [
          ...r.customers.map((c) => ({ label: c.name, hint: t('entity.customer'), type: 'customer' as const, href: `/customers/${c.id}`, group: t('search.customers') })),
          ...r.projects.map((p) => ({ label: p.name, hint: entitySubtitle(t('entity.project'), p.customer_name), type: 'project' as const, href: `/projects/${p.id}`, group: t('search.projects') })),
          ...r.checklists.map((c) => ({ label: c.title, hint: entitySubtitle(t('entity.checklist'), c.project_name), type: 'checklist' as const, href: `/checklists/${c.id}`, group: t('search.checklists') }))
        ];
      } catch {
        results = [];
      }
    }, 150);
  }

  function choose(e: Entry | undefined) {
    if (!e) return;
    palette.open = false;
    goto(e.href);
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      active = Math.min(active + 1, entries.length - 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      active = Math.max(active - 1, 0);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      choose(entries[active]);
    } else if (e.key === 'Escape') {
      palette.open = false;
    }
  }
</script>

<svelte:window onkeydown={onGlobalKey} />

{#if palette.open}
  <div class="fixed inset-0 z-40 flex items-start justify-center bg-zinc-950/20 px-4 pt-[12vh] dark:bg-black/50">
    <button type="button" class="absolute inset-0 cursor-default" aria-label={t('nav.closeMenu')} onclick={() => (palette.open = false)}></button>
    <div
      class="relative w-full max-w-lg overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-900"
      role="dialog"
      aria-modal="true"
      aria-label={t('nav.search')}>
      <input
        bind:this={input}
        bind:value={query}
        oninput={search}
        {onkeydown}
        placeholder={t('search.placeholder')}
        class="w-full border-b border-zinc-200 bg-transparent px-4 py-3 text-[15px] outline-none dark:border-zinc-800" />
      <ul class="max-h-80 overflow-y-auto p-1.5">
        {#each entries as e, i (e.href + i)}
          {#if i === 0 || entries[i - 1].group !== e.group}
            <li class="px-2.5 pt-2 pb-1 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{e.group}</li>
          {/if}
          <li>
            <button
              type="button"
              onclick={() => choose(e)}
              onmousemove={() => (active = i)}
              class="flex w-full items-center justify-between gap-3 rounded-md px-2.5 py-2 text-left text-sm {i === active
                ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
                : ''}">
              <span class="flex min-w-0 items-center gap-2.5">{#if e.type}<EntityIcon type={e.type} size="sm" />{/if}<span class="truncate">{e.label}</span></span>
              {#if e.hint}<span class="shrink-0 text-xs text-zinc-400">{e.hint}</span>{/if}
            </button>
          </li>
        {:else}
          <li class="px-3 py-6 text-center text-sm text-zinc-400">{t('search.empty')}</li>
        {/each}
      </ul>
    </div>
  </div>
{/if}
