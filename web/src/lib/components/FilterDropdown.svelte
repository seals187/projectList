<script lang="ts" module>
  export type FilterId = number | string;
  export interface FilterItem {
    id: FilterId;
    label: string;
    color?: string; // colored dot (labels)
    code?: { text: string; cls: string }; // customer code
    avatar?: boolean; // person: avatar from label + id
  }

  /** Text after "Label:" on a filter button. */
  export function filterSummary(items: FilterItem[], value: FilterId[] | null, all: string, none: string): string {
    if (value === null) return all;
    if (value.length === 0) return none;
    if (value.length === 1) return items.find((i) => i.id === value[0])?.label ?? '1';
    return String(value.length);
  }
</script>

<script lang="ts">
  // Filter button "Label: Value ▾" with a dropdown of checkable entries
  // (customer, label, person responsible, …). value null = all entries (no filter), [] = none.
  import { t } from '$lib/i18n.svelte';
  import ContextMenu from './ContextMenu.svelte';
  import FilterOptions from './FilterOptions.svelte';
  import Icon from './Icon.svelte';

  let {
    label,
    items,
    value,
    onchange,
    none = false
  }: {
    label: string;
    items: FilterItem[];
    value: FilterId[] | null;
    onchange: (next: FilterId[] | null) => void;
    none?: boolean; // offer "None" (show nothing)
  } = $props();

  let menu = $state<{ x: number; y: number } | null>(null);
  const active = $derived(value !== null);
  const summary = $derived(filterSummary(items, value, t('filter.all'), t('filter.none')));

  function open(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = menu ? null : { x: r.left, y: r.bottom + 4 };
  }
</script>

<button type="button" class="ph-ctl max-w-56 {active ? 'ph-ctl-active' : ''}" aria-haspopup="menu" aria-expanded={!!menu} onclick={open}>
  <span class="truncate"><span class={active ? '' : 'text-zinc-400 dark:text-zinc-500'}>{label}:</span> <span class="font-medium">{summary}</span></span>
  <Icon name="chevronDown" size={13} class="ph-muted-icon" />
</button>

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} {label} onclose={() => (menu = null)}>
    <FilterOptions {items} {value} {onchange} {none} />
  </ContextMenu>
{/if}
