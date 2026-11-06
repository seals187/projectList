<script lang="ts">
  // Favorites in the sidebar: two-line entries (type icon, name, "type · context"),
  // ungrouped favorites first, then the user's groups (bookmark icon, name, count;
  // collapsible, state saved per user). Drag & drop sorts favorites within and
  // between groups and sorts the groups. Right click (touch: long press):
  // open, open in new tab, move to group (incl. new group), remove; on a group
  // rename / delete; on the section header: new group.
  import type { Snippet } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { sidebar } from '$lib/state.svelte';
  import { entityLabelKey, entitySubtitle } from '$lib/entityTypes';
  import { moveNode, type DropTarget, type FavRef } from '$lib/favLayout';
  import { favKey, favoritesLayout, groupKey, resolveFavorites, type FavDisplay } from '$lib/favoritesModel';
  import {
    createFavoriteGroup,
    deleteFavoriteGroup,
    moveFavoriteToGroup,
    saveFavoritesOrder,
    toggleFavorite,
    updateFavoriteGroup
  } from '$lib/navMemory.svelte';
  import { ensureTemplates, templatesIndex } from '$lib/templatesIndex.svelte';
  import ContextMenu from './ContextMenu.svelte';
  import EntityIcon from './EntityIcon.svelte';
  import Icon from './Icon.svelte';

  let { closed, head, activePath }: { closed: boolean; head: Snippet; activePath: string } = $props();

  type TKey = Parameters<typeof t>[0];

  // templates are not part of /api/sidebar: load them when a template is a favorite
  $effect(() => ensureTemplates(sidebar.favorites.filter((f) => f.kind === 'template').map((f) => f.id)));
  const templates = $derived(templatesIndex.list);

  const shown = $derived(
    resolveFavorites(sidebar.favorites, sidebar.customers, templates, {
      nProjects: (n) => t('overview.nProjects', { n }),
      nItems: (n) => t('templates.items', { n })
    })
  );
  // only favorites that can be shown take part in the layout
  const layout = $derived(
    favoritesLayout(
      sidebar.favorites.filter((f) => shown.has(favKey(f))),
      sidebar.favoriteGroups
    )
  );
  const visible = $derived(shown.size > 0 || sidebar.favoriteGroups.length > 0);
  const typeLabel = (d: FavDisplay) => t(entityLabelKey(d.kind) as TKey);
  const subtitle = (d: FavDisplay) => entitySubtitle(typeLabel(d), d.context);
  const isActive = (d: FavDisplay) => activePath === d.href;

  // ---- drag & drop ----------------------------------------------------------------
  let drag = $state<FavRef | null>(null);
  let drop = $state<DropTarget | null>(null);
  function start(e: DragEvent, ref: FavRef) {
    if ((e.target as HTMLElement).closest?.('input')) return e.preventDefault();
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
  function overGroup(e: DragEvent, id: string) {
    if (!drag) return;
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const y = (e.clientY - r.top) / r.height;
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
    if (drag && drop) {
      const next = moveNode(layout, drag, drop);
      if (next !== layout) saveFavoritesOrder(next);
    }
    drag = drop = null;
  }
  const isDrop = (pos: 'before' | 'after', ref: FavRef) =>
    !!drop && drop.pos === pos && (ref.t === 'i' ? drop.ref.t === 'i' && drop.ref.k === ref.k : drop.ref.t === 'f' && drop.ref.id === ref.id);

  // ---- groups ------------------------------------------------------------------------
  let renaming = $state<number | null>(null);
  let renameText = $state('');
  function finishRename(id: number) {
    if (renaming !== id) return;
    renaming = null;
    const name = renameText.trim();
    if (name) updateFavoriteGroup(id, { name });
  }
  // one frame later: inputs inside the context menu are moved to <body> first, which drops the focus
  const focusSelect = (el: HTMLInputElement) =>
    void requestAnimationFrame(() => {
      el.focus();
      el.select();
    });

  // ---- context menu (right click, long press on touch) --------------------------------
  type MenuOn = { t: 'section' } | { t: 'fav'; d: FavDisplay; group: number | null } | { t: 'group'; id: number; name: string };
  let menu = $state<{ x: number; y: number; on: MenuOn; sub: boolean; naming: boolean; newName: string } | null>(null);
  function openMenu(x: number, y: number, on: MenuOn) {
    menu = { x, y, on, sub: false, naming: false, newName: '' };
  }
  function onContext(e: MouseEvent, on: MenuOn) {
    if ((e.target as HTMLElement).closest?.('input')) return;
    e.preventDefault();
    e.stopPropagation();
    openMenu(e.clientX, e.clientY, on);
  }
  let pressTimer: ReturnType<typeof setTimeout> | undefined;
  let pressed = false; // long press opened the menu: swallow the following click
  function pressStart(e: PointerEvent, on: MenuOn) {
    if (e.pointerType !== 'touch') return;
    clearTimeout(pressTimer);
    const { clientX: x, clientY: y } = e;
    pressTimer = setTimeout(() => {
      pressed = true;
      openMenu(x, y, on);
    }, 550);
  }
  const pressEnd = () => clearTimeout(pressTimer);
  function swallowClick(e: MouseEvent) {
    if (pressed) {
      pressed = false;
      e.preventDefault();
      e.stopPropagation();
    }
  }
  const close = () => (menu = null);
  async function newGroup(name: string, move?: FavDisplay) {
    const v = name.trim().slice(0, 60);
    close();
    if (v) await createFavoriteGroup(v, move ? { kind: move.kind, id: move.id } : undefined);
  }
  const itemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 disabled:opacity-50 dark:hover:bg-zinc-800';
</script>

{#snippet fav(key: string, group: number | null)}
  {@const d = shown.get(key)}
  {#if d}
    {@const ref = { t: 'i' as const, k: key }}
    {@const on = isActive(d)}
    <li
      class="relative {drag?.t === 'i' && drag.k === key ? 'opacity-40' : ''}"
      draggable="true"
      ondragstart={(e) => start(e, ref)}
      ondragover={(e) => overItem(e, key)}
      ondrop={finish}
      ondragend={() => (drag = drop = null)}
      data-testid="fav-item">
      {#if isDrop('before', ref)}<div class="pointer-events-none absolute inset-x-1 -top-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
      <a
        href={d.href}
        draggable="false"
        class="flex min-h-11 items-center gap-2.5 rounded-[7px] px-2.5 py-[5px] transition-colors {on
          ? 'bg-[#ece9fd] dark:bg-white/12'
          : 'hover:bg-zinc-200/50 dark:hover:bg-white/6'}"
        aria-current={on ? 'page' : undefined}
        title="{d.name}&#10;{subtitle(d)}"
        oncontextmenu={(e) => onContext(e, { t: 'fav', d, group })}
        onpointerdown={(e) => pressStart(e, { t: 'fav', d, group })}
        onpointerup={pressEnd}
        onpointercancel={pressEnd}
        onpointermove={pressEnd}
        onclickcapture={swallowClick}>
        <EntityIcon type={d.kind} size="md" />
        <span class="min-w-0 flex-1 leading-tight">
          <span class="block truncate text-[13px] {on ? 'font-semibold' : ''}">{d.name}</span>
          <span class="mt-0.5 block truncate text-[11px] text-zinc-400" data-testid="fav-subtitle">{subtitle(d)}</span>
        </span>
      </a>
      {#if isDrop('after', ref)}<div class="pointer-events-none absolute inset-x-1 -bottom-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
    </li>
  {/if}
{/snippet}

{#if visible}
  <div oncontextmenu={(e) => onContext(e, { t: 'section' })} role="presentation">{@render head()}</div>
{/if}
{#if visible && !closed}
  <ul class="space-y-px" aria-label={t('nav.favorites')} oncontextmenu={(e) => onContext(e, { t: 'section' })}>
    {#each layout as n (n.t === 'i' ? n.k : n.id)}
      {#if n.t === 'i'}
        {@render fav(n.k, null)}
      {:else}
        {@const gid = Number(n.id.slice(1))}
        {@const ref = { t: 'f' as const, id: n.id }}
        <li class="relative {drag?.t === 'f' && drag.id === n.id ? 'opacity-40' : ''}" data-testid="fav-group">
          {#if isDrop('before', ref)}<div class="pointer-events-none absolute inset-x-1 -top-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
          {#if renaming === gid}
            <!-- rename: Enter or leaving the field saves, Esc cancels -->
            <input
              class="input py-1! text-sm!"
              bind:value={renameText}
              use:focusSelect
              maxlength="60"
              aria-label={t('fav.renameGroup')}
              onblur={() => finishRename(gid)}
              onkeydown={(e) => {
                if (e.key === 'Enter') finishRename(gid);
                if (e.key === 'Escape') renaming = null;
              }} />
          {:else}
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-[7px] px-2.5 py-1.5 text-left hover:bg-zinc-200/50 dark:hover:bg-white/6 {drop?.pos === 'into' && drop.folder === n.id
                ? 'bg-accent-500/12 ring-1 ring-accent-500/50'
                : ''}"
              draggable="true"
              ondragstart={(e) => start(e, ref)}
              ondragover={(e) => overGroup(e, n.id)}
              ondrop={finish}
              ondragend={() => (drag = drop = null)}
              oncontextmenu={(e) => onContext(e, { t: 'group', id: gid, name: n.name })}
              onpointerdown={(e) => pressStart(e, { t: 'group', id: gid, name: n.name })}
              onpointerup={pressEnd}
              onpointercancel={pressEnd}
              onpointermove={pressEnd}
              onclickcapture={swallowClick}
              onclick={() => updateFavoriteGroup(gid, { collapsed: n.open })}
              ondblclick={() => ((renaming = gid), (renameText = n.name))}
              aria-expanded={n.open}>
              <Icon name="chevronDown" size={13} class="shrink-0 text-zinc-400 transition-transform {n.open ? '' : '-rotate-90'}" />
              <EntityIcon type="favGroup" size="sm" class="-mx-1" />
              <span class="min-w-0 flex-1 truncate text-[13.5px] font-semibold">{n.name}</span>
              <span class="text-xs text-zinc-400 tabular-nums">{n.items.length}</span>
            </button>
          {/if}
          {#if n.open}
            <ul class="space-y-px pl-4">
              {#each n.items as k (k)}
                {@render fav(k, gid)}
              {:else}
                <li class="py-1 pl-3 text-xs text-zinc-400">{t('fav.emptyFolder')}</li>
              {/each}
            </ul>
          {/if}
          {#if isDrop('after', ref)}<div class="pointer-events-none absolute inset-x-1 -bottom-px z-10 h-0.5 rounded bg-accent-500"></div>{/if}
        </li>
      {/if}
    {/each}
    <!-- drop zone: end of the list -->
    <li class="relative h-2" ondragover={overEnd} ondrop={finish} aria-hidden="true">
      {#if drop?.pos === 'end'}<div class="pointer-events-none absolute inset-x-1 top-0 h-0.5 rounded bg-accent-500"></div>{/if}
    </li>
  </ul>
{/if}

{#snippet nameInput(move?: FavDisplay)}
  <form
    class="px-1 py-1"
    onsubmit={(e) => {
      e.preventDefault();
      if (menu) newGroup(menu.newName, move);
    }}>
    {#if menu}
      <input class="input py-1! text-sm!" maxlength="60" placeholder={t('fav.groupName')} aria-label={t('fav.groupName')} bind:value={menu.newName} use:focusSelect />
    {/if}
  </form>
{/snippet}

{#if menu}
  {@const m = menu}
  {@const on = m.on}
  <ContextMenu x={m.x} y={m.y} label={t('nav.favorites')} onclose={close}>
    {#if on.t === 'section'}
      {#if m.naming}
        {@render nameInput()}
      {:else}
        <button type="button" class={itemCls} role="menuitem" onclick={() => m && (m.naming = true)}>
          <EntityIcon type="favGroup" size="sm" class="-mx-1" />{t('fav.newGroup')}
        </button>
      {/if}
    {:else if on.t === 'group'}
      <button type="button" class={itemCls} role="menuitem" onclick={() => ((renaming = on.id), (renameText = on.name), close())}>
        <Icon name="pencil" size={15} class="text-zinc-400" />{t('fav.renameGroup')}
      </button>
      <button type="button" class="{itemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => (deleteFavoriteGroup(on.id), close())}>
        <Icon name="trash" size={15} />{t('fav.deleteGroup')}
      </button>
    {:else}
      {@const d = on.d}
      <button type="button" class={itemCls} role="menuitem" onclick={() => (goto(d.href), close())}>
        <Icon name="external" size={15} class="rotate-0 text-zinc-400" />{t('fav.open')}
      </button>
      <button type="button" class={itemCls} role="menuitem" onclick={() => (window.open(d.href, '_blank', 'noopener'), close())}>
        <Icon name="copy" size={15} class="text-zinc-400" />{t('fav.openNewTab')}
      </button>
      <button type="button" class={itemCls} role="menuitem" aria-expanded={m.sub} onclick={() => m && (m.sub = !m.sub)}>
        <EntityIcon type="favGroup" size="sm" class="-mx-1" /><span class="flex-1">{t('fav.moveToGroup')}</span>
        <Icon name="chevron" size={14} class="text-zinc-400 transition-transform {m.sub ? 'rotate-90' : ''}" />
      </button>
      {#if m.sub}
        <div class="ml-4 border-l border-zinc-200 pl-1 dark:border-zinc-700" data-testid="fav-move-menu">
          {#each [...sidebar.favoriteGroups].sort((a, b) => a.position - b.position) as g (g.id)}
            <button type="button" class={itemCls} role="menuitem" disabled={on.group === g.id} onclick={() => (moveFavoriteToGroup(d.kind, d.id, g.id), close())}>
              {#if on.group === g.id}<Icon name="check" size={14} class="text-zinc-400" />{/if}<span class="truncate">{g.name}</span>
            </button>
          {/each}
          {#if on.group !== null}
            <button type="button" class={itemCls} role="menuitem" onclick={() => (moveFavoriteToGroup(d.kind, d.id, null), close())}>{t('fav.noGroup')}</button>
          {/if}
          {#if m.naming}
            {@render nameInput(d)}
          {:else}
            <button type="button" class="{itemCls} text-accent-700 dark:text-accent-300" role="menuitem" onclick={() => m && (m.naming = true)}>{t('fav.newGroup')}</button>
          {/if}
        </div>
      {/if}
      <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
      <button type="button" class="{itemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => (toggleFavorite(d.kind, d.id), close())}>
        <Icon name="trash" size={15} />{t('nav.removeFavorite')}
      </button>
    {/if}
  </ContextMenu>
{/if}
