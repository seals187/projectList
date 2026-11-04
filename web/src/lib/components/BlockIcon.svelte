<script lang="ts">
  // Small icons for block types (slash menu, toolbar).
  import type { Kind } from '$lib/types';

  let { kind, size = 16 }: { kind: Kind | 'ol' | 'ul'; size?: number } = $props();

  const paths: Record<string, string> = {
    item: 'M4 6.5l1.5 1.5 3-3M4 16.5l1.5 1.5 3-3M12 7h8M12 17h8',
    text: 'M5 6h14M5 12h14M5 18h9',
    bullet: 'M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01',
    numbered: 'M10 6h10M10 12h10M10 18h10M4 5h1.5v4M4 9h3M4 14.5c0-.8.6-1.5 1.5-1.5S7 13.6 7 14.3c0 .9-3 2-3 3.7h3',
    quote: 'M7 7H4v5h3l-1 4M17 7h-3v5h3l-1 4',
    code: 'M5 5h14v14H5zM9.5 10l-2 2 2 2M14.5 10l2 2-2 2',
    divider: 'M4 12h16'
  };
</script>

{#if kind === 'h1' || kind === 'h2' || kind === 'h3' || kind === 'h4'}
  <span class="text-[11px] leading-none font-bold" style:width="{size}px" style:text-align="center">{kind.toUpperCase()}</span>
{:else}
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path d={paths[kind === 'ol' ? 'numbered' : kind === 'ul' ? 'bullet' : kind]} />
  </svg>
{/if}
