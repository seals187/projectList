<script lang="ts">
  // Sidebar: search + new project, navigation, favorites, saved views,
  // at the bottom user, theme, logout. Customers are reached via "Customers".
  import { untrack, type Snippet } from 'svelte';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { canWrite, openNewProject, overviewView, palette, session, setTheme, sidebar, theme, toggleSidebar, type Theme } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { customerCode } from '$lib/projectStatus';
  import { migrateLocalFavorites, toggleFavorite } from '$lib/navMemory.svelte';
  import { applySavedView, isActiveView, removeSavedView, savedViews } from '$lib/overviewFilter.svelte';
  import type { SidebarCustomer, SidebarProject } from '$lib/types';
  import Icon from './Icon.svelte';
  import Avatar from './Avatar.svelte';

  // Projects by ID (favorites, active project)
  const lookup = $derived.by(() => {
    const projects = new Map<number, { p: SidebarProject; c: SidebarCustomer }>();
    const checklists = new Map<number, number>(); // checklist → project
    for (const c of sidebar.customers)
      for (const p of c.projects) {
        projects.set(p.id, { p, c });
        for (const cl of p.checklists) checklists.set(cl.id, p.id);
      }
    return { projects, checklists };
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
  const favorites = $derived(sidebar.favorites.map((id) => lookup.projects.get(id)).filter((x) => !!x));

  // Saved views (filters of the overview): open = apply on the overview
  const onOverview = $derived(page.url.pathname === '/');

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

  // Cycle the theme by click: light → dark → dim → system
  const THEME_ORDER: Theme[] = ['light', 'dark', 'dim', 'system'];
  const themeIcon = $derived(theme.value === 'light' ? 'sun' : theme.value === 'system' ? 'monitor' : 'moon');
  const nextTheme = () => setTheme(THEME_ORDER[(THEME_ORDER.indexOf(theme.value) + 1) % THEME_ORDER.length]);

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

  const navCls = (on: boolean) =>
    `flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 ${
      on
        ? 'bg-accent-50 font-semibold text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
        : 'text-zinc-700 hover:bg-zinc-200/50 dark:text-zinc-300 dark:hover:bg-zinc-800'
    }`;
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
      class="flex min-w-0 flex-1 items-center gap-2 rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-zinc-400 hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-700">
      <Icon name="search" size={14} />
      <span class="flex-1 truncate text-left">{t('nav.search')}</span>
      <kbd class="rounded border border-zinc-200 px-1 font-sans text-[10px] dark:border-zinc-700">Ctrl K</kbd>
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
          <a href={n.href} class={navCls(n.on)}>
            <Icon name={n.icon} class={n.on ? '' : 'text-zinc-500'} />
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

    {#if favorites.length}
      {@render sectionHead('favorites', t('nav.favorites'))}
    {/if}
    {#if favorites.length && !closedSections.favorites}
      <ul class="space-y-px">
        {#each favorites as f (f.p.id)}
          <li class="group relative">
            <a href="/projects/{f.p.id}" class="{navCls(active.projectId === f.p.id && !active.checklistId)} pr-8 font-normal!">
              <Icon name="star" size={15} class="fill-amber-400 text-amber-500" />
              <span class="min-w-0 flex-1 truncate">{f.p.name}</span>
              <span class="text-[11px] text-zinc-400 group-hover:invisible">{customerCode(f.c.name)}</span>
            </a>
            <button
              type="button"
              class="absolute top-1/2 right-1.5 hidden -translate-y-1/2 rounded p-1 text-zinc-400 group-hover:block hover:text-zinc-700 dark:hover:text-zinc-200"
              onclick={() => toggleFavorite(f.p.id)}
              title={t('nav.removeFavorite')}
              aria-label="{t('nav.removeFavorite')}: {f.p.name}">
              <Icon name="x" size={13} />
            </button>
          </li>
        {/each}
      </ul>
    {/if}

    {#if savedViews.list.length}
      {@render sectionHead('views', t('views.title'))}
      {#if !closedSections.views}
        <ul class="space-y-px">
          {#each savedViews.list as v (v.id)}
            {@const on = onOverview && isActiveView(v, overviewView.value)}
            <li class="group relative">
              <button type="button" class="{navCls(on)} w-full pr-8 text-left font-normal!" onclick={() => applySavedView(v)} title={v.name}>
                <Icon name="filter" size={15} class={on ? '' : 'text-zinc-500'} />
                <span class="min-w-0 flex-1 truncate">{v.name}</span>
              </button>
              <button
                type="button"
                class="absolute top-1/2 right-1.5 hidden -translate-y-1/2 rounded p-1 text-zinc-400 group-hover:block hover:text-zinc-700 dark:hover:text-zinc-200"
                onclick={() => removeSavedView(v.id)}
                title={t('views.remove')}
                aria-label="{t('views.remove')}: {v.name}">
                <Icon name="x" size={13} />
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
  </div>

  <!-- Footer: user (→ settings), theme, logout -->
  <div class="flex items-center gap-1 border-t border-zinc-200 px-3 py-2.5 dark:border-zinc-800">
    <a
      href="/settings"
      class="flex min-w-0 flex-1 items-center gap-2.5 rounded-lg px-1.5 py-1 hover:bg-zinc-200/50 dark:hover:bg-zinc-800 {path.startsWith('/settings') ? 'bg-zinc-200/60 dark:bg-zinc-800' : ''}"
      title={t('nav.settings')}>
      <Avatar name={session.user?.name ?? ''} id={session.user?.id ?? 0} size={30} />
      <span class="truncate font-semibold">{session.user?.name}</span>
    </a>
    <button
      type="button"
      class="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-200/60 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
      onclick={nextTheme}
      title="{t('settings.theme')}: {t(`theme.${theme.value}`)}"
      aria-label="{t('settings.theme')}: {t(`theme.${theme.value}`)}">
      <Icon name={themeIcon} size={16} />
    </button>
    <button
      type="button"
      class="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-200/60 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
      onclick={logout}
      title={t('nav.logout')}
      aria-label={t('nav.logout')}>
      <Icon name="logout" size={16} />
    </button>
  </div>
</nav>
