<script lang="ts">
  // Bar above every page: back / forward and the breadcrumb of the current
  // page (with type icons). A long press on ← or Alt+← opens the history
  // ("Back to"), which can also be used with the arrow keys and Enter.
  import { page } from '$app/state';
  import { t } from '$lib/i18n.svelte';
  import { sidebar } from '$lib/state.svelte';
  import { backList, crumbsFor, describe, type Described, type NavTexts } from '$lib/navHistory';
  import { clearNav, navHist } from '$lib/navHistory.svelte';
  import { ensureTemplates, templatesIndex } from '$lib/templatesIndex.svelte';
  import { entityLabelKey } from '$lib/entityTypes';
  import { SETTINGS_NAV } from '$lib/settingsNav';
  import ContextMenu from './ContextMenu.svelte';
  import EntityIcon from './EntityIcon.svelte';
  import Icon from './Icon.svelte';

  let { glass = false }: { glass?: boolean } = $props();

  type TKey = Parameters<typeof t>[0];
  const texts = $derived<NavTexts>({
    overview: t('nav.overview'),
    myTasks: t('nav.myTasks'),
    customers: t('nav.customers'),
    templates: t('nav.templates'),
    settings: t('nav.settings'),
    view: (v) => (['cards', 'board', 'list', 'calendar', 'timeline'].includes(v) ? t(`overview.view.${v}` as TKey) : null),
    settingsPage: (slug) => {
      const it = SETTINGS_NAV.find((x) => x.href === `/settings/${slug}`);
      return it ? t(it.label as TKey) : null;
    }
  });
  const data = $derived({ customers: sidebar.customers, templates: templatesIndex.list });

  const crumbs = $derived(crumbsFor(page.url.pathname, page.url.search, data, texts));
  // history menu: all earlier pages (main pages and settings sub pages included)
  const back = $derived(backList(navHist.h));
  const canBack = $derived(navHist.h.index > 0);
  const canForward = $derived(navHist.h.index < navHist.h.entries.length - 1);

  // template names for the breadcrumb and the history
  $effect(() => {
    const ids = [page.url.pathname, ...back.map((b) => b.entry.path)].map((p) => p.match(/^\/templates\/(\d+)/)).filter((m) => !!m).map((m) => Number(m![1]));
    ensureTemplates(ids);
  });

  const typeText = (d: Described) => (d.type ? t(entityLabelKey(d.type) as TKey) : t('nav.page'));

  // ---- history menu --------------------------------------------------------------------
  let backBtn = $state<HTMLButtonElement>();
  let menu = $state<{ x: number; y: number } | null>(null);
  let active = $state(0);
  function openMenu() {
    if (!back.length || !backBtn) return;
    const r = backBtn.getBoundingClientRect();
    menu = { x: r.left, y: r.bottom + 6 };
    active = 0;
  }
  function go(delta: number) {
    menu = null;
    history.go(delta);
  }

  // long press on ← opens the history; a short click goes back
  let timer: ReturnType<typeof setTimeout> | undefined;
  let longPressed = false;
  function pressStart(e: PointerEvent) {
    if (e.button !== 0) return;
    longPressed = false;
    timer = setTimeout(() => {
      longPressed = true;
      openMenu();
    }, 450);
  }
  const pressEnd = () => clearTimeout(timer);
  function clickBack() {
    if (longPressed) {
      longPressed = false;
      return;
    }
    if (canBack) history.back();
  }

  function onKey(e: KeyboardEvent) {
    if (e.altKey && e.key === 'ArrowLeft') {
      // Alt+←: history menu (again: next entry) instead of the browser's back
      e.preventDefault();
      if (menu) active = Math.min(back.length - 1, active + 1);
      else openMenu();
      return;
    }
    if (!menu) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      active = Math.min(back.length - 1, active + 1);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      active = Math.max(0, active - 1);
    } else if (e.key === 'Enter' && back[active]) {
      e.preventDefault();
      go(back[active].delta);
    }
  }
</script>

<svelte:window onkeydown={onKey} />

<div
  class="flex h-11 shrink-0 items-center gap-3 border-b px-3 sm:px-4 {glass
    ? 'glass-bar dark border-white/10 bg-zinc-950/45 text-zinc-100 backdrop-blur-lg'
    : 'border-zinc-200 bg-[#f9f9fb] dark:border-zinc-800 dark:bg-zinc-900'}"
  data-testid="back-bar">
  <div class="flex shrink-0 overflow-hidden rounded-lg border border-zinc-200 bg-white dark:border-zinc-700 dark:bg-zinc-800/60">
    <button
      bind:this={backBtn}
      type="button"
      class="flex size-7 items-center justify-center text-zinc-600 hover:bg-zinc-100 disabled:opacity-35 dark:text-zinc-300 dark:hover:bg-zinc-700"
      disabled={!canBack}
      onpointerdown={pressStart}
      onpointerup={pressEnd}
      onpointerleave={pressEnd}
      onclick={clickBack}
      oncontextmenu={(e) => (e.preventDefault(), openMenu())}
      title="{t('nav.back')} (Alt+←: {t('nav.history')})"
      aria-label={t('nav.back')}
      aria-haspopup="menu"
      aria-expanded={!!menu}>
      <Icon name="chevron" size={15} class="rotate-180" />
    </button>
    <button
      type="button"
      class="flex size-7 items-center justify-center border-l border-zinc-200 text-zinc-600 hover:bg-zinc-100 disabled:opacity-35 dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-700"
      disabled={!canForward}
      onclick={() => history.forward()}
      title={t('nav.forward')}
      aria-label={t('nav.forward')}>
      <Icon name="chevron" size={15} />
    </button>
  </div>

  <nav class="flex min-w-0 flex-1 items-center gap-1.5 text-[13px]" aria-label="Breadcrumb" data-testid="breadcrumb">
    {#each crumbs as c, i (i)}
      {@const last = i === crumbs.length - 1}
      {#if i > 0}<span class="text-zinc-300 dark:text-zinc-600" aria-hidden="true">/</span>{/if}
      <span class="flex min-w-0 items-center gap-1.5 {last ? 'shrink-0 font-semibold' : 'text-zinc-500 dark:text-zinc-400'}">
        {#if c.type}<EntityIcon type={c.type} size="xs" />{:else if c.icon}<Icon name={c.icon} size={14} class="shrink-0 text-zinc-400" />{/if}
        {#if c.href && !last}<a href={c.href} class="truncate hover:text-zinc-900 hover:underline dark:hover:text-zinc-100">{c.label}</a>
        {:else}<span class="truncate" aria-current={last ? 'page' : undefined}>{c.label}</span>{/if}
      </span>
    {/each}
  </nav>

  <span class="hidden shrink-0 items-center gap-1 text-[11px] text-zinc-400 md:flex" aria-hidden="true">
    <kbd class="rounded border border-zinc-200 bg-white px-1 font-sans dark:border-zinc-700 dark:bg-zinc-800">Alt</kbd>+<kbd class="rounded border border-zinc-200 bg-white px-1 font-sans dark:border-zinc-700 dark:bg-zinc-800">←</kbd>
    {t('nav.history')}
  </span>
</div>

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} label={t('nav.backTo')} onclose={() => (menu = null)}>
    <div class="flex items-center justify-between px-2 pt-0.5 pb-1">
      <p class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('nav.backTo')}</p>
      <button
        type="button"
        class="rounded p-1 text-zinc-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/40 dark:hover:text-red-400"
        onclick={() => ((menu = null), clearNav())}
        title={t('nav.clearHistory')}
        aria-label={t('nav.clearHistory')}
        data-testid="history-clear">
        <Icon name="trash" size={14} />
      </button>
    </div>
    {#each back as b, i (b.delta)}
      {@const d = describe(b.entry.path, b.entry.search, data, texts)}
      <button
        type="button"
        role="menuitem"
        class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left {i === active ? 'bg-accent-500/12' : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'}"
        onmouseenter={() => (active = i)}
        onclick={() => go(b.delta)}
        data-testid="history-item">
        {#if d.type}<EntityIcon type={d.type} size="sm" />{:else}<span class="flex size-[22px] items-center justify-center"><Icon name={d.icon ?? 'home'} size={15} class="text-zinc-500" /></span>{/if}
        <span class="min-w-0 flex-1 truncate">{d.label}</span>
        <span class="shrink-0 text-[11px] text-zinc-400">{typeText(d)}</span>
      </button>
    {/each}
    <p class="mt-1 border-t border-zinc-100 px-2 pt-1.5 pb-0.5 text-[11px] text-zinc-400 dark:border-zinc-800">{t('nav.historyHint')}</p>
  </ContextMenu>
{/if}
