<script lang="ts">
  // Tags as colored dots with text. Shows as many as fit into the available
  // width, the rest as "+N" (tooltip: all tags); wrap = show all and wrap.
  // Measured with a ResizeObserver, so it adapts to the window size.
  import { fitTags } from '$lib/listMetrics';
  import type { Tag } from '$lib/types';

  let { tags, wrap = false, class: cls = '' }: { tags: Tag[]; wrap?: boolean; class?: string } = $props();

  const GAP = 10;
  let box = $state<HTMLElement>();
  let measure = $state<HTMLElement>();
  let fit = $state(Infinity);

  function recalc() {
    if (!box || !measure || wrap) return;
    const items = [...measure.children] as HTMLElement[];
    const widths = items.slice(0, -1).map((e) => e.offsetWidth);
    const plus = items.at(-1)?.offsetWidth ?? 24;
    fit = fitTags(widths, box.clientWidth, plus, GAP);
  }
  $effect(() => {
    void tags.length;
    void wrap;
    if (!box || typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(() => recalc());
    ro.observe(box);
    recalc();
    return () => ro.disconnect();
  });

  const shown = $derived(wrap ? tags : tags.slice(0, fit));
  const rest = $derived(tags.length - shown.length);
  const all = $derived(tags.map((x) => x.name).join(', '));
</script>

{#if tags.length}
  <span bind:this={box} class="relative flex min-w-0 items-center gap-x-2.5 gap-y-0.5 text-xs text-zinc-600 dark:text-zinc-400 {wrap ? 'flex-wrap' : 'overflow-hidden'} {cls}" title={all}>
    {#each shown as tag (tag.id)}
      <span class="flex shrink-0 items-center gap-1 whitespace-nowrap"><span class="size-2 shrink-0 rounded-full" style:background-color={tag.color}></span>{tag.name}</span>
    {/each}
    {#if rest > 0}<span class="shrink-0 font-medium text-zinc-400" data-testid="tags-more">+{rest}</span>{/if}
    <!-- invisible copy for measuring the natural widths -->
    <span bind:this={measure} class="pointer-events-none invisible absolute top-0 left-0 flex gap-x-2.5 whitespace-nowrap" aria-hidden="true">
      {#each tags as tag (tag.id)}<span class="flex items-center gap-1"><span class="size-2 rounded-full"></span>{tag.name}</span>{/each}
      <span class="font-medium">+{tags.length}</span>
    </span>
  </span>
{/if}
