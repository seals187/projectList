<script lang="ts">
  // Shared header of the main pages (overview, my tasks, customers, templates):
  // serif title with a subtitle line below, actions on the right. Keeps the
  // layout identical when switching between these pages.
  import type { Snippet } from 'svelte';

  let { title, subtitle, actions }: { title: string; subtitle?: Snippet; actions?: Snippet } = $props();
</script>

<header class="mb-6 flex min-h-[3.75rem] flex-wrap items-start justify-between gap-4">
  <div class="min-w-0">
    <h1 class="truncate font-serif text-[28px] leading-tight font-bold tracking-tight">{title}</h1>
    {#if subtitle}<p class="mt-1 text-sm text-zinc-500">{@render subtitle()}</p>{/if}
  </div>
  {#if actions}<div class="flex flex-wrap items-center gap-3">{@render actions()}</div>{/if}
</header>
