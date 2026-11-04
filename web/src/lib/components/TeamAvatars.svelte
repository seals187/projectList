<script lang="ts">
  // Overlapping avatars (a project's team), names in the tooltip.
  import Avatar from './Avatar.svelte';

  let { team, size = 22, max = 4 }: { team: { id: number; name: string }[]; size?: number; max?: number } = $props();
  const shown = $derived(team.slice(0, max));
</script>

{#if team.length}
  <span class="flex shrink-0 items-center" title={team.map((p) => p.name).join(', ')}>
    {#each shown as p, i (p.id)}
      <span class="rounded-full ring-2 ring-white dark:ring-zinc-900 {i ? '-ml-1.5' : ''}"><Avatar name={p.name} id={p.id} {size} /></span>
    {/each}
    {#if team.length > max}<span class="ml-1 text-xs text-zinc-400 tabular-nums">+{team.length - max}</span>{/if}
  </span>
{/if}
