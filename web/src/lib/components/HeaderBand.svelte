<script lang="ts">
  // Page header area. With a background image and the glass effect on, it
  // becomes a slim dark translucent band over the full width of the page; its
  // content then uses the dark variants (light text, dark controls). With the
  // glass effect off: an opaque band in the normal theme colors. Without a
  // background: a plain wrapper.
  import type { Snippet } from 'svelte';
  import { pageBg } from '$lib/pageBg.svelte';
  import { prefs } from '$lib/state.svelte';

  let { children }: { children: Snippet } = $props();
</script>

{#if pageBg.active && prefs.glass}
  <div
    class="band glass-bar dark -mx-4 -mt-5 mb-5 border-b border-white/10 bg-zinc-900/50 px-4 py-2.5 text-zinc-100 sm:-mx-10 sm:-mt-7 sm:mb-6 sm:px-10 [&>*:last-child]:mb-0">
    {@render children()}
  </div>
{:else if pageBg.active}
  <div class="-mx-4 -mt-5 mb-5 border-b border-zinc-200 bg-white px-4 py-4 sm:-mx-10 sm:-mt-7 sm:mb-6 sm:px-10 dark:border-zinc-800 dark:bg-zinc-900 [&>*:last-child]:mb-0">
    {@render children()}
  </div>
{:else}
  {@render children()}
{/if}
