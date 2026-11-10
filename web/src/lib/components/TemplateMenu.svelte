<script lang="ts">
  // Context menu of a template (template library): open, use, favorite,
  // category, rename, delete (with confirmation). Changes are reported to the page via onchange.
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { canWrite, toast } from '$lib/state.svelte';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';
  import { templatesIndex } from '$lib/templatesIndex.svelte';
  import type { Template, TemplateCategory, TemplateSummary } from '$lib/types';
  import { categoryHex } from '$lib/templateLibrary';
  import ContextMenu from './ContextMenu.svelte';
  import Icon from './Icon.svelte';
  import { templateSizeText } from '$lib/templateText';
  import EntityIcon from './EntityIcon.svelte';

  let {
    template,
    x,
    y,
    onclose,
    onrenamed,
    ondeleted,
    onuse,
    categories = [],
    oncategory
  }: {
    template: TemplateSummary;
    x: number;
    y: number;
    onclose: () => void;
    onrenamed: (name: string) => void;
    ondeleted: () => void;
    onuse?: () => void; // "Use" (apply dialog)
    categories?: TemplateCategory[];
    oncategory?: (id: number | null) => void; // category changed
  } = $props();

  const editable = canWrite();
  const fav = $derived(isFavorite('template', template.id));
  type Section = 'rename' | 'delete' | 'category';
  let section = $state<Section | null>(null);
  let name = $state('');
  let busy = $state(false);

  function open() {
    onclose();
    goto(`/templates/${template.id}`);
  }

  async function rename(e: SubmitEvent) {
    e.preventDefault();
    const n = name.trim();
    if (!n || n === template.name) return;
    try {
      const saved = await api.patch<Template>(`/templates/${template.id}`, { name: n });
      const idx = templatesIndex.list?.find((x) => x.id === template.id); // breadcrumb, favorites
      if (idx) idx.name = saved.name;
      onrenamed(saved.name);
      section = null;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function setCategory(id: number | null) {
    try {
      await api.patch(`/templates/${template.id}`, { category_id: id });
      oncategory?.(id);
      section = null;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function remove() {
    busy = true;
    try {
      await api.del(`/templates/${template.id}`);
      if (templatesIndex.list) templatesIndex.list = templatesIndex.list.filter((x) => x.id !== template.id);
      toast(t('template.deleted', { name: template.name }), 'info');
      ondeleted();
      onclose();
    } catch (err) {
      busy = false;
      toast(errorText(err));
    }
  }

  const itemCls = 'flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800';
</script>

<ContextMenu {x} {y} {onclose} label={template.name}>
  <div class="mb-1 flex items-start gap-2.5 border-b border-zinc-100 px-2 pt-1 pb-2.5 dark:border-zinc-800">
    <EntityIcon type="template" size="sm" class="mt-0.5" />
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">{template.name}</p>
      <p class="truncate text-xs text-zinc-500">{templateSizeText(template)}</p>
    </div>
  </div>

  <button type="button" class={itemCls} role="menuitem" onclick={open}>
    <Icon name="external" size={15} class="text-zinc-400" />{t('menu.openTemplate')}
  </button>
  {#if editable && onuse}
    <button type="button" class={itemCls} role="menuitem" onclick={() => (onclose(), onuse())}>
      <Icon name="plus" size={15} class="text-zinc-400" />{t('apply.title')}
    </button>
  {/if}
  <button type="button" class={itemCls} role="menuitem" onclick={() => (toggleFavorite('template', template.id), onclose())}>
    <Icon name="star" size={15} class={fav ? 'fill-amber-400 text-amber-500' : 'text-zinc-400'} />{t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
  </button>

  {#if editable && categories.length}
    {@const cur = categories.find((c) => c.id === template.category_id)}
    <button type="button" class={itemCls} role="menuitem" aria-expanded={section === 'category'} onclick={() => (section = section === 'category' ? null : 'category')}>
      <Icon name="folder" size={15} class="text-zinc-400" />{t('cat.title')}
      <span class="ml-auto max-w-[45%] truncate text-xs text-zinc-400">{cur?.name ?? ''}</span>
    </button>
    {#if section === 'category'}
      <div class="mx-1 mb-1.5 rounded-lg bg-zinc-50 p-1.5 dark:bg-zinc-800/50">
        {#each categories as c (c.id)}
          <button type="button" class="{itemCls} text-[13px]" onclick={() => setCategory(c.id)} aria-pressed={template.category_id === c.id}>
            <span class="size-2.5 shrink-0 rounded-sm" style:background-color={categoryHex(c)}></span>
            <span class="flex-1 truncate">{c.name}</span>
            {#if template.category_id === c.id}<Icon name="check" size={14} class="text-accent-600 dark:text-accent-400" />{/if}
          </button>
        {/each}
        <button type="button" class="{itemCls} text-[13px] text-zinc-500" onclick={() => setCategory(null)} aria-pressed={!template.category_id}>
          <span class="size-2.5 shrink-0 rounded-sm border border-zinc-300 dark:border-zinc-600"></span>
          <span class="flex-1">{t('cat.none')}</span>
          {#if !template.category_id}<Icon name="check" size={14} class="text-accent-600 dark:text-accent-400" />{/if}
        </button>
      </div>
    {/if}
  {/if}

  {#if editable}
    <button type="button" class={itemCls} role="menuitem" aria-expanded={section === 'rename'} onclick={() => ((name = template.name), (section = section === 'rename' ? null : 'rename'))}>
      <Icon name="pencil" size={15} class="text-zinc-400" />{t('menu.rename')}
    </button>
    {#if section === 'rename'}
      <form onsubmit={rename} class="mx-1 mb-1.5 flex gap-1.5 rounded-lg bg-zinc-50 p-1.5 dark:bg-zinc-800/50">
        <!-- svelte-ignore a11y_autofocus -->
        <input class="input py-1! text-[13px]!" bind:value={name} maxlength="200" autofocus aria-label={t('menu.rename')} />
        <button type="submit" class="btn-primary px-2! py-1!" disabled={!name.trim()} aria-label={t('common.save')}><Icon name="check" size={14} /></button>
      </form>
    {/if}

    <div class="my-1 border-t border-zinc-100 dark:border-zinc-800"></div>
    {#if section !== 'delete'}
      <button type="button" class="{itemCls} text-red-600 dark:text-red-400" role="menuitem" onclick={() => (section = 'delete')}>
        <Icon name="trash" size={15} class="shrink-0" />{t('template.delete')}
      </button>
    {:else}
      <div class="mx-1 mb-1 rounded-lg border border-red-200 bg-red-50 p-2.5 text-[13px] dark:border-red-500/30 dark:bg-red-500/10" role="alertdialog" aria-label={t('template.delete')} data-testid="template-delete-confirm">
        <p class="font-semibold text-red-700 dark:text-red-300">{t('template.deleteAsk', { name: template.name })}</p>
        <p class="mt-1.5 text-zinc-500 dark:text-zinc-400">{t('template.deleteNote')}</p>
        <div class="mt-2.5 flex justify-end gap-1.5">
          <button type="button" class="btn-outline py-1!" onclick={() => (section = null)}>{t('common.cancel')}</button>
          <!-- svelte-ignore a11y_autofocus -->
          <button type="button" class="btn bg-red-600 py-1! text-white hover:bg-red-700" onclick={remove} disabled={busy} autofocus>{t('common.delete')}</button>
        </div>
      </div>
    {/if}
  {/if}
</ContextMenu>
