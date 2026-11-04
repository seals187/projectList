<script lang="ts">
  import { toasts } from '$lib/state.svelte';
</script>

<div class="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4" aria-live="polite">
  {#each toasts.list as t (t.id)}
    <div
      class="pointer-events-auto max-w-md rounded-md px-4 py-2 text-sm shadow-lg {t.kind === 'error'
        ? 'bg-red-600 text-white'
        : 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'}">
      {t.text}
    </div>
  {/each}
</div>
