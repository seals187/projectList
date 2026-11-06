<script lang="ts" generics="V extends string | number | boolean">
  // Segmented control (gray track, selected option as white pill) for the settings.
  import Icon from './Icon.svelte';

  type IconName = import('svelte').ComponentProps<typeof Icon>['name'];
  let {
    options,
    value,
    onchange,
    label
  }: { options: { value: V; label: string; icon?: IconName }[]; value: V; onchange: (v: V) => void; label: string } = $props();
</script>

<div class="inline-flex gap-0.5 rounded-[11px] bg-[#f1f2f5] p-1 dark:bg-zinc-800" role="radiogroup" aria-label={label}>
  {#each options as o (o.value)}
    {@const on = o.value === value}
    <button
      type="button"
      role="radio"
      aria-checked={on}
      class="inline-flex h-8 items-center gap-1.5 rounded-[8px] px-3 text-[13.5px] whitespace-nowrap transition-colors {on
        ? 'bg-white font-semibold text-zinc-900 shadow-sm dark:bg-zinc-600 dark:text-white'
        : 'font-medium text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100'}"
      onclick={() => onchange(o.value)}>
      {#if o.icon}<Icon name={o.icon} size={15} />{/if}{o.label}
    </button>
  {/each}
</div>
