<script lang="ts">
  import { t } from '$lib/i18n.svelte';

  let {
    done,
    total,
    width = 'w-20',
    label = true,
    compact = false
  }: { done: number; total: number; width?: string; label?: boolean; compact?: boolean } = $props();
  const pct = $derived(total ? Math.round((done / total) * 100) : 0);
</script>

<span class="inline-flex items-center gap-2.5" title="{pct} %">
  <span class="h-1 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800 {width}">
    <span
      class="block h-full rounded-full transition-[width] duration-300 {pct === 100 ? 'bg-emerald-500' : 'bg-accent-500'}"
      style:width="{pct}%"></span>
  </span>
  {#if label}
    <span class="text-xs whitespace-nowrap text-zinc-500 tabular-nums dark:text-zinc-400"
      >{compact ? `${done}/${total}` : t('checklist.progress', { done, total })}</span>
  {/if}
</span>
