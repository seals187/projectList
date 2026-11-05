<script lang="ts">
  // Shared header of the main pages (overview, my tasks, customers, templates):
  // serif title with a subtitle line below, actions on the right, optional
  // toolbar row (filters, tabs). Keeps the layout identical when switching
  // between these pages. With a background image it sits on the slim header band:
  // smaller title with the subtitle next to it, and fixed row heights (the toolbar
  // row is reserved even when empty) so the band is equally high on every page.
  import type { Snippet } from 'svelte';
  import { pageBg } from '$lib/pageBg.svelte';
  import HeaderBand from './HeaderBand.svelte';

  let { title, subtitle, actions, toolbar }: { title: string; subtitle?: Snippet; actions?: Snippet; toolbar?: Snippet } = $props();
</script>

<HeaderBand>
  {#if pageBg.active}
    <header class="mb-2 flex min-h-9 flex-wrap items-center justify-between gap-x-4 gap-y-2">
      <div class="flex min-w-0 items-baseline gap-3">
        <h1 class="truncate font-serif text-xl leading-tight font-bold tracking-tight">{title}</h1>
        {#if subtitle}<p class="hidden truncate text-sm text-zinc-400 sm:block">{@render subtitle()}</p>{/if}
      </div>
      {#if actions}<div class="flex flex-wrap items-center gap-3">{@render actions()}</div>{/if}
    </header>
    <div class="flex min-h-10 flex-col justify-center [&>*]:mb-0!">
      {#if toolbar}{@render toolbar()}{/if}
    </div>
  {:else}
    <header class="mb-6 flex min-h-[3.75rem] flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="truncate font-serif text-[28px] leading-tight font-bold tracking-tight">{title}</h1>
        {#if subtitle}<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{@render subtitle()}</p>{/if}
      </div>
      {#if actions}<div class="flex flex-wrap items-center gap-3">{@render actions()}</div>{/if}
    </header>
    {#if toolbar}{@render toolbar()}{/if}
  {/if}
</HeaderBand>
