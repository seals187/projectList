<script lang="ts" module>
  import type { ComponentProps, Snippet } from 'svelte';
  import type Icon from './Icon.svelte';
  import type { FilterId, FilterItem } from './FilterDropdown.svelte';

  export type IconName = ComponentProps<typeof Icon>['name'];
  export interface HeaderView {
    key: string;
    label: string;
    icon: IconName;
    count?: number;
  }
  /** Multi-select filter (checkable entries) or single choice (e.g. "Group by"). */
  export type HeaderFilter =
    | { key: string; label: string; items: FilterItem[]; value: FilterId[] | null; onchange: (v: FilterId[] | null) => void; none?: boolean }
    | { key: string; label: string; options: { value: string; label: string }[]; value: string; defaultValue?: string; onchange: (v: string) => void };
  export interface QuickFilter {
    key: string;
    label: string;
    count?: number;
    active: boolean;
    color?: string; // colored square before the label (e.g. template categories)
  }
  export interface HeaderProps {
    icon: IconName;
    title: string;
    subtitle?: string | Snippet;
    views?: HeaderView[];
    activeView?: string;
    onViewChange?: (key: string) => void;
    primaryAction?: { label: string; onClick: (e: MouseEvent) => void; icon?: IconName }; // icon: default "plus"
    extraActions?: { label: string; icon: IconName; onClick: (e: MouseEvent) => void; testid?: string }[]; // buttons left of sorting
    reservePrimary?: boolean; // views with and without primary action: keep its space (stable layout)
    searchPlaceholder: string;
    searchValue: string;
    noSearch?: boolean; // page without free-text search (e.g. reporting)
    onSearch: (q: string) => void;
    filters?: HeaderFilter[];
    quickFilters?: QuickFilter[];
    onQuickFilter?: (key: string) => void;
    onSaveView?: (name: string) => unknown;
    sortOptions?: { value: string; label: string }[];
    activeSort?: string;
    onSortChange?: (v: string) => void;
    collapsed?: boolean;
    onToggleCollapse?: () => void; // without it there is no expand/collapse button
    collapseDisabled?: boolean;
    displayOptions?: { key: string; label: string; value: boolean }[]; // "Display" popover (per page)
    onDisplayChange?: (key: string, value: boolean) => void;
  }

  /** "Freitag, 06.11.2026": weekday in the UI language, date always dd.mm.yyyy. */
  export function headerDate(d: Date, loc: string): string {
    const wd = d.toLocaleDateString(loc, { weekday: 'long' });
    const dd = String(d.getDate()).padStart(2, '0');
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    return `${wd.charAt(0).toUpperCase()}${wd.slice(1)}, ${dd}.${mm}.${d.getFullYear()}`;
  }

  /** Is a filter active (differs from "all" / its default)? */
  export const filterActive = (f: HeaderFilter) => ('items' in f ? f.value !== null : f.defaultValue !== undefined && f.value !== f.defaultValue);
</script>

<script lang="ts">
  // The one header of the main pages (overview, my tasks, customers, templates):
  // a gray card with
  //   row 1: page icon, title + subtitle … view switcher, primary action
  //   row 2 (white strip): search | filter icon, filters "Name: Value ▾", quick
  //          filter chips, "+ Save view" … sorting, expand/collapse toggle.
  // Nothing wraps. When space runs out: 1. filters merge into "Filters (n)" (and the
  // search field gets narrower, chips lose their counts, "Save view" its label),
  // 2. the view switcher shows icons only, 3. the strip scrolls horizontally.
  // Sizes and colors: ph-* utilities in app.css.
  import { t } from '$lib/i18n.svelte';
  import IconCmp from './Icon.svelte';
  import FilterDropdown from './FilterDropdown.svelte';
  import FilterOptions from './FilterOptions.svelte';
  import SelectDropdown from './SelectDropdown.svelte';
  import ContextMenu from './ContextMenu.svelte';

  let {
    icon,
    title,
    subtitle,
    views = [],
    activeView,
    onViewChange,
    primaryAction,
    reservePrimary = false,
    extraActions = [],
    searchPlaceholder,
    searchValue,
    noSearch = false,
    onSearch,
    filters = [],
    quickFilters = [],
    onQuickFilter,
    onSaveView,
    sortOptions = [],
    activeSort,
    onSortChange,
    collapsed,
    onToggleCollapse,
    collapseDisabled = false,
    displayOptions = [],
    onDisplayChange
  }: HeaderProps = $props();

  // ---- fitting (no wrapping) -------------------------------------------------------
  let row1 = $state<HTMLElement>();
  let right1 = $state<HTMLElement>();
  let row2 = $state<HTMLElement>();
  let left2 = $state<HTMLElement>();
  let right2 = $state<HTMLElement>();
  let compactFilters = $state(false);
  let iconViews = $state(false);
  let fullLeft2 = 0; // width of the left group with all filters shown
  let fullRight1 = 0; // width of views + action with labels
  const TITLE_MIN = 220; // px kept for icon + title

  function fit() {
    if (row2 && left2 && right2) {
      const avail = row2.clientWidth - 16;
      if (!compactFilters) {
        fullLeft2 = left2.offsetWidth;
        if (fullLeft2 + right2.offsetWidth + 16 > avail && (filters.length > 0 || quickFilters.length > 0 || !!onSaveView)) compactFilters = true;
      } else if (fullLeft2 + right2.offsetWidth + 16 <= avail) compactFilters = false;
    }
    if (row1 && right1) {
      const avail = row1.clientWidth - TITLE_MIN;
      if (!iconViews) {
        fullRight1 = right1.offsetWidth;
        if (fullRight1 > avail && views.length) iconViews = true;
      } else if (fullRight1 <= avail) iconViews = false;
    }
  }
  // other chips or filters (e.g. another view): measure the full width again
  let lastStrip = '';
  $effect(() => {
    const key = [...quickFilters.map((c) => c.key), ...filters.map((f) => f.key)].join('|');
    if (key !== lastStrip) {
      lastStrip = key;
      compactFilters = false;
    }
  });
  $effect(() => {
    const els = [row1, right1, row2, left2, right2].filter((e): e is HTMLElement => !!e);
    if (typeof ResizeObserver === 'undefined' || !els.length) return;
    const ro = new ResizeObserver(() => fit());
    els.forEach((e) => ro.observe(e));
    return () => ro.disconnect();
  });

  // ---- menus ---------------------------------------------------------------------------
  let filterMenu = $state<{ x: number; y: number } | null>(null);
  let saveAt = $state<{ x: number; y: number } | null>(null);
  let displayAt = $state<{ x: number; y: number } | null>(null);
  let viewName = $state('');
  const below = (e: MouseEvent) => {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    return { x: r.left, y: r.bottom + 4 };
  };
  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!viewName.trim() || !onSaveView) return;
    await onSaveView(viewName.trim());
    viewName = '';
    saveAt = null;
  }
  const activeCount = $derived(filters.filter(filterActive).length);
</script>

<section class="ph-card surface" aria-label={title} data-testid="page-header">
  <!-- row 1: icon, title, subtitle … views, primary action -->
  <div bind:this={row1} class="flex items-center gap-3 whitespace-nowrap">
    <span class="ph-icon"><IconCmp name={icon} size={18} /></span>
    <div class="min-w-0 flex-1">
      <h1 class="ph-title">{title}</h1>
      {#if subtitle}
        <p class="ph-subtitle">{#if typeof subtitle === 'string'}{subtitle}{:else}{@render subtitle()}{/if}</p>
      {/if}
    </div>
    <div bind:this={right1} class="flex shrink-0 items-center gap-2.5">
      {#if views.length}
        <div class="ph-seg" role="group" aria-label={t('ph.views')}>
          {#each views as v (v.key)}
            <button
              type="button"
              class="ph-seg-btn {activeView === v.key ? 'ph-seg-on' : ''}"
              aria-pressed={activeView === v.key}
              title={iconViews ? v.label : undefined}
              aria-label={iconViews ? v.label : undefined}
              onclick={() => onViewChange?.(v.key)}>
              <IconCmp name={v.icon} size={14} class={activeView === v.key ? 'text-accent-600 dark:text-accent-300' : ''} />
              {#if !iconViews}
                <!-- the bold (active) width is always reserved: the switcher keeps its size -->
                <span class="inline-grid"><span class="invisible col-start-1 row-start-1 font-semibold" aria-hidden="true">{v.label}</span><span class="col-start-1 row-start-1">{v.label}</span></span>
              {/if}
              {#if v.count !== undefined}<span class="text-[11px] font-normal text-zinc-400 tabular-nums">{v.count}</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
      {#if primaryAction}
        <button type="button" class="ph-primary {reservePrimary ? 'min-w-32 justify-center' : ''}" onclick={primaryAction.onClick}><IconCmp name={primaryAction.icon ?? 'plus'} size={15} />{primaryAction.label}</button>
      {:else if reservePrimary}
        <span class="min-w-32" aria-hidden="true"></span>
      {/if}
    </div>
  </div>

  <!-- row 2: search | filters, chips, save view … sorting, toggle -->
  <div bind:this={row2} class="ph-strip" role="toolbar" aria-label={t('ph.sort')}>
    <div bind:this={left2} class="flex shrink-0 items-center gap-2">
      {#if !noSearch}
      <label class="flex h-8 items-center gap-2 pr-1 pl-2">
        <IconCmp name="search" size={14} class="ph-muted-icon" />
        <input
          type="search"
          class="{compactFilters ? 'w-24' : 'w-40'} bg-transparent text-[12.5px] outline-none placeholder:text-zinc-400"
          placeholder={searchPlaceholder}
          aria-label={searchPlaceholder}
          value={searchValue}
          oninput={(e) => onSearch(e.currentTarget.value)} />
      </label>
      {#if filters.length || quickFilters.length || onSaveView}
        <span class="h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700" aria-hidden="true"></span>
      {/if}
      {/if}
      {#if filters.length}
        <IconCmp name="filter" size={14} class="ph-muted-icon mx-0.5" />
        {#if compactFilters}
          <button type="button" class="ph-ctl {activeCount ? 'ph-ctl-active' : ''}" aria-haspopup="menu" onclick={(e) => (filterMenu = filterMenu ? null : below(e))}>
            {t('ph.filters', { n: activeCount })}
            <IconCmp name="chevronDown" size={13} class="ph-muted-icon" />
          </button>
        {:else}
          {#each filters as f (f.key)}
            {#if 'items' in f}
              <FilterDropdown label={f.label} items={f.items} value={f.value} onchange={f.onchange} none={f.none} />
            {:else}
              <SelectDropdown label={f.label} options={f.options} value={f.value} defaultValue={f.defaultValue} onchange={f.onchange} />
            {/if}
          {/each}
        {/if}
      {/if}
      {#each quickFilters as c (c.key)}
        <button type="button" class="ph-chip {c.active ? 'ph-chip-on' : 'ph-chip-off'}" aria-pressed={c.active} onclick={() => onQuickFilter?.(c.key)}>
          {#if c.color}<span class="size-2 shrink-0 rounded-sm" style:background-color={c.color}></span>{/if}{c.label}{#if c.count !== undefined && !compactFilters}<span class="tabular-nums opacity-70">{c.count}</span>{/if}
        </button>
      {/each}
      {#if onSaveView}
        <button
          type="button"
          class="ph-ctl border-dashed! text-zinc-500 dark:text-zinc-400"
          onclick={(e) => (saveAt = saveAt ? null : below(e))}
          title={t('views.save')}
          aria-label={t('views.save')}>
          <IconCmp name="plus" size={13} />{#if !compactFilters}{t('views.save')}{/if}
        </button>
      {/if}
    </div>
    <div class="min-w-4 flex-1"></div>
    <div bind:this={right2} class="flex shrink-0 items-center gap-2">
      {#each extraActions as a (a.label)}
        <button type="button" class="ph-ctl" onclick={a.onClick} data-testid={a.testid}><IconCmp name={a.icon} size={14} class="ph-muted-icon" />{a.label}</button>
      {/each}
      {#if sortOptions.length && activeSort !== undefined}
        <SelectDropdown label={t('ph.sort')} options={sortOptions} value={activeSort} onchange={(v) => onSortChange?.(v)} icon="arrowUpDown" bare />
      {/if}
      {#if onToggleCollapse}
        <button type="button" class="ph-ctl disabled:cursor-default disabled:opacity-50" disabled={collapseDisabled} onclick={onToggleCollapse} aria-pressed={collapsed}>
        <IconCmp name={collapsed ? 'chevronsUpDown' : 'chevronsDownUp'} size={14} class="ph-muted-icon" />
        {t(collapsed ? 'ph.expandAll' : 'ph.collapseAll')}
        </button>
      {/if}
      <button
        type="button"
        class="ph-ctl disabled:cursor-default disabled:opacity-50 {displayAt ? 'ph-ctl-active' : ''}"
        disabled={!displayOptions.length}
        aria-haspopup="menu"
        aria-expanded={!!displayAt}
        onclick={(e) => {
          const r = e.currentTarget.getBoundingClientRect();
          displayAt = displayAt ? null : { x: r.right - 288, y: r.bottom + 4 };
        }}>
        <IconCmp name="columns" size={14} class={displayAt ? 'text-accent-600 dark:text-accent-300' : 'ph-muted-icon'} />
        {t('ph.display')}
      </button>
    </div>
  </div>
</section>

{#if filterMenu}
  <ContextMenu x={filterMenu.x} y={filterMenu.y} label={t('ph.filters', { n: activeCount })} onclose={() => (filterMenu = null)}>
    {#each filters as f (f.key)}
      <p class="px-2 pt-2 pb-1 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{f.label}</p>
      {#if 'items' in f}
        <FilterOptions items={f.items} value={f.value} onchange={f.onchange} none={f.none} searchable={false} />
      {:else}
        {#each f.options as o (o.value)}
          <button
            type="button"
            role="menuitemradio"
            aria-checked={o.value === f.value}
            class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-zinc-200/60 dark:hover:bg-zinc-800"
            onclick={() => f.onchange(o.value)}>
            <span class="flex size-4 items-center justify-center text-accent-600">{#if o.value === f.value}<IconCmp name="check" size={13} />{/if}</span>
            {o.label}
          </button>
        {/each}
      {/if}
    {/each}
  </ContextMenu>
{/if}

{#if displayAt}
  <ContextMenu x={displayAt.x} y={displayAt.y} label={t('ph.display')} onclose={() => (displayAt = null)}>
    <p class="px-2 pt-1 pb-1.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('ph.display')}</p>
    {#each displayOptions as o (o.key)}
      <button
        type="button"
        role="switch"
        aria-checked={o.value}
        class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-zinc-200/60 dark:hover:bg-zinc-800"
        onclick={() => onDisplayChange?.(o.key, !o.value)}>
        <span class="relative inline-flex h-[18px] w-8 shrink-0 items-center rounded-full transition-colors {o.value ? 'bg-[var(--accent-primary)]' : 'bg-zinc-300 dark:bg-zinc-600'}">
          <span class="absolute size-3.5 rounded-full bg-white shadow-sm transition-[left] {o.value ? 'left-4' : 'left-0.5'}"></span>
        </span>
        <span class="min-w-0 flex-1 truncate">{o.label}</span>
      </button>
    {/each}
    <p class="mt-1.5 border-t border-zinc-200/80 px-2 pt-2 pb-0.5 text-xs text-zinc-400 dark:border-zinc-700">{t('ph.displayHint')}</p>
  </ContextMenu>
{/if}

{#if saveAt}
  <ContextMenu x={saveAt.x} y={saveAt.y} label={t('views.save')} onclose={() => (saveAt = null)}>
    <form onsubmit={save} class="space-y-2 p-1.5">
      <p class="text-xs text-zinc-500">{t('views.saveHint')}</p>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input py-1.5! text-sm!" maxlength="60" placeholder={t('views.namePlaceholder')} bind:value={viewName} autofocus aria-label={t('views.name')} />
      <button type="submit" class="btn-primary w-full justify-center" disabled={!viewName.trim()}>{t('common.save')}</button>
    </form>
  </ContextMenu>
{/if}
