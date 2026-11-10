<script lang="ts">
  import '../app.css';
  import { onMount, untrack } from 'svelte';
  import { goto, afterNavigate } from '$app/navigation';
  import { page } from '$app/state';
  import { api, setCsrf } from '$lib/api';
  import {
    newProject,
    prefs,
    refreshSidebar,
    saveState,
    session,
    setSidebarWidth,
    sidebar,
    SIDEBAR_DEFAULT,
    SIDEBAR_MAX,
    SIDEBAR_MIN,
    sidebarWidth,
    toggleSidebar
  } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { syncFromServer } from '$lib/overviewFilter.svelte';
  import { backgroundCss, userBackground } from '$lib/backgrounds';
  import { pageBg } from '$lib/pageBg.svelte';
  import type { User } from '$lib/types';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import CommandPalette from '$lib/components/CommandPalette.svelte';
  import NewProjectDialog from '$lib/components/NewProjectDialog.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  import BackBar from '$lib/components/BackBar.svelte';
  import { DIM_ALPHA } from '$lib/appearance';
  import ChecklistTabs from '$lib/components/ChecklistTabs.svelte';
  import { recordNav } from '$lib/navHistory.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let { children } = $props();

  // Active background: project (also on its checklists and tables), else its customer, else the user's own
  const bgCss = $derived.by(() => {
    const m = page.url.pathname.match(/^\/(customers|projects|checklists|tables)\/(\d+)/);
    if (m) {
      const id = Number(m[2]);
      for (const c of sidebar.customers) {
        if (m[1] === 'customers' && c.id === id) return backgroundCss(c.background || userBackground());
        for (const p of c.projects)
          if ((m[1] === 'projects' && p.id === id) || (m[1] === 'checklists' && p.checklists.some((cl) => cl.id === id)) || (m[1] === 'tables' && (p.tables ?? []).some((tb) => tb.id === id)))
            return backgroundCss(p.background || c.background || userBackground());
      }
    }
    return backgroundCss(userBackground());
  });
  // Sidebar and bars over a background image: dark glass (glass effect on); with the
  // glass effect off they look exactly as without a background image
  const glassChrome = $derived(!!bgCss && prefs.glass);
  const dimAlpha = $derived(DIM_ALPHA[prefs.dimContent] ?? 0);
  // "Darken image" (Settings → Appearance): a dark veil over the background
  const bgImage = $derived(bgCss && prefs.dimBg ? `linear-gradient(rgb(0 0 0 / 0.35), rgb(0 0 0 / 0.35)), ${bgCss}` : bgCss);
  // pages with free-flowing content (checklist/table/template editor, settings) on one panel
  const panelPage = $derived(/^\/(checklists|tables|templates\/\d|settings)/.test(page.url.pathname));
  // settings: own two-column shell (SettingsShell) that fills the panel without padding
  const settingsPage = $derived(page.url.pathname.startsWith('/settings'));
  // reachable without a session: sign-in and invitation / password links
  const publicPage = $derived(page.url.pathname === '/login' || page.url.pathname === '/invite');
  // print views (report as PDF): without sidebar and back bar
  const printPage = $derived(page.url.pathname.startsWith('/reports/print'));
  // checklist page: tabs of the open checklists above the content
  const checklistId = $derived(Number(page.url.pathname.match(/^\/checklists\/(\d+)/)?.[1] ?? 0) || null);
  $effect(() => {
    pageBg.active = !!bgCss && !panelPage;
    pageBg.panel = !!bgCss && panelPage;
  });

  onMount(async () => {
    try {
      const me = await api.get<{ user: User; csrf_token: string; app_subtitle?: string; version?: string }>('/auth/me');
      session.user = me.user;
      session.subtitle = me.app_subtitle ?? '';
      session.version = me.version ?? '';
      setCsrf(me.csrf_token);
      refreshSidebar(0);
    } catch {
      session.user = null;
    }
    session.ready = true;
  });

  // overview filters and saved views (synced per user) whenever the sidebar data arrives
  $effect(() => {
    if (sidebar.prefs) untrack(syncFromServer);
  });

  $effect(() => {
    if (session.ready && !session.user && !publicPage) goto('/login');
  });

  // Desktop width (md): only there is the collapsed sidebar made inert for keyboard/screen readers.
  let isDesktop = $state(true);
  onMount(() => {
    const mq = matchMedia('(min-width: 48rem)');
    isDesktop = mq.matches;
    const fn = (e: MediaQueryListEvent) => (isDesktop = e.matches);
    mq.addEventListener('change', fn);
    return () => mq.removeEventListener('change', fn);
  });

  // Drag the sidebar wider/narrower (desktop). No animation while
  // dragging; saved only on release.
  let resizing = $state(false);
  function startResize(e: PointerEvent) {
    if (e.button !== 0) return;
    e.preventDefault();
    const handle = e.currentTarget as HTMLElement;
    handle.setPointerCapture(e.pointerId);
    const startX = e.clientX;
    const startW = sidebarWidth.value;
    resizing = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    const move = (ev: PointerEvent) => setSidebarWidth(startW + ev.clientX - startX, false);
    const up = () => {
      resizing = false;
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      setSidebarWidth(sidebarWidth.value);
      handle.removeEventListener('pointermove', move);
      handle.removeEventListener('pointerup', up);
      handle.removeEventListener('pointercancel', up);
    };
    handle.addEventListener('pointermove', move);
    handle.addEventListener('pointerup', up);
    handle.addEventListener('pointercancel', up);
  }

  // Ctrl+\ collapses/expands the sidebar (desktop).
  function onKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === '\\') {
      e.preventDefault();
      toggleSidebar();
    }
  }

  // After navigation: history of the back bar, close the mobile sidebar, reset the save indicator.
  afterNavigate((nav) => {
    if (nav.to) recordNav(nav.type, nav.delta, nav.to.url);
    sidebar.open = false;
    saveState.saved = false;
    saveState.error = false;
  });
</script>

<svelte:window onkeydown={onKey} />

{#if !session.ready}
  <div class="h-dvh"></div>
{:else if publicPage}
  {@render children()}
{:else if session.user && printPage}
  {@render children()}
{:else if session.user}
  <div
    class="flex h-dvh overflow-hidden"
    style:background-image={bgImage || undefined}
    style:background-size={bgCss ? 'cover' : undefined}
    style:background-position={bgCss ? 'center' : undefined}>
    {#if sidebar.open}
      <button
        type="button"
        class="fixed inset-0 z-30 bg-zinc-950/30 md:hidden"
        aria-label={t('nav.closeMenu')}
        onclick={() => (sidebar.open = false)}></button>
    {/if}
    <aside
      class="fixed inset-y-0 left-0 z-30 w-64 overflow-hidden md:relative md:translate-x-0
        {glassChrome ? 'glass-bar dark border-white/10 bg-zinc-950/45 text-zinc-100 backdrop-blur-lg' : 'border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900'}
        {resizing ? '' : 'transition-[translate,width] duration-200'}
        {sidebar.open ? 'translate-x-0' : '-translate-x-full'}
        {sidebar.collapsed ? 'border-r md:w-0 md:border-r-0' : 'border-r md:w-(--sb-w)'}"
      style:--sb-w="{sidebarWidth.value}px"
      inert={sidebar.collapsed && !sidebar.open && isDesktop}>
      <!-- fixed width so the content doesn't reflow when collapsing/expanding -->
      <div class="h-full w-64 md:w-(--sb-w)">
        <Sidebar />
      </div>
      {#if !sidebar.collapsed}
        <!-- drag handle on the right edge: change width, double-click = default width -->
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions, a11y_no_noninteractive_tabindex -->
        <div
          class="absolute inset-y-0 right-0 z-10 hidden w-1.5 cursor-col-resize touch-none transition-colors hover:bg-accent-400/50 md:block {resizing ? 'bg-accent-500/60' : ''}"
          role="separator"
          aria-orientation="vertical"
          aria-valuenow={sidebarWidth.value}
          aria-valuemin={SIDEBAR_MIN}
          aria-valuemax={SIDEBAR_MAX}
          aria-label={t('nav.resizeSidebar')}
          title={t('nav.resizeSidebar')}
          tabindex="0"
          onpointerdown={startResize}
          ondblclick={() => setSidebarWidth(SIDEBAR_DEFAULT)}
          onkeydown={(e) => {
            if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
              e.preventDefault();
              setSidebarWidth(sidebarWidth.value + (e.key === 'ArrowRight' ? 16 : -16));
            }
          }}></div>
      {/if}
    </aside>

    {#if sidebar.collapsed}
      <button
        type="button"
        class="fixed top-3 left-2 z-20 hidden rounded-md border border-zinc-200 bg-white/90 p-1.5 text-zinc-500 shadow-xs backdrop-blur hover:text-zinc-800 md:block dark:border-zinc-800 dark:bg-zinc-900/90 dark:text-zinc-400 dark:hover:text-zinc-100"
        onclick={() => toggleSidebar(false)}
        title="{t('nav.expandSidebar')} (Ctrl+\)"
        aria-label={t('nav.expandSidebar')}>
        <Icon name="sidebar" size={16} />
      </button>
    {/if}

    <div class="flex min-w-0 flex-1 flex-col">
      <header class="flex items-center gap-2 border-b px-3 py-2 md:hidden {glassChrome ? 'glass-bar dark border-white/10 bg-zinc-950/45 text-zinc-100 backdrop-blur-lg' : 'border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900'}">
        <button type="button" class="btn-ghost px-2!" onclick={() => (sidebar.open = true)} aria-label={t('nav.openMenu')}>
          <Icon name="menu" size={18} />
        </button>
        <img src="/favicon.svg" alt="" width="20" height="20" />
        <span class="font-semibold">ProjectList</span>
      </header>
      <BackBar glass={glassChrome} />
      <!-- Background (project → customer → own) lies behind the whole app; sidebar and
           header bands are dark glass, content surfaces light glass (app.css, main.has-bg).
           Editor and settings pages keep their content on one opaque panel instead.
           No backdrop-filter inside main: it would make fixed menus/dialogs position wrongly. -->
      <!-- main window; "Darken content" (Settings → Appearance) lays a dark veil over it that lets
           clicks through (menus and dialogs lie above it) -->
      <div class="relative isolate flex min-h-0 flex-1 flex-col">
      <main class="flex-1 overflow-y-auto {bgCss && !panelPage ? 'has-bg' : ''}">
        {#if checklistId}
          <!-- open checklists as tabs -->
          <div class={bgCss ? 'px-2 pt-2 sm:px-4 sm:pt-3' : 'px-4 pt-3 sm:px-10'}>
            <ChecklistTabs activeId={checklistId} onImage={!!bgCss} />
          </div>
        {/if}
        <div
          class={settingsPage
            ? bgCss
              ? 'glass-bar m-2 h-[calc(100%-1rem)] overflow-hidden rounded-2xl bg-white shadow-lg sm:m-4 sm:h-[calc(100%-2rem)] dark:bg-zinc-900'
              : 'h-full'
            : bgCss && panelPage
              ? `glass-bar m-2 min-h-[calc(100%-1rem)] rounded-2xl bg-white px-4 py-5 shadow-lg sm:m-4 sm:min-h-[calc(100%-2rem)] sm:px-8 sm:py-6 dark:bg-zinc-900 ${checklistId ? 'mt-1! sm:mt-1.5!' : ''}`
              : `px-4 py-5 sm:px-10 sm:py-7 ${checklistId ? 'pt-3! sm:pt-4!' : ''}`}>
          {@render children()}
        </div>
      </main>
      {#if dimAlpha > 0}
        <div class="pointer-events-none absolute inset-0 z-40" style:background-color="rgb(0 0 0 / {dimAlpha})" aria-hidden="true" data-testid="content-dim"></div>
      {/if}
      </div>
    </div>
  </div>
  <CommandPalette />
  {#if newProject.value}
    <NewProjectDialog customerId={newProject.value.customerId} status={newProject.value.status} onclose={() => (newProject.value = null)} />
  {/if}
{/if}
<Toasts />
