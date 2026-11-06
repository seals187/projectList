<script lang="ts">
  // Template library. Header (PageHeader): search, sorting, expand/collapse
  // (descriptions); state per user in localStorage and in the URL.
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { tick } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { canWrite, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { addPageView } from '$lib/overviewFilter.svelte';
  import { pageState } from '$lib/pageState.svelte';
  import { displayLabel, displayOf, setDisplay } from '$lib/display.svelte';
  import type { Template, TemplateSummary } from '$lib/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import EntityIcon from '$lib/components/EntityIcon.svelte';

  type Sort = 'name' | 'items' | 'newest';
  const SORTS: Sort[] = ['name', 'items', 'newest'];

  const ps = pageState('templates', {
    view: { def: 'library' as string, url: 'str' },
    q: { def: '', url: 'str' },
    sort: { def: 'name' as string, url: 'str' },
    collapsed: { def: false } // descriptions hidden
  });
  const sort = $derived<Sort>(SORTS.includes(ps.sort as Sort) ? (ps.sort as Sort) : 'name');
  // "Display" options (per user)
  const DISPLAY = { description: true, items: true };
  const disp = $derived(displayOf('templates.library', DISPLAY));

  let templates = $state<TemplateSummary[] | null>(null);
  let newName = $state('');
  let newInput = $state<HTMLInputElement>();

  $effect(() => {
    api.get<TemplateSummary[]>('/templates').then(
      (r) => (templates = r),
      (e) => toast(errorText(e))
    );
  });

  const shown = $derived.by(() => {
    if (!templates) return null;
    const q = ps.q.trim().toLowerCase();
    const list = templates.filter((x) => !q || x.name.toLowerCase().includes(q) || x.description.toLowerCase().includes(q));
    const byName = (a: TemplateSummary, b: TemplateSummary) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
    if (sort === 'items') return list.sort((a, b) => b.item_count - a.item_count || byName(a, b));
    if (sort === 'newest') return list.sort((a, b) => b.id - a.id);
    return list.sort(byName);
  });

  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      const tpl = await api.post<Template>('/templates', { name: newName });
      goto(`/templates/${tpl.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }
  // "+ Template": to the name field below the list
  async function focusNew() {
    await tick();
    newInput?.scrollIntoView({ block: 'center', behavior: 'smooth' });
    newInput?.focus();
  }
</script>

<svelte:head><title>{t('nav.templates')} · ProjectList</title></svelte:head>

<PageHeader
  icon="fileText"
  title={t('nav.templates')}
  subtitle={t('templates.subtitle')}
  views={[{ key: 'library', label: t('templates.view.library'), icon: 'fileText', count: templates?.length }]}
  activeView="library"
  onViewChange={(v) => (ps.view = v)}
  primaryAction={canWrite() ? { label: t('ph.addTemplate'), onClick: focusNew } : undefined}
  searchPlaceholder={t('ph.search.templates')}
  searchValue={ps.q}
  onSearch={(q) => (ps.q = q)}
  onSaveView={(name) => addPageView(name, 'templates', page.url.search.slice(1))}
  sortOptions={SORTS.map((s) => ({ value: s, label: t(`sort.${s}`) }))}
  activeSort={sort}
  onSortChange={(s) => (ps.sort = s)}
  collapsed={ps.collapsed}
  onToggleCollapse={() => (ps.collapsed = !ps.collapsed)}
  collapseDisabled={!shown?.some((x) => x.description)}
  displayOptions={Object.keys(DISPLAY).map((k) => ({ key: k, label: displayLabel(k), value: disp[k] }))}
  onDisplayChange={(k, v) => setDisplay('templates.library', k, v)} />

<ul class="surface rounded-xl in-[.has-bg]:px-3 in-[.has-bg]:py-1">
  {#if shown === null}
    <li class="py-2 text-sm text-zinc-400">{t('common.loading')}</li>
  {:else}
    {#each shown as tpl (tpl.id)}
      <li>
        <a href="/templates/{tpl.id}" class="row-link">
          <EntityIcon type="template" size="md" />
          <div class="min-w-0 flex-1">
            <p class="truncate font-medium">{tpl.name}</p>
            {#if tpl.description && disp.description && !ps.collapsed}<p class="truncate text-xs text-zinc-400">{tpl.description}</p>{/if}
          </div>
          {#if disp.items}<span class="text-xs text-zinc-400">{t('templates.items', { n: tpl.item_count })}</span>{/if}
        </a>
      </li>
    {:else}
      <li class="py-2 text-sm text-zinc-400">{t('templates.empty')}</li>
    {/each}
  {/if}
</ul>
{#if canWrite()}
  <form onsubmit={create} class="surface mt-1 flex items-center gap-2 rounded-xl py-1 in-[.has-bg]:mt-3 in-[.has-bg]:px-3 in-[.has-bg]:py-2">
    <Icon name="plus" class="text-zinc-400" />
    <input bind:this={newInput} class="flex-1 bg-transparent py-1 text-sm outline-none placeholder:text-zinc-400" placeholder={t('templates.new')} bind:value={newName} />
    {#if newName.trim()}<button class="btn-primary py-1!" type="submit">{t('common.create')}</button>{/if}
  </form>
{/if}
