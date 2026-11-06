<script lang="ts">
  // Checkable entries of a filter (menu body): "All" / "None", optional search,
  // entries with "only". value null = all entries (no filter), [] = none.
  // Used by FilterDropdown and by the combined "Filters (n)" menu of PageHeader.
  import { t } from '$lib/i18n.svelte';
  import type { FilterId, FilterItem } from './FilterDropdown.svelte';
  import Avatar from './Avatar.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    items,
    value,
    onchange,
    none = false,
    searchable = true
  }: {
    items: FilterItem[];
    value: FilterId[] | null;
    onchange: (next: FilterId[] | null) => void;
    none?: boolean;
    searchable?: boolean;
  } = $props();

  let query = $state('');
  const shown = $derived(items.filter((i) => !query.trim() || i.label.toLowerCase().includes(query.trim().toLowerCase())));

  function toggle(id: FilterId) {
    const cur = value ?? items.map((i) => i.id);
    const next = cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id];
    // everything checked again = no filter
    onchange(next.length === items.length && items.every((i) => next.includes(i.id)) ? null : next);
  }
  const checked = (id: FilterId) => value === null || value.includes(id);
</script>

{#if searchable && items.length > 8}
  <div class="relative mb-1">
    <Icon name="search" size={13} class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-zinc-400" />
    <input class="input py-1! pl-7! text-[13px]!" placeholder={t('filter.search')} bind:value={query} aria-label={t('filter.search')} />
  </div>
{/if}
<div class="mb-1 flex gap-1 border-b border-zinc-100 px-1 pb-1.5 text-xs dark:border-zinc-800">
  <button type="button" class="rounded px-2 py-0.5 {value === null ? 'bg-zinc-200/70 font-medium dark:bg-zinc-800' : 'text-zinc-500 hover:bg-zinc-200/70 dark:hover:bg-zinc-800'}" onclick={() => onchange(null)}>
    {t('filter.all')}
  </button>
  {#if none}
    <button type="button" class="rounded px-2 py-0.5 {value?.length === 0 ? 'bg-zinc-200/70 font-medium dark:bg-zinc-800' : 'text-zinc-500 hover:bg-zinc-200/70 dark:hover:bg-zinc-800'}" onclick={() => onchange([])}>
      {t('filter.none')}
    </button>
  {/if}
</div>
<div class="max-h-72 overflow-y-auto">
  {#each shown as i (i.id)}
    <div class="group flex items-center rounded-md hover:bg-zinc-200/60 dark:hover:bg-zinc-800">
      <button type="button" role="menuitemcheckbox" aria-checked={checked(i.id)} class="flex min-w-0 flex-1 items-center gap-2 px-2 py-1.5 text-left text-[13px]" onclick={() => toggle(i.id)}>
        <span class="flex size-4 shrink-0 items-center justify-center rounded border {checked(i.id) ? 'border-accent-600 bg-accent-600 text-white' : 'border-zinc-300 dark:border-zinc-600'}">
          {#if checked(i.id)}<Icon name="check" size={11} />{/if}
        </span>
        {#if i.entity}<EntityIcon type={i.entity} size="xs" />{/if}
        {#if i.avatar}<Avatar name={i.label} id={Number(i.id)} size={18} />{/if}
        {#if i.color}<span class="size-2.5 shrink-0 rounded-full" style:background-color={i.color}></span>{/if}
        <span class="min-w-0 flex-1 truncate">{i.label}</span>
      </button>
      <button
        type="button"
        class="mr-1 hidden shrink-0 rounded px-1.5 py-0.5 text-[11px] text-zinc-500 group-hover:block hover:bg-zinc-200 dark:hover:bg-zinc-700"
        onclick={() => onchange([i.id])}
        title={t('filter.only')}>
        {t('filter.only')}
      </button>
    </div>
  {:else}
    <p class="px-2 py-1.5 text-[13px] text-zinc-400">{t('filter.noEntries')}</p>
  {/each}
</div>
