<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto, afterNavigate } from '$app/navigation';
  import { page } from '$app/state';
  import { api, setCsrf } from '$lib/api';
  import {
    newProject,
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
  import type { User } from '$lib/types';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import CommandPalette from '$lib/components/CommandPalette.svelte';
  import NewProjectDialog from '$lib/components/NewProjectDialog.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let { children } = $props();

  onMount(async () => {
    try {
      const me = await api.get<{ user: User; csrf_token: string; app_subtitle?: string }>('/auth/me');
      session.user = me.user;
      session.subtitle = me.app_subtitle ?? '';
      setCsrf(me.csrf_token);
      refreshSidebar(0);
    } catch {
      session.user = null;
    }
    session.ready = true;
  });

  $effect(() => {
    if (session.ready && !session.user && page.url.pathname !== '/login') goto('/login');
  });

  // Desktop-Breite (md): nur dort wird die eingeklappte Sidebar für Tastatur/Screenreader gesperrt.
  let isDesktop = $state(true);
  onMount(() => {
    const mq = matchMedia('(min-width: 48rem)');
    isDesktop = mq.matches;
    const fn = (e: MediaQueryListEvent) => (isDesktop = e.matches);
    mq.addEventListener('change', fn);
    return () => mq.removeEventListener('change', fn);
  });

  // Seitenleiste breiter/schmaler ziehen (Desktop). Während des Ziehens ohne
  // Animation; gespeichert wird erst beim Loslassen.
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

  // Strg+\ klappt die Sidebar ein/aus (Desktop).
  function onKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === '\\') {
      e.preventDefault();
      toggleSidebar();
    }
  }

  // Nach Navigation: mobile Sidebar schließen, Speicheranzeige zurücksetzen.
  afterNavigate(() => {
    sidebar.open = false;
    saveState.saved = false;
    saveState.error = false;
  });
</script>

<svelte:window onkeydown={onKey} />

{#if !session.ready}
  <div class="h-dvh"></div>
{:else if page.url.pathname === '/login'}
  {@render children()}
{:else if session.user}
  <div class="flex h-dvh overflow-hidden">
    {#if sidebar.open}
      <button
        type="button"
        class="fixed inset-0 z-30 bg-zinc-950/30 md:hidden"
        aria-label={t('nav.closeMenu')}
        onclick={() => (sidebar.open = false)}></button>
    {/if}
    <aside
      class="fixed inset-y-0 left-0 z-30 w-64 overflow-hidden border-zinc-200 bg-zinc-50 md:relative md:translate-x-0 dark:border-zinc-800 dark:bg-zinc-900
        {resizing ? '' : 'transition-[translate,width] duration-200'}
        {sidebar.open ? 'translate-x-0' : '-translate-x-full'}
        {sidebar.collapsed ? 'border-r md:w-0 md:border-r-0' : 'border-r md:w-(--sb-w)'}"
      style:--sb-w="{sidebarWidth.value}px"
      inert={sidebar.collapsed && !sidebar.open && isDesktop}>
      <!-- feste Breite, damit der Inhalt beim Ein-/Ausklappen nicht umbricht -->
      <div class="h-full w-64 md:w-(--sb-w)">
        <Sidebar />
      </div>
      {#if !sidebar.collapsed}
        <!-- Ziehgriff am rechten Rand: Breite ändern, Doppelklick = Standardbreite -->
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
      <header class="flex items-center gap-2 border-b border-zinc-200 px-3 py-2 md:hidden dark:border-zinc-800">
        <button type="button" class="btn-ghost px-2!" onclick={() => (sidebar.open = true)} aria-label={t('nav.openMenu')}>
          <Icon name="menu" size={18} />
        </button>
        <img src="/favicon.svg" alt="" width="20" height="20" />
        <span class="font-semibold">ProjectList</span>
      </header>
      <main class="flex-1 overflow-y-auto">
        <div class="px-4 py-5 sm:px-10 sm:py-7">
          {@render children()}
        </div>
      </main>
    </div>
  </div>
  <CommandPalette />
  {#if newProject.value}
    <NewProjectDialog customerId={newProject.value.customerId} status={newProject.value.status} onclose={() => (newProject.value = null)} />
  {/if}
{/if}
<Toasts />
