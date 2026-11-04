<script lang="ts" module>
  export interface FilterItem {
    id: number;
    label: string;
    color?: string; // colored dot (labels)
    code?: { text: string; cls: string }; // customer code
    avatar?: boolean; // person: avatar from label + id
  }
</script>

<script lang="ts">
  // Filter button with a dropdown of checkable entries (customer, label, person
  // responsible). value null = all entries (no filter), [] = none.
  import { t } from '$lib/i18n.svelte';
  import ContextMenu from './ContextMenu.svelte';
  import Avatar from './Avatar.svelte';
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
    value: number[] | null;
    onchange: (next: number[] | null) => void;
    none?: boolean; // offer "None" (show nothing)
  } = $props();

  let menu = $state<{ x: number; y: number } | null>(null);
  let query = $state('');

  const active = $derived(value !== null);
  const shown = $derived(items.filter((i) => !query.trim() || i.label.toLowerCase().includes(query.trim().toLowerCase())));
  const summary = $derived.by(() => {
    if (value === null) return t('filter.all');
    if (value.length === 0) return t('filter.none');
    if (value.length === 1) return items.find((i) => i.id === value[0])?.label ?? '1';
    return String(value.length);
  });

  function toggle(id: number) {
    const cur = value ?? items.map((i) => i.id);
    const next = cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id];
    // everything checked again = no filter
    onchange(next.length === items.length && items.every((i) => next.includes(i.id)) ? null : next);
  }
  const checked = (id: number) => value === null || value.includes(id);

  function open(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    query = '';
    menu = menu ? null : { x: r.left, y: r.bottom + 4 };
  }
</script>

<button
  type="button"
  class="inline-flex max-w-56 items-center gap-1.5 rounded-lg border px-2.5 py-1 text-sm transition-colors
    {active
    ? 'border-accent-400/60 bg-accent-50 text-accent-800 dark:border-accent-500/40 dark:bg-accent-950/40 dark:text-accent-300'
    : 'border-zinc-200 bg-white text-zinc-700 hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-700'}"
  aria-haspopup="menu"
  aria-expanded={!!menu}
  onclick={open}>
  <span class="truncate">{label}: <span class={active ? 'font-semibold' : ''}>{summary}</span></span>
  <Icon name="chevronDown" size={13} class="shrink-0 text-zinc-400" />
</button>

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} {label} onclose={() => (menu = null)}>
    {#if items.length > 8}
      <div class="relative mb-1">
        <Icon name="search" size={13} class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-zinc-400" />
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input py-1! pl-7! text-[13px]!" placeholder={t('filter.search')} bind:value={query} autofocus aria-label={t('filter.search')} />
      </div>
    {/if}
    <div class="mb-1 flex gap-1 border-b border-zinc-100 px-1 pb-1.5 text-xs dark:border-zinc-800">
      <button type="button" class="rounded px-2 py-0.5 {value === null ? 'bg-zinc-100 font-medium dark:bg-zinc-800' : 'text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800'}" onclick={() => onchange(null)}>
        {t('filter.all')}
      </button>
      {#if none}
        <button type="button" class="rounded px-2 py-0.5 {value?.length === 0 ? 'bg-zinc-100 font-medium dark:bg-zinc-800' : 'text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800'}" onclick={() => onchange([])}>
          {t('filter.none')}
        </button>
      {/if}
    </div>
    <div class="max-h-72 overflow-y-auto">
      {#each shown as i (i.id)}
        <div class="group flex items-center rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800">
          <button type="button" role="menuitemcheckbox" aria-checked={checked(i.id)} class="flex min-w-0 flex-1 items-center gap-2 px-2 py-1.5 text-left text-[13px]" onclick={() => toggle(i.id)}>
            <span
              class="flex size-4 shrink-0 items-center justify-center rounded border {checked(i.id)
                ? 'border-accent-600 bg-accent-600 text-white'
                : 'border-zinc-300 dark:border-zinc-600'}">
              {#if checked(i.id)}<Icon name="check" size={11} />{/if}
            </span>
            {#if i.code}<span class="flex size-5 shrink-0 items-center justify-center rounded text-[8px] font-bold {i.code.cls}">{i.code.text}</span>{/if}
            {#if i.avatar}<Avatar name={i.label} id={i.id} size={18} />{/if}
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
  </ContextMenu>
{/if}
