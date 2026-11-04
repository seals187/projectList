<script lang="ts">
  import { saveState } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';

  const state = $derived(saveState.pending > 0 ? 'saving' : saveState.error ? 'error' : saveState.saved ? 'saved' : 'idle');
</script>

{#if state !== 'idle'}
  <span class="inline-flex items-center gap-1.5 text-xs text-zinc-500 dark:text-zinc-400" aria-live="polite">
    <span
      class="size-1.5 rounded-full {state === 'saving'
        ? 'animate-pulse bg-zinc-400'
        : state === 'error'
          ? 'bg-red-500'
          : 'bg-emerald-500'}"></span>
    {state === 'saving' ? t('save.saving') : state === 'error' ? t('save.error') : t('save.saved')}
  </span>
{/if}
