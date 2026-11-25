<script lang="ts">
  // Search (Ctrl+K): customers, projects and checklists. Filter "All / Active /
  // Archived" (Tab switches it); archived projects and checklists of archived
  // projects are shown greyed out with the date of archiving, after the active ones.
  import { goto } from '$app/navigation';
  import { tick } from 'svelte';
  import { api } from '$lib/api';
  import { palette } from '$lib/dialogs.svelte';
  import { fmtDate, t } from '$lib/i18n.svelte';
  import type { ProjectStatus, SearchResult } from '$lib/types';
  import { entitySubtitle, type EntityType } from '$lib/entityTypes';
  import { localDay } from '$lib/archive';
  import { statusBadgeStyle, statusLabel } from '$lib/projectStatus';
  import EntityIcon from './EntityIcon.svelte';

  interface Entry {
    label: string;
    hint: string;
    type?: EntityType; // object type (search results): type icon
    href: string;
    group: string;
    archived?: boolean; // greyed out
    badge?: string; // right: status or "Archived dd.mm.yyyy"
    status?: ProjectStatus; // active project: badge in the status color
  }

  type Scope = 'all' | 'active' | 'archived';
  const SCOPES: Scope[] = ['all', 'active', 'archived'];

  let query = $state('');
  let scope = $state<Scope>('all');
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
    { label: t('nav.archive'), hint: '', href: '/archive', group: t('search.navigation') },
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
      scope = 'all';
      results = [];
      active = 0;
      tick().then(() => input?.focus());
    }
  });

  const archivedBadge = (at: string | null) => (at ? t('archive.archivedOn', { date: fmtDate(localDay(at)) }) : t('search.scope.archived'));

  function search(delay = 150) {
    clearTimeout(timer);
    active = 0;
    const q = query.trim();
    if (!q) return;
    timer = setTimeout(async () => {
      const my = ++seq;
      try {
        const sc = scope === 'all' ? '' : `&scope=${scope}`;
        const r = await api.get<SearchResult>('/search?q=' + encodeURIComponent(q) + sc);
        if (my !== seq) return; // stale response
        results = [
          ...r.customers.map((c) => ({ label: c.name, hint: t('entity.customer'), type: 'customer' as const, href: `/customers/${c.id}`, group: t('search.customers') })),
          ...r.projects.map((p) => {
            const archived = p.status === 'archived';
            return {
              label: p.name,
              hint: entitySubtitle(t('entity.project'), p.customer_name),
              type: 'project' as const,
              href: `/projects/${p.id}`,
              group: t('search.projects'),
              archived,
              badge: archived ? archivedBadge(p.archived_at) : statusLabel(p.status),
              status: archived ? undefined : p.status
            };
          }),
          ...r.checklists.map((c) => {
            const archived = c.project_status === 'archived';
            return {
              label: c.title,
              hint: entitySubtitle(t('entity.checklist'), c.project_name),
              type: 'checklist' as const,
              href: `/checklists/${c.id}`,
              group: t('search.checklists'),
              archived,
              badge: archived ? t('archive.inArchive') : undefined
            };
          })
        ];
      } catch {
        results = [];
      }
    }, delay);
  }

  function setScope(s: Scope) {
    scope = s;
    search(0);
    input?.focus();
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
    } else if (e.key === 'Tab') {
      // Tab / Shift+Tab: next / previous filter
      e.preventDefault();
      const i = SCOPES.indexOf(scope);
      setScope(SCOPES[(i + (e.shiftKey ? SCOPES.length - 1 : 1)) % SCOPES.length]);
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
      class="relative w-full max-w-xl overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-900"
      role="dialog"
      aria-modal="true"
      aria-label={t('nav.search')}>
      <div class="flex items-center gap-2 border-b border-zinc-200 pr-2 dark:border-zinc-800">
        <input
          bind:this={input}
          bind:value={query}
          oninput={() => search()}
          {onkeydown}
          placeholder={t('search.placeholder')}
          class="min-w-0 flex-1 bg-transparent px-4 py-3 text-[15px] outline-none" />
        <div class="flex shrink-0 gap-1" role="radiogroup" aria-label={t('search.scope')}>
          {#each SCOPES as s (s)}
            <button
              type="button"
              role="radio"
              aria-checked={scope === s}
              tabindex="-1"
              class="rounded-full px-2.5 py-1 text-xs font-medium transition-colors {scope === s
                ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
                : 'bg-zinc-100 text-zinc-600 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700'}"
              onclick={() => setScope(s)}
              data-testid="search-scope-{s}">{t(`search.scope.${s}`)}</button>
          {/each}
        </div>
      </div>
      <ul class="max-h-96 overflow-y-auto p-1.5">
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
                : ''}"
              data-archived={e.archived || undefined}>
              <span class="flex min-w-0 items-center gap-2.5 {e.archived ? 'opacity-55' : ''}">
                {#if e.type}<EntityIcon type={e.type} size="sm" />{/if}
                <span class="min-w-0">
                  <span class="block truncate">{e.label}</span>
                  {#if e.type && e.hint}<span class="block truncate text-xs text-zinc-400">{e.hint}</span>{/if}
                </span>
              </span>
              {#if e.badge}
                {#if e.status}
                  <span class="badge shrink-0" style={statusBadgeStyle(e.status)}>{e.badge}</span>
                {:else}
                  <span class="shrink-0 rounded-md bg-zinc-100 px-2 py-0.5 text-xs text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400">{e.badge}</span>
                {/if}
              {:else if e.hint && !e.type}
                <span class="shrink-0 text-xs text-zinc-400">{e.hint}</span>
              {/if}
            </button>
          </li>
        {:else}
          <li class="px-3 py-6 text-center text-sm text-zinc-400">{t('search.empty')}</li>
        {/each}
      </ul>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-zinc-200 px-4 py-2 text-xs text-zinc-400 dark:border-zinc-800">
        <span><kbd class="font-sans">↑↓</kbd> {t('search.hint.select')}</span>
        <span><kbd class="font-sans">↵</kbd> {t('search.hint.open')}</span>
        <span><kbd class="font-sans">Tab</kbd> {t('search.hint.tab')}</span>
        <span class="ml-auto">{t('search.hint.archived')}</span>
      </div>
    </div>
  </div>
{/if}
