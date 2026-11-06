<script lang="ts">
  // One row of a settings card: title and description on the left, the
  // control (children) on the right; wraps below each other when narrow.
  import type { Snippet } from 'svelte';

  let { title, desc = '', labelFor, children }: { title: string; desc?: string; labelFor?: string; children?: Snippet } = $props();
</script>

<div class="flex flex-wrap items-center gap-x-6 gap-y-3 px-[18px] py-3.5">
  <div class="min-w-48 flex-1">
    {#if labelFor}
      <label for={labelFor} class="block text-[14.5px] font-semibold">{title}</label>
    {:else}
      <p class="text-[14.5px] font-semibold">{title}</p>
    {/if}
    {#if desc}<p class="mt-0.5 text-[13px] text-zinc-500 dark:text-zinc-400">{desc}</p>{/if}
  </div>
  {#if children}<div class="flex shrink-0 flex-wrap items-center gap-2">{@render children()}</div>{/if}
</div>
