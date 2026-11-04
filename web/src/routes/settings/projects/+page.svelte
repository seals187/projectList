<script lang="ts">
  // Project settings (admin): Easy Redmine base URL and the tag catalog. Status names and colors are
  // edited directly on the overview board (click on a column title).
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { isAdmin, refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { TAG_PRESETS, tagStyle } from '$lib/tags';
  import type { Tag } from '$lib/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SettingsTabs from '$lib/components/SettingsTabs.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import ColorPicker from '$lib/components/ColorPicker.svelte';
  import Icon from '$lib/components/Icon.svelte';

  $effect(() => {
    if (!isAdmin()) goto('/settings');
  });

  // Easy Redmine base URL (ticket links on the project page)
  let redmineUrl = $state('');
  let savedRedmine = $state('');
  $effect(() => {
    api.get<{ redmine_url: string }>('/settings/app').then(
      (r) => (redmineUrl = savedRedmine = r.redmine_url),
      () => {}
    );
  });
  async function saveRedmine(e: SubmitEvent) {
    e.preventDefault();
    try {
      const r = await api.put<{ redmine_url: string }>('/settings/app', { redmine_url: redmineUrl });
      redmineUrl = savedRedmine = r.redmine_url;
      toast(t('settings.redmineSaved'), 'info');
    } catch (err) {
      toast(errorText(err));
    }
  }

  let newName = $state('');
  let newColor = $state(TAG_PRESETS[29]);

  // color popover of an existing tag
  let colorFor = $state<{ tag: Tag; x: number; y: number } | null>(null);

  /** Number of projects per tag (visible projects). */
  const usage = $derived.by(() => {
    const m = new Map<number, number>();
    for (const c of sidebar.customers) for (const p of c.projects) for (const id of p.tag_ids) m.set(id, (m.get(id) ?? 0) + 1);
    return m;
  });

  const sortTags = (list: Tag[]) => [...list].sort((a, b) => a.name.localeCompare(b.name));

  async function createTag(e: SubmitEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      const tag = await api.post<Tag>('/tags', { name: newName, color: newColor });
      sidebar.tags = sortTags([...sidebar.tags, tag]);
      newName = '';
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function patchTag(id: number, body: Partial<Tag>) {
    try {
      const tag = await api.patch<Tag>(`/tags/${id}`, body);
      sidebar.tags = sortTags(sidebar.tags.map((x) => (x.id === id ? tag : x)));
    } catch (err) {
      toast(errorText(err));
      throw err;
    }
  }

  async function removeTag(id: number) {
    try {
      await api.del(`/tags/${id}`);
      sidebar.tags = sidebar.tags.filter((x) => x.id !== id);
      refreshSidebar(0);
    } catch (err) {
      toast(errorText(err));
    }
  }
</script>

<svelte:head><title>{t('nav.settings')} · ProjectList</title></svelte:head>

<PageHeader>
  {#snippet title()}{t('nav.settings')}{/snippet}
</PageHeader>
<SettingsTabs />

<div class="max-w-xl space-y-10">
  <section>
    <h2 class="section-label mb-1">Easy Redmine</h2>
    <p class="mb-3 text-sm text-zinc-500">{t('settings.redmineHint')}</p>
    <form onsubmit={saveRedmine} class="flex items-center gap-2">
      <input class="input" type="url" placeholder="https://redmine.example.com" bind:value={redmineUrl} aria-label={t('settings.redmineUrl')} />
      <button class="btn-primary shrink-0" type="submit" disabled={redmineUrl.trim().replace(/\/+$/, '') === savedRedmine}>{t('common.save')}</button>
    </form>
  </section>

  <section>
    <h2 class="section-label mb-1">{t('tags.title')}</h2>
    <p class="mb-1 text-sm text-zinc-500">{t('tags.hint')}</p>
    <p class="mb-4 text-sm text-zinc-500">{t('settings.statusOnBoard')}</p>

    <ul class="divide-y divide-zinc-100 rounded-xl border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
      {#each sidebar.tags as tag (tag.id)}
        <li class="flex items-center gap-3 px-3 py-2">
          <button
            type="button"
            class="size-6 shrink-0 rounded-full border border-black/10"
            style:background-color={tag.color}
            title={t('tags.color')}
            aria-label="{t('tags.color')}: {tag.name}"
            onclick={(e) => {
              const r = e.currentTarget.getBoundingClientRect();
              colorFor = { tag, x: r.left, y: r.bottom + 4 };
            }}></button>
          <div class="min-w-0 flex-1 text-sm">
            <Editable value={tag.name} required onsave={(name) => patchTag(tag.id, { name })} />
          </div>
          <span class="hidden rounded-full px-2 py-px text-[11px] font-medium sm:inline" style={tagStyle(tag.color)}>{tag.name}</span>
          <span class="w-20 shrink-0 text-right text-xs text-zinc-400 tabular-nums">{t('tags.nProjects', { n: usage.get(tag.id) ?? 0 })}</span>
          <ConfirmButton class="btn-ghost px-1.5! text-zinc-400 hover:text-red-600" onconfirm={() => removeTag(tag.id)} confirmLabel={t('common.confirmDelete')}>
            <Icon name="trash" size={14} />
          </ConfirmButton>
        </li>
      {:else}
        <li class="px-3 py-3 text-sm text-zinc-400">{t('tags.none')}</li>
      {/each}
    </ul>

    <form onsubmit={createTag} class="mt-3 flex items-center gap-2">
      <span class="size-8 shrink-0 rounded-full border border-black/10" style:background-color={newColor}></span>
      <input class="input" maxlength="40" placeholder={t('tags.newName')} bind:value={newName} aria-label={t('tags.newName')} />
      <button class="btn-primary shrink-0" type="submit" disabled={!newName.trim()}>{t('common.create')}</button>
    </form>
    <div class="mt-3 max-w-72 pl-10">
      <ColorPicker bind:value={newColor} />
    </div>
  </section>
</div>

{#if colorFor}
  {@const c = colorFor}
  <ContextMenu x={c.x} y={c.y} label="{t('tags.color')}: {c.tag.name}" onclose={() => (colorFor = null)}>
    <div class="p-1.5">
      <ColorPicker
        value={c.tag.color}
        onpick={(color) => {
          colorFor = null;
          patchTag(c.tag.id, { color }).catch(() => {});
        }} />
    </div>
  </ContextMenu>
{/if}
