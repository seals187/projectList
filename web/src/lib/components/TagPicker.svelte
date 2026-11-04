<script lang="ts">
  // Tags of a project: chips plus a dropdown to toggle existing tags
  // and to create new ones with a free color.
  import { t } from '$lib/i18n.svelte';
  import { setProjectTags } from '$lib/projectActions';
  import { tagsByIds, tagStyle } from '$lib/tags';
  import Icon from './Icon.svelte';
  import TagMenu from './TagMenu.svelte';

  let {
    projectId,
    ids = $bindable(),
    disabled = false
  }: { projectId: number; ids: number[]; disabled?: boolean } = $props();

  let open = $state(false);
  const assigned = $derived(tagsByIds(ids));

  async function save(next: number[]) {
    const prev = ids;
    ids = next; // optimistic
    const saved = await setProjectTags(projectId, next);
    ids = saved ?? prev;
  }
</script>

<svelte:window onmousedown={(e) => open && !(e.target as HTMLElement).closest('[data-tagpicker]') && (open = false)} />

<div class="relative flex flex-wrap items-center gap-1.5" data-tagpicker>
  {#each assigned as tag (tag.id)}
    <span class="inline-flex items-center gap-1 rounded-full py-0.5 pr-1.5 pl-2.5 text-xs font-medium {disabled ? 'pr-2.5' : ''}" style={tagStyle(tag.color)}>
      {tag.name}
      {#if !disabled}
        <button type="button" class="rounded-full opacity-70 hover:opacity-100" onclick={() => save(ids.filter((x) => x !== tag.id))} aria-label="{t('tags.remove')}: {tag.name}">
          <Icon name="x" size={12} />
        </button>
      {/if}
    </span>
  {/each}
  {#if !disabled}
    <button
      type="button"
      class="inline-flex items-center gap-1 rounded-full border border-dashed border-zinc-300 px-2.5 py-0.5 text-xs text-zinc-500 hover:border-zinc-400 hover:text-zinc-700 dark:border-zinc-700 dark:hover:text-zinc-200"
      onclick={() => (open = !open)}
      aria-expanded={open}>
      <Icon name="plus" size={12} />{t('tags.add')}
    </button>
  {/if}

  {#if open}
    <div class="absolute top-full left-0 z-30 mt-1.5 w-64 rounded-xl border border-zinc-200 bg-white p-2 shadow-lg dark:border-zinc-700 dark:bg-zinc-900">
      <TagMenu {ids} onchange={save} onescape={() => (open = false)} />
    </div>
  {/if}
</div>
