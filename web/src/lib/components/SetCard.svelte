<script lang="ts">
  // Section of a settings page: heading (optional) and a bordered card whose
  // children (SetRow) are separated by lines.
  import type { Snippet } from 'svelte';

  let { title = '', id, class: cls = '', children }: { title?: string; id?: string; class?: string; children: Snippet } = $props();
</script>

<section {id} class="mb-8 scroll-mt-6 {cls}">
  {#if title}<h3 class="mb-2.5 text-[16px] font-semibold">{title}</h3>{/if}
  <div class="divide-y divide-zinc-100 rounded-[14px] border border-zinc-200/90 bg-white dark:divide-zinc-800 dark:border-zinc-800 dark:bg-zinc-900">
    {@render children()}
  </div>
</section>
