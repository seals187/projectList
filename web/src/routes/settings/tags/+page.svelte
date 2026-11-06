<script lang="ts">
  // Settings → Tags (admin): tag catalog (filter, "+ Tag", per row "…" → edit
  // (name + color) or delete). Status names and colors stay on the overview board.
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { isAdmin, refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { tagStyle } from '$lib/tags';
  import type { Tag } from '$lib/types';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import Icon from '$lib/components/Icon.svelte';

  $effect(() => {
    if (!isAdmin()) goto('/settings');
  });

  // 16 colors of the tag editor (Tailwind 500/600 shades)
  const SWATCHES = [
    '#64748b', '#dc2626', '#ea580c', '#d97706', '#eab308', '#65a30d', '#16a34a', '#059669',
    '#0d9488', '#0891b2', '#0ea5e9', '#2563eb', '#4f46e5', '#7c3aed', '#a21caf', '#db2777'
  ];

  // ---- tags -------------------------------------------------------------------
  let query = $state('');
  const usage = $derived.by(() => {
    const m = new Map<number, number>();
    for (const c of sidebar.customers) for (const p of c.projects) for (const id of p.tag_ids) m.set(id, (m.get(id) ?? 0) + 1);
    return m;
  });
  const shown = $derived(sidebar.tags.filter((x) => x.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())));
  const sortTags = (list: Tag[]) => [...list].sort((a, b) => a.name.localeCompare(b.name));

  // "…" menu and the editor (tag = null: new tag)
  let menu = $state<{ tag: Tag; x: number; y: number; confirm: boolean } | null>(null);
  let editor = $state<{ tag: Tag | null; name: string; color: string; x: number; y: number } | null>(null);

  function openMenu(e: MouseEvent, tag: Tag) {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = { tag, x: r.right - 288, y: r.bottom + 4, confirm: false };
  }
  function openEditor(tag: Tag | null, anchor: HTMLElement) {
    const r = anchor.getBoundingClientRect();
    editor = { tag, name: tag?.name ?? '', color: tag?.color ?? SWATCHES[13], x: r.right - 288, y: r.bottom + 4 };
  }

  async function saveTag(e: SubmitEvent) {
    e.preventDefault();
    if (!editor || !editor.name.trim()) return;
    const { tag, name, color } = editor;
    try {
      if (tag) {
        const updated = await api.patch<Tag>(`/tags/${tag.id}`, { name, color });
        sidebar.tags = sortTags(sidebar.tags.map((x) => (x.id === tag.id ? updated : x)));
      } else {
        const created = await api.post<Tag>('/tags', { name, color });
        sidebar.tags = sortTags([...sidebar.tags, created]);
      }
      editor = null;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function removeTag(tag: Tag) {
    menu = null;
    try {
      await api.del(`/tags/${tag.id}`);
      sidebar.tags = sidebar.tags.filter((x) => x.id !== tag.id);
      refreshSidebar(0);
    } catch (err) {
      toast(errorText(err));
    }
  }

</script>

<SettingsShell title={t('set.tags.title')} subtitle={t('set.tags.subtitle')}>
  <div class="max-w-[680px]">
    <!-- tags -->
    <section aria-labelledby="tags-h">
      <div class="mb-3 flex items-center gap-2">
        <h3 id="tags-h" class="text-[16px] font-semibold">{t('tags.title')}</h3>
        <span class="text-[14px] text-zinc-400 tabular-nums">{sidebar.tags.length}</span>
        <button type="button" class="ph-primary ml-auto" onclick={(e) => openEditor(null, e.currentTarget)}><Icon name="plus" size={15} />{t('tags.add')}</button>
      </div>
      <div class="overflow-hidden rounded-[14px] border border-zinc-200/90 bg-white dark:border-zinc-800 dark:bg-zinc-900">
        <label class="flex items-center gap-2 border-b border-zinc-100 px-4 py-2.5 text-zinc-400 dark:border-zinc-800">
          <Icon name="search" size={14} />
          <input class="min-w-0 flex-1 bg-transparent text-[14px] text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100" placeholder={t('set.tags.filter')} bind:value={query} aria-label={t('set.tags.filter')} />
        </label>
        <ul class="divide-y divide-zinc-100 dark:divide-zinc-800">
          {#each shown as tag (tag.id)}
            <li class="group flex items-center gap-4 px-4 py-2.5 hover:bg-zinc-50 dark:hover:bg-zinc-800/40" data-testid="tag-row">
              <span class="size-[26px] shrink-0 rounded-[7px]" style:background-color={tag.color} aria-hidden="true"></span>
              <span class="min-w-28 flex-1 truncate text-[15px]">{tag.name}</span>
              <span class="hidden max-w-40 truncate rounded-[5px] px-2 py-px text-[12.5px] font-semibold sm:inline" style={tagStyle(tag.color)}>{tag.name}</span>
              <span class="w-16 shrink-0 text-right text-[13.5px] text-zinc-400 tabular-nums">{t('set.tags.nProj', { n: usage.get(tag.id) ?? 0 })}</span>
              <button type="button" class="rounded-md p-1 text-zinc-400 hover:bg-zinc-200/60 hover:text-zinc-700 dark:hover:bg-zinc-700 dark:hover:text-zinc-200" onclick={(e) => openMenu(e, tag)} aria-label="{t('menu.actions')}: {tag.name}">
                <Icon name="dots" size={16} />
              </button>
            </li>
          {:else}
            <li class="px-4 py-4 text-sm text-zinc-400">{sidebar.tags.length ? t('set.noMatch') : t('tags.none')}</li>
          {/each}
        </ul>
      </div>
      <p class="mt-3 text-[13px] text-zinc-500">{t('settings.statusOnBoard')}</p>
    </section>

  </div>
</SettingsShell>

{#if menu}
  {@const m = menu}
  <ContextMenu x={m.x} y={m.y} label="{t('menu.actions')}: {m.tag.name}" onclose={() => (menu = null)}>
    <button
      type="button"
      role="menuitem"
      class="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-left hover:bg-zinc-200/60 dark:hover:bg-zinc-800"
      onclick={() => {
        menu = null;
        editor = { tag: m.tag, name: m.tag.name, color: m.tag.color, x: m.x, y: m.y };
      }}>
      <Icon name="pencil" size={14} />{t('set.tags.edit')}
    </button>
    {#if m.confirm}
      <button type="button" role="menuitem" class="flex w-full items-center gap-2 rounded-lg bg-red-50 px-2.5 py-1.5 text-left font-semibold text-red-700 hover:bg-red-100 dark:bg-red-950/40 dark:text-red-300" onclick={() => removeTag(m.tag)}>
        <Icon name="trash" size={14} />{t('common.confirmDelete')}
      </button>
    {:else}
      <button type="button" role="menuitem" class="flex w-full items-center gap-2 rounded-lg px-2.5 py-1.5 text-left text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-950/40" onclick={() => (menu = { ...m, confirm: true })}>
        <Icon name="trash" size={14} />{t('set.tags.delete')}
      </button>
    {/if}
  </ContextMenu>
{/if}

{#if editor}
  {@const ed = editor}
  <ContextMenu x={ed.x} y={ed.y} label={ed.tag ? t('set.tags.edit') : t('tags.newName')} onclose={() => (editor = null)}>
    <form class="p-2" onsubmit={saveTag}>
      <p class="mb-2.5 text-[14.5px] font-semibold">{ed.tag ? t('set.tags.editTitle') : t('set.tags.newTitle')}</p>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input h-10!" maxlength="40" required autofocus bind:value={editor.name} placeholder={t('tags.newName')} aria-label={t('common.name')} />
      <div class="mt-3 grid grid-cols-8 gap-1.5" role="radiogroup" aria-label={t('tags.color')}>
        {#each SWATCHES as c (c)}
          {@const on = ed.color.toLowerCase() === c}
          <button
            type="button"
            role="radio"
            aria-checked={on}
            aria-label={c}
            class="aspect-square rounded-[7px] {on ? 'ring-2 ring-offset-2 ring-offset-zinc-50 dark:ring-offset-zinc-900' : ''}"
            style:background-color={c}
            style:--tw-ring-color={c}
            onclick={() => editor && (editor.color = c)}></button>
        {/each}
      </div>
      <div class="mt-3.5 flex justify-end gap-2">
        <button type="button" class="btn-outline" onclick={() => (editor = null)}>{t('common.cancel')}</button>
        <button type="submit" class="btn-primary" disabled={!ed.name.trim()}>{t('common.save')}</button>
      </div>
    </form>
  </ContextMenu>
{/if}
