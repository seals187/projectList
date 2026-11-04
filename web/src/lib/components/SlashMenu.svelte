<script lang="ts">
  // Command menu when typing "/" in a block.
  import type { KINDS } from '$lib/blocks';
  import { t } from '$lib/i18n.svelte';
  import type { Kind } from '$lib/types';
  import BlockIcon from './BlockIcon.svelte';

  let { list, active, onselect, onhover }: { list: typeof KINDS; active: number; onselect: (k: Kind) => void; onhover: (i: number) => void } =
    $props();
</script>

<div
  class="absolute top-full left-6 z-30 mt-1 w-72 overflow-hidden rounded-xl border border-zinc-200 bg-white py-1.5 shadow-xl dark:border-zinc-700 dark:bg-zinc-900"
  role="listbox"
  tabindex="-1">
  {#each list as k, i (k.kind)}
    {#if i === 0 || list[i - 1].group !== k.group}
      <p
        class="px-3 pt-1.5 pb-1 text-[10px] font-semibold tracking-wider text-zinc-400 uppercase {i > 0
          ? 'mt-1 border-t border-zinc-100 pt-2.5 dark:border-zinc-800'
          : ''}">
        {k.group === 'headings' ? t('editor.headings') : t('editor.blocks')}
      </p>
    {/if}
    <button
      type="button"
      role="option"
      aria-selected={i === active}
      class="mx-1.5 flex w-[calc(100%-0.75rem)] items-center gap-3 rounded-lg px-2 py-1.5 text-left {i === active ? 'bg-accent-50 dark:bg-accent-950/60' : ''}"
      onmousedown={(e) => {
        e.preventDefault();
        onselect(k.kind);
      }}
      onmousemove={() => onhover(i)}>
      <span
        class="flex size-8 shrink-0 items-center justify-center rounded-md border border-zinc-200 bg-white text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
        <BlockIcon kind={k.kind} />
      </span>
      <span class="min-w-0 flex-1">
        <span class="block text-sm font-medium">{t(k.label)}</span>
        <span class="block text-xs text-zinc-400">{t(k.desc)}</span>
      </span>
      {#if i === active}<span class="text-xs text-zinc-400">↵</span>{/if}
    </button>
  {:else}
    <p class="px-3 py-2 text-sm text-zinc-400">{t('editor.noMatch')}</p>
  {/each}
</div>
