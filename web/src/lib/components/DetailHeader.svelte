<script lang="ts">
  // Page header of detail pages (checklist, template): title (serif) with the
  // type icon of the page, an optional meta line and actions on the right.
  // The breadcrumb is shown by the back bar above every page (BackBar).
  import type { Snippet } from 'svelte';
  import type { EntityType } from '$lib/entityTypes';
  import EntityIcon from './EntityIcon.svelte';

  let {
    type,
    actions,
    title,
    meta
  }: {
    type?: EntityType; // type icon (lg) before the title
    actions?: Snippet;
    title: Snippet;
    meta?: Snippet;
  } = $props();
</script>

<header class="mb-6 flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
  <div class="flex min-w-0 flex-wrap items-center gap-x-5 gap-y-1">
    <div class="flex min-w-0 items-center gap-3">
      {#if type}<EntityIcon {type} size="lg" />{/if}
      <h1 class="min-w-0 font-serif text-[28px] leading-tight font-bold tracking-tight">{@render title()}</h1>
    </div>
    {#if meta}<div class="flex items-center gap-3">{@render meta()}</div>{/if}
  </div>
  {#if actions}<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>{/if}
</header>
