<script lang="ts">
  // Sidebar section with user-defined order and folders (favorites, saved views).
  // The layout is a per-user pref (e.g. "fav_layout"), synced across devices.
  // Drag & drop moves entries before/after others or into a folder; folders
  // move on the top level. Right click: on the section "New folder"; on a folder
  // rename / move / delete; on an entry move up/down, into a folder, plus the
  // entry's own actions (`itemMenu`). The entry itself is rendered by `item`.
  import type { Snippet } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { sidebar, toast } from '$lib/state.svelte';
  import {
    addFolder,
    deleteFolder,
    folderOf,
    moveNode,
    parseLayout,
    renameFolder,
    resolveLayout,
    setFolderOpen,
    stepNode,
    type DropTarget,
    type FavLayout,
    type FavRef
  } from '$lib/favLayout';
  import ContextMenu from './ContextMenu.svelte';
  import Icon from './Icon.svelte';

  let {
    keys,
    prefKey,
    label,
    closed,
    head,
    navCls,
    item,
    itemMenu,
    testid
  }: {
    keys: string[]; // entries in their default order (new ones are appended)
    prefKey: 'fav_layout' | 'views_layout';
    label: string;
    closed: boolean;
    head: Snippet;
    navCls: (on: boolean) => string;
    item: Snippet<[string, boolean]>; // (key, inside a folder)
    itemMenu?: Snippet<[string, () => void]>; // (key, close menu)
    testid: string;
  } = $props();

  const layout = $derived(resolveLayout(parseLayout(sidebar.prefs[prefKey]), keys));
  const folders = $derived(layout.filter((n) => n.t === 'f'));
  const visible = $derived(keys.length > 0 || folders.length > 0);

  function save(l: FavLayout) {
    sidebar.prefs[prefKey] = l;
    api.put(`/prefs/${prefKey}`, l).catch((e) => toast(errorText(e)));
  }

  // ---- drag & drop ----------------------------------------------------------------
  let drag = $state<FavRef | null>(null);
  let drop = $state<DropTarget | null>(null);

  function start(e: DragEvent, ref: FavRef) {
    if ((e.target as HTMLElement).closest?.('input')) return e.preventDefault(); // renaming
    drag = ref;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', ref.t === 'i' ? ref.k : ref.id);
    }
  }
  function overItem(e: DragEvent, key: string) {
    if (!drag) return;
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    drop = { pos: e.clientY < r.top + r.height / 2 ? 'before' : 'after', ref: { t: 'i', k: key } };
  }
  function overFolder(e: DragEvent, id: string) {
    if (!drag) return;
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const y = (e.clientY - r.top) / r.height;
    // an entry goes into the folder (upper edge: before it); a folder goes before/after
    if (drag.t === 'i') drop = y < 0.3 ? { pos: 'before', ref: { t: 'f', id } } : { pos: 'into', folder: id };
    else drop = { pos: y < 0.5 ? 'before' : 'after', ref: { t: 'f', id } };
  }
  function overEnd(e: DragEvent) {
    if (!drag) return;
    e.preventDefault();
    drop = { pos: 'end' };
  }
  function finish(e: DragEvent) {
    e.preventDefault();
    if (drag && drop) save(moveNode(layout, drag, drop));
    drag = drop = null;
  }
  const isDrop = (pos: 'before' | 'after', ref: FavRef) =>
    !!drop && drop.pos === pos && (ref.t === 'i' ? drop.ref.t === 'i' && drop.ref.k === ref.k : drop.ref.t === 'f' && drop.ref.id === ref.id);

  // ---- folders ----------------------------------------------------------------------
  let renaming = $state<string | null>(null);
  let renameText = $state('');
  function newFolder() {
    const id = Math.random().toString(36).slice(2, 10);
    const name = t('fav.newFolder');
    save(addFolder(layout, id, name));
    renaming = id;
    renameText = name;
  }
  function finishRename(id: string) {
    if (renaming !== id) return;
    renaming = null;
    save(renameFolder(layout, id, renameText));
  }
  const selectAll = (el: HTMLInputElement) => {
    el.focus();
    el.select();
  };

  // ---- context menus ----------------------------------------------------------------
  let menu = $state<{ x: number; y: number; on: { t: 'section' } | FavRef } | null>(null);
  function openMenu(e: MouseEvent, on: { t: 'section' } | FavRef) {
    if ((e.target as HTMLElement).closest?.('input')) return; // native menu while renaming
    e.preventDefault();
    e.stopPropagation();
    menu = { x: e.clientX, y: e.clientY, on };
  }
  const closeMenu = () => (menu = null);
  const run = (fn: () => void) => () => {
    fn();
    menu = null;
  };
  const itemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800';
</script>

{#snippet entry(key: string, nested: boolean)}
  {@const ref = { t: 'i' as const, k: key }}
  <li
    class="group relative {drag?.t === 'i' && drag.k === key ? 'opacity-40' : ''}"
    draggable="true"
    ondragstart={(ev) => start(ev, ref)}
    ondragover={(ev) => overItem(ev, key)}
    ondrop={finish}
    ondragend={() => (drag = drop = null)}
    oncontextmenu={(ev) => openMenu(ev, ref)}
    data-testid="{testid}-item">
    {#if isDrop('before', ref)}<div class="pointer-events-none absolute inset-x-1 -top-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
    {@render item(key, nested)}
    {#if isDrop('after', ref)}<div class="pointer-events-none absolute inset-x-1 -bottom-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
  </li>
{/snippet}

{#if visible}
  <div oncontextmenu={(e) => openMenu(e, { t: 'section' })} role="presentation">{@render head()}</div>
{/if}
{#if visible && !closed}
  <ul class="space-y-px" oncontextmenu={(e) => openMenu(e, { t: 'section' })} aria-label={label}>
    {#each layout as n (n.t === 'i' ? n.k : `f:${n.id}`)}
      {#if n.t === 'i'}
        {@render entry(n.k, false)}
      {:else}
        {@const ref = { t: 'f' as const, id: n.id }}
        <li class="relative {drag?.t === 'f' && drag.id === n.id ? 'opacity-40' : ''}" data-testid="{testid}-folder">
          {#if isDrop('before', ref)}<div class="pointer-events-none absolute inset-x-1 -top-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
          {#if renaming === n.id}
            <!-- rename: Enter or leaving the field saves, Esc cancels -->
            <input
              class="input py-1! text-sm!"
              bind:value={renameText}
              use:selectAll
              maxlength="60"
              aria-label={t('fav.rename')}
              onblur={() => finishRename(n.id)}
              onkeydown={(e) => {
                if (e.key === 'Enter') finishRename(n.id);
                if (e.key === 'Escape') renaming = null;
              }} />
          {:else}
            <button
              type="button"
              class="{navCls(false)} w-full font-normal! {drop?.pos === 'into' && drop.folder === n.id ? 'bg-accent-500/12 ring-1 ring-accent-500/50' : ''}"
              draggable="true"
              ondragstart={(e) => start(e, ref)}
              ondragover={(e) => overFolder(e, n.id)}
              ondrop={finish}
              ondragend={() => (drag = drop = null)}
              oncontextmenu={(e) => openMenu(e, ref)}
              onclick={() => save(setFolderOpen(layout, n.id, !n.open))}
              ondblclick={() => ((renaming = n.id), (renameText = n.name))}
              aria-expanded={n.open}>
              <Icon name="chevronDown" size={13} class="-ml-0.5 shrink-0 text-zinc-400 transition-transform {n.open ? '' : '-rotate-90'}" />
              <Icon name="folder" size={15} class="-ml-1 shrink-0 text-zinc-500 dark:text-zinc-400" />
              <span class="min-w-0 flex-1 truncate text-left">{n.name}</span>
              <span class="text-xs text-zinc-400 tabular-nums">{n.items.length}</span>
            </button>
          {/if}
          {#if n.open}
            <ul class="space-y-px">
              {#each n.items as k (k)}
                {@render entry(k, true)}
              {:else}
                <li class="py-1 pl-7 text-xs text-zinc-400">{t('fav.emptyFolder')}</li>
              {/each}
            </ul>
          {/if}
          {#if isDrop('after', ref)}<div class="pointer-events-none absolute inset-x-1 -bottom-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
        </li>
      {/if}
    {/each}
    <!-- drop zone: end of the list (top level) -->
    <li class="relative h-2" ondragover={overEnd} ondrop={finish} aria-hidden="true">
      {#if drop?.pos === 'end'}<div class="pointer-events-none absolute inset-x-1 top-0 h-0.5 rounded bg-accent-500"></div>{/if}
    </li>
  </ul>
{/if}

{#if menu}
  {@const m = menu}
  {@const on = m.on}
  <ContextMenu x={m.x} y={m.y} {label} onclose={closeMenu}>
    {#if on.t === 'section'}
      <button type="button" class={itemCls} role="menuitem" onclick={run(newFolder)}><Icon name="folder" size={15} class="text-zinc-400" />{t('fav.newFolder')}</button>
    {:else if on.t === 'f'}
      {@const f = folders.find((x) => x.t === 'f' && x.id === on.id)}
      <button type="button" class={itemCls} role="menuitem" onclick={run(() => ((renaming = on.id), (renameText = f?.t === 'f' ? f.name : '')))}>
        <Icon name="pencil" size={15} class="text-zinc-400" />{t('fav.rename')}
      </button>
      <button type="button" class={itemCls} role="menuitem" onclick={run(() => save(stepNode(layout, on, -1)))}><Icon name="chevronDown" size={15} class="rotate-180 text-zinc-400" />{t('fav.moveUp')}</button>
      <button type="button" class={itemCls} role="menuitem" onclick={run(() => save(stepNode(layout, on, 1)))}><Icon name="chevronDown" size={15} class="text-zinc-400" />{t('fav.moveDown')}</button>
      <button type="button" class={itemCls} role="menuitem" onclick={run(newFolder)}><Icon name="folder" size={15} class="text-zinc-400" />{t('fav.newFolder')}</button>
      <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
      <button type="button" class="{itemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={run(() => save(deleteFolder(layout, on.id)))}>
        <Icon name="trash" size={15} />{t('fav.deleteFolder')}
      </button>
    {:else}
      {@const cur = folderOf(layout, on.k)}
      <button type="button" class={itemCls} role="menuitem" onclick={run(() => save(stepNode(layout, on, -1)))}><Icon name="chevronDown" size={15} class="rotate-180 text-zinc-400" />{t('fav.moveUp')}</button>
      <button type="button" class={itemCls} role="menuitem" onclick={run(() => save(stepNode(layout, on, 1)))}><Icon name="chevronDown" size={15} class="text-zinc-400" />{t('fav.moveDown')}</button>
      {#if folders.length}
        <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
        <p class="px-2 pt-0.5 pb-1 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('fav.moveTo')}</p>
        {#each folders as f (f.t === 'f' ? f.id : '')}
          {#if f.t === 'f'}
            <button type="button" class={itemCls} role="menuitem" disabled={cur === f.id} onclick={run(() => save(moveNode(layout, on, { pos: 'into', folder: f.id })))}>
              <Icon name={cur === f.id ? 'check' : 'folder'} size={15} class="text-zinc-400" />{f.name}
            </button>
          {/if}
        {/each}
        {#if cur}
          <button type="button" class={itemCls} role="menuitem" onclick={run(() => save(moveNode(layout, on, { pos: 'after', ref: { t: 'f', id: cur } })))}>
            <Icon name="x" size={15} class="text-zinc-400" />{t('fav.noFolder')}
          </button>
        {/if}
      {/if}
      <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
      <button type="button" class={itemCls} role="menuitem" onclick={run(newFolder)}><Icon name="folder" size={15} class="text-zinc-400" />{t('fav.newFolder')}</button>
      {@render itemMenu?.(on.k, closeMenu)}
    {/if}
  </ContextMenu>
{/if}
