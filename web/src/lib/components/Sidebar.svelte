<script lang="ts">
  // Sidebar: search + new project, navigation, favorites, saved views,
  // at the bottom user, theme, logout. Customers are reached via "Customers".
  import { untrack, type Snippet } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { canWrite, openNewProject, overviewView, palette, session, setTheme, sidebar, theme, toggleSidebar } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { migrateLocalFavorites, toggleFavorite } from '$lib/navMemory.svelte';
  import { applySavedView, isActivePageView, isActiveView, removeSavedView, renameSavedView, resetFilters, savedViews } from '$lib/overviewFilter.svelte';
  import type { SidebarCustomer, SidebarProject } from '$lib/types';
  import type { FavEntry } from '$lib/favLayout';
  import SidebarTree from './SidebarTree.svelte';
  import Icon from './Icon.svelte';
  import Avatar from './Avatar.svelte';

  // Customers, projects and checklists by ID (favorites, active project)
  const lookup = $derived.by(() => {
    const customers = new Map<number, SidebarCustomer>();
    const projects = new Map<number, { p: SidebarProject; c: SidebarCustomer }>();
    const checklists = new Map<number, number>(); // checklist → project
    const checklistTitles = new Map<number, string>();
    for (const c of sidebar.customers) {
      customers.set(c.id, c);
      for (const p of c.projects) {
        projects.set(p.id, { p, c });
        for (const cl of p.checklists) {
          checklists.set(cl.id, p.id);
          checklistTitles.set(cl.id, cl.title);
        }
      }
    }
    return { customers, projects, checklists, checklistTitles };
  });

  // Active customer/project from the URL ("+" preselects the customer, favorite highlighted)
  const active = $derived.by(() => {
    const m = page.url.pathname.match(/^\/(customers|projects|checklists)\/(\d+)/);
    let customerId: number | null = null,
      projectId: number | null = null,
      checklistId: number | null = null;
    if (m) {
      const id = Number(m[2]);
      if (m[1] === 'customers') customerId = id;
      if (m[1] === 'checklists') checklistId = id;
      projectId = m[1] === 'projects' ? id : m[1] === 'checklists' ? (lookup.checklists.get(id) ?? null) : null;
      if (projectId) customerId = lookup.projects.get(projectId)?.c.id ?? null;
    }
    return { customerId, projectId, checklistId };
  });

  // Favorites: per user in the DB
  $effect(() => {
    if (sidebar.customers.length) untrack(migrateLocalFavorites); // migrate old browser favorites once
  });
  // resolved favorites (targets that are gone or no longer visible are skipped)
  const favorites = $derived(
    sidebar.favorites.flatMap((f): FavEntry[] => {
      const key = `${f.kind}:${f.id}`;
      if (f.kind === 'customer') {
        const c = lookup.customers.get(f.id);
        return c ? [{ key, ...f, href: `/customers/${c.id}`, label: c.name, hint: '', on: page.url.pathname === `/customers/${c.id}` }] : [];
      }
      if (f.kind === 'project') {
        const x = lookup.projects.get(f.id);
        return x ? [{ key, ...f, href: `/projects/${x.p.id}`, label: x.p.name, hint: '', on: active.projectId === x.p.id && !active.checklistId }] : [];
      }
      const pid = lookup.checklists.get(f.id);
      const x = pid ? lookup.projects.get(pid) : undefined;
      return x ? [{ key, ...f, href: `/checklists/${f.id}`, label: lookup.checklistTitles.get(f.id) ?? '', hint: x.p.name, on: active.checklistId === f.id }] : [];
    })
  );

  const favByKey = $derived(new Map(favorites.map((f) => [f.key, f])));
  const viewById = $derived(new Map(savedViews.list.map((v) => [v.id, v])));
  const menuItemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800';

  // Saved views (filters of the overview): open = apply on the overview
  const onOverview = $derived(page.url.pathname === '/');
  // rename a saved view (pencil or double click)
  let renaming = $state<string | null>(null);
  let renameText = $state('');
  function startRename(id: string, name: string) {
    renaming = id;
    renameText = name;
  }
  function finishRename(id: string) {
    if (renaming !== id) return;
    renaming = null;
    renameSavedView(id, renameText);
  }

  // Sections (favorites, saved views) collapsible, remembered per browser
  type SectionKey = 'favorites' | 'views';
  let closedSections = $state<Partial<Record<SectionKey, boolean>>>(loadSections());
  function loadSections() {
    try {
      return JSON.parse(localStorage.getItem('sidebar-sections') ?? '{}');
    } catch {
      return {};
    }
  }
  function toggleSection(k: SectionKey) {
    closedSections[k] = !closedSections[k];
    try {
      localStorage.setItem('sidebar-sections', JSON.stringify(closedSections));
    } catch {
      /* ignore */
    }
  }

  // Toggle the theme by click: light ↔ dark (from "system": the opposite of what is shown)
  const themeIcon = $derived(theme.dark ? 'moon' : 'sun');
  const nextTheme = () => setTheme(theme.dark ? 'light' : 'dark');

  async function logout() {
    try {
      await api.post('/auth/logout');
    } finally {
      session.user = null;
      goto('/login');
    }
  }

  const path = $derived(page.url.pathname);
  const nav = $derived([
    { href: '/', icon: 'home' as const, label: t('nav.overview'), on: path === '/' },
    { href: '/tasks', icon: 'tasks' as const, label: t('nav.myTasks'), on: path === '/tasks', badge: sidebar.myTasks },
    { href: '/customers', icon: 'users' as const, label: t('nav.customers'), on: /^\/(customers|projects|checklists)(\/|$)/.test(path) },
    { href: '/templates', icon: 'template' as const, label: t('nav.templates'), on: path.startsWith('/templates') }
  ]);

  // same selection colors as view switchers and chips (app.css: sel-on / sel-off)
  const navCls = (on: boolean) => `flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 ${on ? 'sel-on' : 'sel-off font-medium'}`;
  const canCreate = $derived(canWrite() && sidebar.customers.length > 0);
</script>

<nav class="flex h-full flex-col text-sm">
  <!-- Header: logo, name, subtitle, collapse -->
  <div class="flex items-center gap-2.5 pt-4 pr-3 pb-3 pl-4">
    <a href="/" class="flex min-w-0 flex-1 items-center gap-2.5">
      <img src="/favicon.svg" alt="" width="30" height="30" class="shrink-0" />
      <span class="min-w-0">
        <span class="block truncate text-[15px] leading-tight font-semibold tracking-tight">ProjectList</span>
        {#if session.subtitle}<span class="block truncate text-xs text-zinc-500">{session.subtitle}</span>{/if}
      </span>
    </a>
    <button
      type="button"
      class="hidden rounded-md p-1 text-zinc-400 hover:bg-zinc-200/70 hover:text-zinc-700 md:block dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
      onclick={() => toggleSidebar(true)}
      title="{t('nav.collapseSidebar')} (Ctrl+\)"
      aria-label={t('nav.collapseSidebar')}>
      <Icon name="sidebar" size={16} />
    </button>
  </div>

  <!-- Search + new project -->
  <div class="flex gap-2 px-3">
    <button
      type="button"
      onclick={() => (palette.open = true)}
      class="flex min-w-0 flex-1 items-center gap-2 rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-zinc-500 transition-colors hover:border-zinc-300 hover:text-zinc-700 dark:border-white/10 dark:bg-white/10 dark:text-zinc-300 dark:hover:border-white/20 dark:hover:bg-white/15 dark:hover:text-white">
      <Icon name="search" size={14} />
      <span class="flex-1 truncate text-left">{t('nav.search')}</span>
      <kbd class="rounded border border-zinc-200 px-1 font-sans text-[10px] dark:border-white/20">Ctrl K</kbd>
    </button>
    {#if canCreate}
      <button
        type="button"
        class="flex size-[34px] shrink-0 items-center justify-center rounded-lg bg-accent-600 text-white shadow-xs hover:bg-accent-700"
        onclick={() => openNewProject(active.customerId)}
        title={t('overview.newProject')}
        aria-label={t('overview.newProject')}>
        <Icon name="plus" size={17} />
      </button>
    {/if}
  </div>

  <div class="mt-3 flex-1 overflow-y-auto px-3 pb-4">
    <ul class="space-y-px">
      {#each nav as n (n.href)}
        <li>
          <!-- "Overview" always opens the unfiltered overview (saved views apply their own filters) -->
          <a href={n.href} class={navCls(n.on)} onclick={n.href === '/' ? resetFilters : undefined}>
            <Icon name={n.icon} class={n.on ? '' : 'opacity-70'} />
            <span class="flex-1">{n.label}</span>
            {#if n.badge}<span class="rounded-md bg-zinc-200/70 px-1.5 text-[11px] text-zinc-600 tabular-nums dark:bg-zinc-800 dark:text-zinc-400">{n.badge}</span>{/if}
          </a>
        </li>
      {/each}
    </ul>

    {#snippet sectionHead(k: SectionKey, label: string, extra?: Snippet)}
      <div class="mt-5 mb-1 flex items-center justify-between pr-2.5 pl-1">
        <button
          type="button"
          class="flex items-center gap-1 rounded py-0.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase hover:text-zinc-600 dark:hover:text-zinc-300"
          onclick={() => toggleSection(k)}
          aria-expanded={!closedSections[k]}>
          <Icon name="chevronDown" size={12} class="transition-transform {closedSections[k] ? '-rotate-90' : ''}" />
          {label}
        </button>
        {@render extra?.()}
      </div>
    {/snippet}

    <!-- Favorites: own order and folders (pref "fav_layout") -->
    {#snippet favHead()}{@render sectionHead('favorites', t('nav.favorites'))}{/snippet}
    {#snippet favItem(key: string, nested: boolean)}
      {@const f = favByKey.get(key)}
      {#if f}
        <a href={f.href} class="{navCls(f.on)} pr-8 font-normal! {nested ? 'pl-7!' : ''}" title={f.hint ? `${f.label} · ${f.hint}` : f.label} draggable="false">
          <Icon name="star" size={15} class="fill-amber-400 text-amber-500" />
          <span class="min-w-0 flex-1 truncate">{f.label}</span>
        </a>
        <button
          type="button"
          class="absolute top-1/2 right-1.5 hidden -translate-y-1/2 rounded p-1 text-zinc-400 group-hover:block hover:text-zinc-700 dark:hover:text-zinc-200 [@media(hover:none)]:block"
          onclick={() => toggleFavorite(f.kind, f.id)}
          title={t('nav.removeFavorite')}
          aria-label="{t('nav.removeFavorite')}: {f.label}">
          <Icon name="x" size={13} />
        </button>
      {/if}
    {/snippet}
    {#snippet favMenu(key: string, close: () => void)}
      {@const f = favByKey.get(key)}
      {#if f}
        <button type="button" class="{menuItemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => (toggleFavorite(f.kind, f.id), close())}>
          <Icon name="star" size={15} />{t('nav.removeFavorite')}
        </button>
      {/if}
    {/snippet}
    <SidebarTree
      keys={favorites.map((f) => f.key)}
      prefKey="fav_layout"
      label={t('nav.favorites')}
      closed={!!closedSections.favorites}
      head={favHead}
      {navCls}
      item={favItem}
      itemMenu={favMenu}
      testid="fav" />

    <!-- Saved views: own order and folders (pref "views_layout") -->
    {#snippet viewsHead()}{@render sectionHead('views', t('views.title'))}{/snippet}
    {#snippet viewItem(id: string, nested: boolean)}
      {@const v = viewById.get(id)}
      {#if v}
        {@const on = v.page === 'overview' ? onOverview && isActiveView(v, overviewView.value) : isActivePageView(v, page.url.pathname, page.url.search)}
        {#if renaming === v.id}
          <!-- rename: Enter or leaving the field saves, Esc cancels -->
          <!-- svelte-ignore a11y_autofocus -->
          <input
            class="input py-1! text-sm!"
            maxlength="60"
            bind:value={renameText}
            autofocus
            aria-label={t('views.rename')}
            onkeydown={(e) => {
              if (e.key === 'Enter') finishRename(v.id);
              if (e.key === 'Escape') renaming = null;
            }}
            onblur={() => finishRename(v.id)} />
        {:else}
          <button type="button" class="{navCls(on)} w-full pr-14 text-left font-normal! {nested ? 'pl-7!' : ''}" onclick={() => applySavedView(v)} ondblclick={() => startRename(v.id, v.name)} title={v.name}>
            <Icon name="filter" size={15} class={on ? '' : 'opacity-70'} />
            <span class="min-w-0 flex-1 truncate">{v.name}</span>
          </button>
          <span class="absolute top-1/2 right-1.5 hidden -translate-y-1/2 items-center group-hover:flex [@media(hover:none)]:flex">
            <button
              type="button"
              class="rounded p-1 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
              onclick={() => startRename(v.id, v.name)}
              title={t('views.rename')}
              aria-label="{t('views.rename')}: {v.name}">
              <Icon name="pencil" size={13} />
            </button>
            <button
              type="button"
              class="rounded p-1 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
              onclick={() => removeSavedView(v.id)}
              title={t('views.remove')}
              aria-label="{t('views.remove')}: {v.name}">
              <Icon name="x" size={13} />
            </button>
          </span>
        {/if}
      {/if}
    {/snippet}
    {#snippet viewMenu(id: string, close: () => void)}
      {@const v = viewById.get(id)}
      {#if v}
        <button type="button" class={menuItemCls} role="menuitem" onclick={() => (startRename(v.id, v.name), close())}>
          <Icon name="pencil" size={15} class="text-zinc-400" />{t('views.rename')}
        </button>
        <button type="button" class="{menuItemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => (removeSavedView(v.id), close())}>
          <Icon name="trash" size={15} />{t('views.remove')}
        </button>
      {/if}
    {/snippet}
    <SidebarTree
      keys={savedViews.list.map((v) => v.id)}
      prefKey="views_layout"
      label={t('views.title')}
      closed={!!closedSections.views}
      head={viewsHead}
      {navCls}
      item={viewItem}
      itemMenu={viewMenu}
      testid="view" />
  </div>

  <!-- Footer: user (→ settings), theme, logout -->
  <div class="flex items-center gap-1 border-t border-zinc-200 px-3 py-2.5 dark:border-zinc-800">
    <a
      href="/settings"
      class="flex min-w-0 flex-1 items-center gap-2.5 rounded-lg px-1.5 py-1 {path.startsWith('/settings') ? 'sel-on' : 'sel-off'}"
      title={t('nav.settings')}>
      <Avatar name={session.user?.name ?? ''} id={session.user?.id ?? 0} size={30} />
      <span class="truncate font-semibold">{session.user?.name}</span>
    </a>
    <button
      type="button"
      class="rounded-md p-1.5 text-zinc-600 hover:bg-zinc-200/60 hover:text-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
      onclick={nextTheme}
      title="{t('settings.theme')}: {t(`theme.${theme.value}`)}"
      aria-label="{t('settings.theme')}: {t(`theme.${theme.value}`)}">
      <Icon name={themeIcon} size={16} />
    </button>
    <button
      type="button"
      class="rounded-md p-1.5 text-zinc-600 hover:bg-zinc-200/60 hover:text-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
      onclick={logout}
      title={t('nav.logout')}
      aria-label={t('nav.logout')}>
      <Icon name="logout" size={16} />
    </button>
  </div>
</nav>
