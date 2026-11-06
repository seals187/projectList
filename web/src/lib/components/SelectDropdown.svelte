<script lang="ts" generics="T extends string">
  // Single choice in the page header (e.g. "Group by", sorting): same button and
  // menu as FilterDropdown. bare = no frame, only icon + value (sorting).
  import ContextMenu from './ContextMenu.svelte';
  import Icon from './Icon.svelte';

  let {
    label,
    options,
    value,
    defaultValue,
    onchange,
    icon,
    bare = false
  }: {
    label: string;
    options: { value: T; label: string }[];
    value: T;
    defaultValue?: T; // highlighted like an active filter when the value differs
    onchange: (next: T) => void;
    icon?: 'arrowUpDown';
    bare?: boolean;
  } = $props();

  let menu = $state<{ x: number; y: number } | null>(null);
  const current = $derived(options.find((o) => o.value === value)?.label ?? '');
  const active = $derived(!bare && defaultValue !== undefined && value !== defaultValue);

  function open(e: MouseEvent) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = menu ? null : { x: Math.max(8, r.right - 288), y: r.bottom + 4 };
  }
  function pick(v: T) {
    menu = null;
    if (v !== value) onchange(v);
  }
</script>

<button
  type="button"
  class="ph-ctl max-w-56 {bare ? 'border-transparent! hover:bg-zinc-100 dark:hover:bg-zinc-800' : ''} {active ? 'ph-ctl-active' : ''}"
  aria-haspopup="menu"
  aria-expanded={!!menu}
  aria-label="{label}: {current}"
  title={bare ? label : undefined}
  onclick={open}>
  {#if icon}<Icon name={icon} size={14} class="ph-muted-icon" />{/if}
  {#if bare}
    <span class="truncate font-medium">{current}</span>
  {:else}
    <span class="truncate"><span class={active ? '' : 'text-zinc-400 dark:text-zinc-500'}>{label}:</span> <span class="font-medium">{current}</span></span>
    <Icon name="chevronDown" size={13} class="ph-muted-icon" />
  {/if}
</button>

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} {label} onclose={() => (menu = null)}>
    {#each options as o (o.value)}
      <button
        type="button"
        role="menuitemradio"
        aria-checked={o.value === value}
        class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-[13px] hover:bg-zinc-200/60 dark:hover:bg-zinc-800 {o.value === value ? 'font-medium' : ''}"
        onclick={() => pick(o.value)}>
        <span class="flex size-4 shrink-0 items-center justify-center text-accent-600 dark:text-accent-400">
          {#if o.value === value}<Icon name="check" size={13} />{/if}
        </span>
        <span class="min-w-0 flex-1 truncate">{o.label}</span>
      </button>
    {/each}
  </ContextMenu>
{/if}
