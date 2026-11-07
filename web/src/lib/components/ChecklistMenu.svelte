<script lang="ts">
  // Context menu of a checklist (My tasks): open, open project, rename,
  // favorite, save as template, delete (with confirmation).
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { t, type Key } from '$lib/i18n.svelte';
  import { canWrite, refreshSidebar, sidebar, toast } from '$lib/state.svelte';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';
  import type { SidebarChecklist, Template } from '$lib/types';
  import ContextMenu from './ContextMenu.svelte';
  import Icon from './Icon.svelte';
  import EntityIcon from './EntityIcon.svelte';

  let {
    checklist,
    project,
    x,
    y,
    onclose
  }: {
    checklist: SidebarChecklist;
    project: { id: number; name: string };
    x: number;
    y: number;
    onclose: () => void;
  } = $props();

  // current values from the store (the page may pass a copy)
  const storeChecklist = () => sidebar.customers.flatMap((c) => c.projects).flatMap((p) => p.checklists).find((c) => c.id === checklist.id);
  const cl = $derived(storeChecklist() ?? checklist);
  const editable = canWrite();
  const fav = $derived(isFavorite('checklist', cl.id));

  type Section = 'rename' | 'template' | 'delete';
  let section = $state<Section | null>(null);
  const toggle = (s: Section) => (section = section === s ? null : s);
  let name = $state('');
  let busy = $state(false);

  function open(href: string) {
    onclose();
    goto(href);
  }

  async function rename(e: SubmitEvent) {
    e.preventDefault();
    const title = name.trim();
    if (!title || title === cl.title) return;
    try {
      await api.patch(`/checklists/${cl.id}`, { title });
      const s = storeChecklist();
      if (s) s.title = title;
      refreshSidebar(0);
      section = null;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function saveAsTemplate(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    try {
      const tpl = await api.post<Template>(`/checklists/${cl.id}/save-as-template`, { name: name.trim() });
      toast(t('checklist.templateSaved', { name: tpl.name }), 'info');
      onclose();
    } catch (err) {
      toast(errorText(err));
    }
  }

  const count = (n: number, one: Key, many: Key) => (n === 1 ? t(one) : t(many, { n }));
  async function remove() {
    busy = true;
    try {
      await api.del(`/checklists/${cl.id}`);
      for (const c of sidebar.customers) for (const p of c.projects) p.checklists = p.checklists.filter((x) => x.id !== cl.id);
      refreshSidebar(0);
      toast(t('checklist.deleted', { name: cl.title }), 'info');
      onclose();
    } catch (err) {
      busy = false;
      toast(errorText(err));
    }
  }

  const itemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800';
  const subCls = 'mx-1 mb-1.5 rounded-lg bg-zinc-50 p-1.5 dark:bg-zinc-800/50';
</script>

<ContextMenu {x} {y} {onclose} label={cl.title}>
  <div class="mb-1 flex items-start gap-2.5 border-b border-zinc-100 px-2 pt-1 pb-2.5 dark:border-zinc-800">
    <EntityIcon type="checklist" size="sm" class="mt-0.5" />
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">{cl.title}</p>
      <p class="truncate text-xs text-zinc-500">{project.name}</p>
    </div>
    <span class="shrink-0 text-xs text-zinc-400 tabular-nums">{cl.done}/{cl.total}</span>
  </div>

  <button type="button" class={itemCls} role="menuitem" onclick={() => open(`/checklists/${cl.id}`)}>
    <Icon name="external" size={15} class="text-zinc-400" />{t('menu.openChecklist')}
  </button>
  <button type="button" class={itemCls} role="menuitem" onclick={() => open(`/projects/${project.id}`)}>
    <Icon name="folder" size={15} class="text-zinc-400" />{t('menu.openProject')}
  </button>
  <button type="button" class={itemCls} role="menuitem" onclick={() => (toggleFavorite('checklist', cl.id), onclose())}>
    <Icon name="star" size={15} class={fav ? 'fill-amber-400 text-amber-500' : 'text-zinc-400'} />{t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
  </button>

  {#if editable}
    <button type="button" class={itemCls} role="menuitem" aria-expanded={section === 'rename'} onclick={() => ((name = cl.title), toggle('rename'))}>
      <Icon name="pencil" size={15} class="text-zinc-400" />{t('menu.rename')}
    </button>
    {#if section === 'rename'}
      <form onsubmit={rename} class="{subCls} flex gap-1.5">
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input py-1! text-[13px]!" bind:value={name} maxlength="200" autofocus aria-label={t('menu.rename')} />
        <button type="submit" class="btn-primary px-2! py-1!" disabled={!name.trim()} aria-label={t('common.save')}><Icon name="check" size={14} /></button>
      </form>
    {/if}

    <button type="button" class={itemCls} role="menuitem" aria-expanded={section === 'template'} onclick={() => ((name = cl.title), toggle('template'))}>
      <Icon name="fileText" size={15} class="text-zinc-400" />{t('checklist.saveAsTemplate')}
    </button>
    {#if section === 'template'}
      <form onsubmit={saveAsTemplate} class="{subCls} flex gap-1.5">
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input py-1! text-[13px]!" bind:value={name} maxlength="200" autofocus aria-label={t('checklist.saveAsTemplate')} />
        <button type="submit" class="btn-primary px-2! py-1!" disabled={!name.trim()} aria-label={t('common.save')}><Icon name="check" size={14} /></button>
      </form>
    {/if}

    <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
    {#if section !== 'delete'}
      <button type="button" class="{itemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => (section = 'delete')}>
        <Icon name="trash" size={15} class="shrink-0" />{t('checklist.delete')}
      </button>
    {:else}
      <div class="mx-1 mb-1 rounded-lg border border-red-200 bg-red-50 p-2.5 text-[13px] dark:border-red-500/30 dark:bg-red-500/10" role="alertdialog" aria-label={t('checklist.delete')} data-testid="checklist-delete-confirm">
        <p class="font-semibold text-red-700 dark:text-red-300">{t('checklist.deleteAsk', { name: cl.title })}</p>
        {#if cl.total}
          <p class="mt-1.5 flex gap-1.5 text-red-700 dark:text-red-300">
            <Icon name="alert" size={14} class="mt-0.5 shrink-0" />
            <span>{t('checklist.deleteWarn', { items: count(cl.total, 'project.del.it1', 'project.del.itn') })}</span>
          </p>
        {/if}
        <p class="mt-1.5 text-zinc-500 dark:text-zinc-400">{t('common.irreversible')}</p>
        <div class="mt-2.5 flex justify-end gap-1.5">
          <button type="button" class="btn-outline py-1!" onclick={() => (section = null)}>{t('common.cancel')}</button>
          <!-- svelte-ignore a11y_autofocus -->
          <button type="button" class="btn bg-red-600 py-1! text-white hover:bg-red-700" onclick={remove} disabled={busy} autofocus>{t('common.delete')}</button>
        </div>
      </div>
    {/if}
  {/if}
</ContextMenu>
