<script lang="ts">
  // Page header as in the design: breadcrumbs left, actions right, below them
  // the title (serif) and an optional meta line.
  import type { Snippet } from 'svelte';

  let {
    crumbs = [],
    actions,
    title,
    meta
  }: {
    crumbs?: { label: string; href?: string }[];
    actions?: Snippet;
    title: Snippet;
    meta?: Snippet;
  } = $props();
</script>

<header class="mb-6">
  <div class="flex min-h-8 flex-wrap items-center justify-between gap-x-4 gap-y-2">
    <nav class="flex min-w-0 items-center gap-1.5 text-[13px] text-zinc-400" aria-label="Breadcrumb">
      {#each crumbs as c, i (i)}
        {#if i > 0}<span aria-hidden="true">›</span>{/if}
        {#if c.href}<a href={c.href} class="truncate hover:text-zinc-700 dark:hover:text-zinc-200">{c.label}</a>
        {:else}<span class="truncate">{c.label}</span>{/if}
      {/each}
    </nav>
    {#if actions}<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>{/if}
  </div>
  <div class="mt-1 flex flex-wrap items-baseline gap-x-5 gap-y-1">
    <h1 class="min-w-0 font-serif text-[28px] leading-tight font-bold tracking-tight">{@render title()}</h1>
    {#if meta}<div class="flex items-center gap-3">{@render meta()}</div>{/if}
  </div>
</header>
