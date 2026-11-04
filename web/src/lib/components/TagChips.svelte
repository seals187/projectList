<script lang="ts">
  // Colored tag chips of a project (overview cards, list, board).
  import { tagsByIds, tagStyle } from '$lib/tags';

  let { ids, class: cls = '' }: { ids: readonly number[] | undefined; class?: string } = $props();
  const tags = $derived(tagsByIds(ids));
</script>

{#if tags.length}
  <span class="flex flex-wrap gap-1 {cls}">
    {#each tags as tag (tag.id)}
      <span class="inline-block max-w-full truncate rounded-full px-2 py-px text-[11px] leading-4 font-medium" style={tagStyle(tag.color)}>{tag.name}</span>
    {/each}
  </span>
{/if}
