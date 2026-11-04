<script lang="ts">
  // Color swatches (TAG_PRESETS, 9 per row) plus a free color via the native color input.
  import { t } from '$lib/i18n.svelte';
  import { TAG_PRESETS } from '$lib/tags';
  import Icon from './Icon.svelte';

  let { value = $bindable(''), onpick }: { value?: string; onpick?: (color: string) => void } = $props();

  const custom = $derived(value !== '' && !TAG_PRESETS.includes(value));

  function pick(c: string) {
    value = c;
    onpick?.(c);
  }
</script>

<div class="grid grid-cols-9 gap-1.5" role="radiogroup" aria-label={t('tags.color')}>
  {#each TAG_PRESETS as c (c)}
    <button
      type="button"
      role="radio"
      aria-checked={value === c}
      class="aspect-square w-full rounded-full transition-transform hover:scale-110 {value === c ? 'ring-2 ring-zinc-900 ring-offset-1 dark:ring-white dark:ring-offset-zinc-900' : ''}"
      style:background-color={c}
      onclick={() => pick(c)}
      aria-label={c}
      title={c}></button>
  {/each}
  <label
    class="relative aspect-square w-full cursor-pointer overflow-hidden rounded-full border border-zinc-300 dark:border-zinc-600 {custom ? 'ring-2 ring-zinc-900 ring-offset-1 dark:ring-white dark:ring-offset-zinc-900' : ''}"
    title={t('tags.customColor')}
    style:background-color={custom ? value : undefined}>
    {#if !custom}<Icon name="plus" size={12} class="absolute inset-0 m-auto text-zinc-500" />{/if}
    <input
      type="color"
      class="absolute inset-0 cursor-pointer opacity-0"
      value={value || '#2563eb'}
      onchange={(e) => pick(e.currentTarget.value)}
      aria-label={t('tags.customColor')} />
  </label>
</div>
