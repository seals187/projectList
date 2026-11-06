<script lang="ts">
  // Frame of all settings pages: left the settings navigation (search, groups
  // "Personal" and "Workspace"), right the page with title, subtitle and
  // optional actions; optionally a third column (`aside`, e.g. live preview or
  // user details) and a footer (e.g. "Changes are saved automatically").
  // Each column scrolls on its own; below md everything stacks and scrolls with the page.
  import type { Snippet } from 'svelte';
  import { page } from '$app/state';
  import { t } from '$lib/i18n.svelte';
  import { isAdmin } from '$lib/state.svelte';
  import { activeNav, filterNav, SETTINGS_NAV } from '$lib/settingsNav';
  import Icon from './Icon.svelte';

  let {
    title,
    subtitle = '',
    actions,
    aside,
    asideClass = '',
    footer,
    children
  }: { title: string; subtitle?: string; actions?: Snippet; aside?: Snippet; asideClass?: string; footer?: Snippet; children: Snippet } = $props();

  type TKey = Parameters<typeof t>[0];
  const tr = (k: string) => t(k as TKey);

  let query = $state('');
  const items = $derived(filterNav(SETTINGS_NAV, query, isAdmin(), tr));
  const bottom = $derived(items.filter((i) => i.bottom));
  const groups = $derived([
    { label: t('set.group.personal'), items: items.filter((i) => !i.admin && !i.bottom) },
    { label: t('set.group.workspace'), items: items.filter((i) => i.admin) }
  ]);
  const active = $derived(activeNav(SETTINGS_NAV, page.url.pathname, page.url.hash));

  // jump to an anchor (e.g. #checklists) after navigation and on hash changes
  $effect(() => {
    const id = page.url.hash.slice(1);
    if (!id) return;
    requestAnimationFrame(() => document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' }));
  });
</script>

<svelte:head><title>{title} · {t('nav.settings')} · ProjectList</title></svelte:head>

<div class="flex h-full flex-col md:flex-row">
  <!-- settings navigation -->
  <nav class="flex shrink-0 flex-col border-b border-zinc-200/80 bg-[#f9f9fb] px-4 pt-6 pb-4 md:w-60 md:overflow-y-auto md:border-r md:border-b-0 dark:border-zinc-800 dark:bg-zinc-950/30" aria-label={t('nav.settings')}>
    <h1 class="px-2 text-[22px] font-bold tracking-tight">{t('nav.settings')}</h1>
    <label class="mt-4 flex h-9 items-center gap-2 rounded-[10px] border border-zinc-200 bg-white px-3 text-zinc-400 focus-within:border-accent-400 focus-within:ring-2 focus-within:ring-accent-500/15 dark:border-zinc-700 dark:bg-zinc-900">
      <Icon name="search" size={14} />
      <input class="min-w-0 flex-1 bg-transparent text-[13.5px] text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100" placeholder={t('set.search')} bind:value={query} aria-label={t('set.search')} />
    </label>
    {#each groups as g (g.label)}
      {#if g.items.length}
        <p class="mt-6 mb-1.5 px-2 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{g.label}</p>
        <ul class="flex flex-wrap gap-0.5 md:block md:space-y-0.5">
          {#each g.items as it (it.id)}
            <li>
              <a
                href={it.href}
                class="flex h-[34px] items-center gap-2.5 rounded-lg px-2.5 text-[14px] transition-colors {active === it.id
                  ? 'bg-accent-500/12 font-semibold text-accent-700 dark:bg-accent-400/20 dark:text-accent-200'
                  : 'text-zinc-700 hover:bg-zinc-200/50 dark:text-zinc-300 dark:hover:bg-zinc-800'}"
                aria-current={active === it.id ? 'page' : undefined}>
                <Icon name={it.icon} size={16} class={active === it.id ? '' : 'text-zinc-500 dark:text-zinc-400'} />
                {tr(it.label)}
              </a>
            </li>
          {/each}
        </ul>
      {/if}
    {/each}
    {#if !items.length}<p class="mt-6 px-2 text-sm text-zinc-400">{t('set.noMatch')}</p>{/if}
    <!-- at the very bottom: About -->
    {#if bottom.length}
      <ul class="mt-6 border-t border-zinc-200/80 pt-3 md:mt-auto dark:border-zinc-800">
        {#each bottom as it (it.id)}
          <li>
            <a
              href={it.href}
              class="flex h-[34px] items-center gap-2.5 rounded-lg px-2.5 text-[14px] transition-colors {active === it.id
                ? 'bg-accent-500/12 font-semibold text-accent-700 dark:bg-accent-400/20 dark:text-accent-200'
                : 'text-zinc-700 hover:bg-zinc-200/50 dark:text-zinc-300 dark:hover:bg-zinc-800'}"
              aria-current={active === it.id ? 'page' : undefined}>
              <Icon name={it.icon} size={16} class={active === it.id ? '' : 'text-zinc-500 dark:text-zinc-400'} />
              {tr(it.label)}
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  </nav>

  <!-- page -->
  <div class="flex min-w-0 flex-1 flex-col md:min-h-0">
    <div class="flex-1 px-5 pt-7 pb-10 sm:px-8 md:overflow-y-auto">
      <header class="mb-6 flex flex-wrap items-start gap-4">
        <div class="min-w-0 flex-1">
          <h2 class="text-[26px] leading-tight font-bold tracking-tight">{title}</h2>
          {#if subtitle}<p class="mt-1 text-[14.5px] text-zinc-500 dark:text-zinc-400">{subtitle}</p>{/if}
        </div>
        {#if actions}<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>{/if}
      </header>
      {@render children()}
    </div>
    {#if footer}
      <div class="shrink-0 border-t border-zinc-200/80 px-5 py-3 sm:px-8 dark:border-zinc-800">{@render footer()}</div>
    {/if}
  </div>

  {#if aside}
    <aside class="shrink-0 border-t border-zinc-200/80 md:w-[340px] md:overflow-y-auto md:border-t-0 md:border-l lg:w-[380px] dark:border-zinc-800 {asideClass}">
      {@render aside()}
    </aside>
  {/if}
</div>
