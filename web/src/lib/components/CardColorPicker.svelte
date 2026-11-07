<script lang="ts">
  // Overview card colors (see $lib/cardColors): one row light, one row strong
  // (darker in the light theme, lighter in the dark theme) plus "none". Each
  // swatch is a small preview of the tinted card.
  import { t } from '$lib/i18n.svelte';
  import { CARD_COLORS, CARD_HEX, STRONG, tintStyle } from '$lib/cardColors';
  import { cardColorLabel } from '$lib/cardColorLabel';
  import Icon from './Icon.svelte';

  let { value, onpick, label }: { value: string; onpick: (color: string) => void; label: string } = $props();

  const rows = [
    { key: 'light', suffix: '' },
    { key: 'strong', suffix: STRONG }
  ] as const;
</script>

<div class="space-y-1.5 p-1" role="radiogroup" aria-label={label} data-testid="card-color-picker">
  {#each rows as row (row.key)}
    <div class="grid w-max grid-cols-7 gap-1.5">
      {#if row.key === 'light'}
        <button
          type="button"
          role="radio"
          aria-checked={!value}
          title={t('color.none')}
          aria-label={t('color.none')}
          class="flex size-6 items-center justify-center rounded-md border border-zinc-300 text-zinc-400 hover:scale-110 dark:border-zinc-600 {!value ? 'ring-2 ring-accent-500 ring-offset-1 ring-offset-white dark:ring-offset-zinc-900' : ''}"
          onclick={() => onpick('')}>
          <Icon name="x" size={12} />
        </button>
      {:else}
        <span class="size-6" aria-hidden="true"></span>
      {/if}
      {#each CARD_COLORS as c (c)}
        {@const v = c + row.suffix}
        <button
          type="button"
          role="radio"
          aria-checked={value === v}
          title={cardColorLabel(v)}
          aria-label={cardColorLabel(v)}
          class="card-tint size-6 rounded-md border transition-transform hover:scale-110 {value === v ? 'ring-2 ring-offset-1 ring-offset-white dark:ring-offset-zinc-900' : ''}"
          style="{tintStyle(v)};--tw-ring-color:{CARD_HEX[c]}"
          onclick={() => onpick(v)}></button>
      {/each}
    </div>
  {/each}
</div>
