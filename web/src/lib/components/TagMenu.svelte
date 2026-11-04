<script lang="ts">
  // Tag list with search: toggle existing tags, create new ones with a color.
  // Used by the tag picker (project page) and the project context menu.
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { sidebar, toast } from '$lib/state.svelte';
  import { TAG_PRESETS, tagStyle } from '$lib/tags';
  import type { Tag } from '$lib/types';
  import Icon from './Icon.svelte';
  import ColorPicker from './ColorPicker.svelte';

  let {
    ids,
    onchange,
    onescape,
    autofocus = true
  }: { ids: number[]; onchange: (next: number[]) => void | Promise<void>; onescape?: () => void; autofocus?: boolean } = $props();

  const randomColor = () => TAG_PRESETS[Math.floor(Math.random() * TAG_PRESETS.length)];

  let query = $state('');
  let color = $state(randomColor());
  let busy = $state(false);

  const q = $derived(query.trim().toLowerCase());
  const matches = $derived(sidebar.tags.filter((tag) => !q || tag.name.toLowerCase().includes(q)));
  const exact = $derived(sidebar.tags.some((tag) => tag.name.toLowerCase() === q));

  function toggle(id: number) {
    query = '';
    onchange(ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id]);
  }

  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!q || exact || busy) return;
    busy = true;
    try {
      const tag = await api.post<Tag>('/tags', { name: query.trim(), color });
      sidebar.tags = [...sidebar.tags, tag].sort((a, b) => a.name.localeCompare(b.name));
      query = '';
      color = randomColor();
      await onchange([...ids, tag.id]);
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }
</script>

<form onsubmit={create}>
  <!-- svelte-ignore a11y_autofocus -->
  <input
    class="input py-1! text-sm!"
    placeholder={t('tags.searchOrCreate')}
    bind:value={query}
    {autofocus}
    onkeydown={(e) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onescape?.();
      }
    }}
    aria-label={t('tags.searchOrCreate')} />
</form>
<ul class="mt-1.5 max-h-56 overflow-y-auto">
  {#each matches as tag (tag.id)}
    {@const on = ids.includes(tag.id)}
    <li>
      <button
        type="button"
        class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
        onclick={() => toggle(tag.id)}
        aria-pressed={on}>
        <span class="size-3 shrink-0 rounded-full" style:background-color={tag.color}></span>
        <span class="min-w-0 flex-1 truncate">{tag.name}</span>
        {#if on}<Icon name="check" size={14} class="text-accent-600 dark:text-accent-400" />{/if}
      </button>
    </li>
  {:else}
    {#if !q}<li class="px-2 py-1.5 text-sm text-zinc-400">{t('tags.none')}</li>{/if}
  {/each}
</ul>
{#if q && !exact}
  <form onsubmit={create} class="mt-1.5 border-t border-zinc-100 pt-2 dark:border-zinc-800">
    <ColorPicker bind:value={color} />
    <button type="submit" class="mt-2 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" disabled={busy}>
      <Icon name="plus" size={14} />
      <span class="truncate">{t('tags.create')}</span>
      <span class="truncate rounded-full px-2 py-px text-[11px] font-medium" style={tagStyle(color)}>{query.trim()}</span>
    </button>
  </form>
{/if}
