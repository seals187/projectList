<script lang="ts">
  // Content of a main page in the size chosen in the page header (− 100 % +).
  // CSS zoom keeps the layout correct (unlike transform); menus and dialogs are
  // rendered outside (body / top layer) and keep their normal size.
  import type { Snippet } from 'svelte';
  import { prefs } from '$lib/state.svelte';
  import type { ZoomPage } from '$lib/pageZoom';

  let { page, children }: { page: ZoomPage; children: Snippet } = $props();
  const zoom = $derived(prefs.pageZoom[page] ?? 1);
</script>

<div style:zoom={zoom === 1 ? undefined : zoom} data-testid="zoom-area">
  {@render children()}
</div>
