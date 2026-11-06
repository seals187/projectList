<script lang="ts">
  // Page header area. With a background image it becomes a slim dark translucent
  // band over the full width of the page (like Planka); its content then uses the
  // dark variants (light text, dark controls). Without a background: a plain wrapper.
  import type { Snippet } from 'svelte';
  import { pageBg } from '$lib/pageBg.svelte';

  let { children }: { children: Snippet } = $props();
</script>

{#if pageBg.active}
  <div
    class="band glass-bar dark -mx-4 -mt-5 mb-5 border-b border-white/10 bg-zinc-900/50 px-4 py-2.5 text-zinc-100 sm:-mx-10 sm:-mt-7 sm:mb-6 sm:px-10 [&>*:last-child]:mb-0">
    {@render children()}
  </div>
{:else}
  {@render children()}
{/if}
