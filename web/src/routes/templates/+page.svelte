<script lang="ts">
  // Templates page with three views:
  //  - Library: templates grouped by category (chips filter a category), items,
  //    usage, last use, "Use" (apply dialog) and menu; admins manage categories.
  //  - Automations: rules trigger → template and the latest runs (admins edit).
  //  - Variables: system and custom variables for {{name}} (admins edit).
  // Header (PageHeader): search, chips, sorting, expand/collapse (library
  // groups), display options; state per user in localStorage and in the URL.
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { api, errorText } from '$lib/api';
  import { canWrite, isAdmin, toast } from '$lib/state.svelte';
  import { t, type Key } from '$lib/i18n.svelte';
  import { addPageView } from '$lib/overviewFilter.svelte';
  import { pageState } from '$lib/pageState.svelte';
  import { allCollapsed } from '$lib/pageStateCore';
  import { displayLabel, displayOf, setDisplay } from '$lib/display.svelte';
  import { categoryHex, groupTemplates, lastUsed } from '$lib/templateLibrary';
  import { templateSizeText } from '$lib/templateText';
  import { triggerDetail, triggerTitle } from '$lib/automationText';
  import type { AutomationRule, AutomationRun, Template, TemplateCategory, TemplateKind, TemplateSummary, TemplateVariable } from '$lib/types';
  import PageHeader, { type QuickFilter } from '$lib/components/PageHeader.svelte';
  import ZoomArea from '$lib/components/ZoomArea.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import TemplateMenu from '$lib/components/TemplateMenu.svelte';
  import ApplyTemplateDialog from '$lib/components/ApplyTemplateDialog.svelte';
  import CategoryDialog from '$lib/components/CategoryDialog.svelte';
  import NewTemplateDialog from '$lib/components/NewTemplateDialog.svelte';
  import RuleDialog from '$lib/components/RuleDialog.svelte';
  import VariableDialog from '$lib/components/VariableDialog.svelte';
  import AutomationsView from '$lib/components/AutomationsView.svelte';
  import VariablesView from '$lib/components/VariablesView.svelte';

  type View = 'library' | 'automations' | 'variables';
  const VIEWS: View[] = ['library', 'automations', 'variables'];
  const SORTS: Record<View, string[]> = {
    library: ['name', 'items', 'used', 'lastUsed', 'newest'],
    automations: ['name', 'runs'],
    variables: ['name', 'used']
  };

  const ps = pageState('templates', {
    view: { def: 'library' as string, url: 'str' },
    q: { def: '', url: 'str' },
    sort: { def: 'name' as string, url: 'str' },
    chip: { def: 'all' as string, url: 'str' },
    closedCats: { def: [] as number[] } // collapsed category groups (0 = without category)
  });
  const view = $derived<View>(VIEWS.includes(ps.view as View) ? (ps.view as View) : 'library');
  const sort = $derived(SORTS[view].includes(ps.sort) ? ps.sort : 'name');
  const DISPLAY = { description: true, items: true, used: true, lastUsed: true };
  const disp = $derived(displayOf('templates.library', DISPLAY));
  const admin = $derived(isAdmin());
  const writer = $derived(canWrite());

  // ---- data ------------------------------------------------------------------------
  let templates = $state<TemplateSummary[] | null>(null);
  let categories = $state<TemplateCategory[]>([]);
  let rules = $state<AutomationRule[]>([]);
  let runs = $state<AutomationRun[]>([]);
  let variables = $state<TemplateVariable[]>([]);

  const loadTemplates = () => api.get<TemplateSummary[]>('/templates').then((r) => (templates = r), (e) => toast(errorText(e)));
  const loadCategories = () => api.get<TemplateCategory[]>('/template-categories').then((r) => (categories = r), (e) => toast(errorText(e)));
  const loadAutomations = () =>
    api.get<{ rules: AutomationRule[]; runs: AutomationRun[] }>('/automations').then(
      (r) => ((rules = r.rules), (runs = r.runs)),
      (e) => toast(errorText(e))
    );
  const loadVariables = () => api.get<TemplateVariable[]>('/template-variables').then((r) => (variables = r), (e) => toast(errorText(e)));
  $effect(() => {
    loadTemplates();
    loadCategories();
    loadAutomations();
    loadVariables();
  });

  function switchView(v: string) {
    ps.view = v;
    ps.chip = 'all';
    ps.sort = 'name';
  }

  // ---- library -----------------------------------------------------------------------
  const q = $derived(ps.q.trim().toLowerCase());
  const hit = (...s: (string | undefined)[]) => !q || s.some((x) => x?.toLowerCase().includes(q));
  const byName = (a: { name: string }, b: { name: string }) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
  const libSorts: Record<string, (a: TemplateSummary, b: TemplateSummary) => number> = {
    name: byName,
    items: (a, b) => b.item_count - a.item_count || byName(a, b),
    used: (a, b) => (b.used_count ?? 0) - (a.used_count ?? 0) || byName(a, b),
    lastUsed: (a, b) => (b.last_used ?? '').localeCompare(a.last_used ?? '') || byName(a, b),
    newest: (a, b) => b.id - a.id
  };
  const shownTemplates = $derived.by(() => {
    if (!templates) return null;
    const cat = ps.chip.startsWith('cat:') ? Number(ps.chip.slice(4)) : null;
    return templates
      .filter((x) => cat === null || (x.category_id ?? 0) === cat || (cat === 0 && !categories.some((c) => c.id === x.category_id)))
      .filter((x) => hit(x.name, x.description))
      .sort(libSorts[sort] ?? byName);
  });
  const groups = $derived(shownTemplates ? groupTemplates(shownTemplates, categories) : []);
  const closed = $derived(new Set(ps.closedCats));
  const toggleGroup = (id: number) => (ps.closedCats = closed.has(id) ? ps.closedCats.filter((x) => x !== id) : [...ps.closedCats, id]);
  const groupsCollapsed = $derived(view === 'library' && allCollapsed(groups.map((g) => !closed.has(g.id))));

  function lastUsedText(iso: string | undefined): string {
    const l = lastUsed(iso, new Date());
    if (l.kind === 'never') return '–';
    if (l.kind === 'date') return l.date;
    if (l.kind === 'days') return t('lib.daysAgo', { n: l.n });
    if (l.kind === 'weeks') return t(l.n === 1 ? 'lib.weekAgo' : 'lib.weeksAgo', { n: l.n });
    return t(l.kind === 'today' ? 'due.today' : 'due.yesterday');
  }

  // ---- automations / variables (filters) --------------------------------------------
  const ruleSorts: Record<string, (a: AutomationRule, b: AutomationRule) => number> = {
    name: (a, b) => a.template_name.localeCompare(b.template_name, undefined, { sensitivity: 'base' }),
    runs: (a, b) => b.run_count - a.run_count
  };
  const shownRules = $derived(
    rules
      .filter((r) => ps.chip === 'all' || (ps.chip === 'active') === r.enabled)
      .filter((r) => hit(r.template_name, triggerTitle(r), triggerDetail(r)))
      .sort(ruleSorts[sort] ?? ruleSorts.name)
  );
  // default order: system, custom, then the integrations; by name within
  const SOURCE_ORDER = ['system', 'custom'];
  const rank = (s: string) => (SOURCE_ORDER.includes(s) ? SOURCE_ORDER.indexOf(s) : 2);
  const bySource = (a: TemplateVariable, b: TemplateVariable) => rank(a.source) - rank(b.source) || byName(a, b);
  const varSources = $derived([...new Set(variables.map((v) => v.source))].filter((s) => s !== 'system' && s !== 'custom'));
  const shownVariables = $derived(
    variables
      .filter((v) => ps.chip === 'all' || v.source === ps.chip)
      .filter((v) => hit(v.name, v.description, v.desc_key ? t(v.desc_key as Key) : ''))
      .sort(sort === 'used' ? (a, b) => (b.used_in ?? 0) - (a.used_in ?? 0) || byName(a, b) : bySource)
  );

  const chips = $derived.by((): QuickFilter[] => {
    const chip = (key: string, label: string, count: number, color?: string): QuickFilter => ({ key, label, count, active: ps.chip === key, color });
    if (view === 'library') {
      const all = templates ?? [];
      const list = [chip('all', t('lib.all'), all.length)];
      for (const c of categories) list.push(chip(`cat:${c.id}`, c.name, all.filter((x) => x.category_id === c.id).length, categoryHex(c)));
      const none = all.filter((x) => !categories.some((c) => c.id === x.category_id)).length;
      if (none && categories.length) list.push(chip('cat:0', t('cat.none'), none));
      return list;
    }
    if (view === 'automations') {
      const on = rules.filter((r) => r.enabled).length;
      return [chip('all', t('lib.all'), rules.length), chip('active', t('auto.active'), on), chip('paused', t('auto.paused'), rules.length - on)];
    }
    return [
      chip('all', t('lib.all'), variables.length),
      ...['system', 'custom', ...varSources].map((s) => chip(s, t(`var.source.${s}` as Key), variables.filter((v) => v.source === s).length))
    ];
  });

  // ---- actions -------------------------------------------------------------------------
  let newName = $state('');
  let newKind = $state<TemplateKind>('checklist');
  async function create(e: SubmitEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      const cat = ps.chip.startsWith('cat:') ? Number(ps.chip.slice(4)) : 0;
      const tpl = await api.post<Template>('/templates', { name: newName, kind: newKind });
      if (cat) await api.patch(`/templates/${tpl.id}`, { category_id: cat });
      goto(`/templates/${tpl.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }
  // "+ Template": dialog with name, type and category (preselected: the category chip)
  let adding = $state(false);
  const chipCategory = $derived(ps.chip.startsWith('cat:') ? Number(ps.chip.slice(4)) : 0);

  let menu = $state<{ template: TemplateSummary; x: number; y: number } | null>(null);
  function openMenu(e: MouseEvent, tpl: TemplateSummary, below = false) {
    e.preventDefault();
    e.stopPropagation();
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    menu = below ? { template: tpl, x: r.right - 240, y: r.bottom + 4 } : { template: tpl, x: e.clientX, y: e.clientY };
  }
  let applying = $state<TemplateSummary | null>(null);
  let managing = $state(false);
  let ruleEdit = $state<{ rule: AutomationRule | null } | null>(null);
  let varEdit = $state<{ variable: TemplateVariable | null } | null>(null);

  const primary = $derived.by(() => {
    if (view === 'library') return writer ? { label: t('ph.addTemplate'), onClick: () => (adding = true) } : undefined;
    if (view === 'automations') return admin && templates?.length ? { label: t('auto.add'), onClick: () => (ruleEdit = { rule: null }) } : undefined;
    return admin ? { label: t('var.add'), onClick: () => (varEdit = { variable: null }) } : undefined;
  });
  const searchKey: Record<View, Key> = { library: 'ph.search.templates', automations: 'auto.search', variables: 'var.search' };
  const cols = 'grid grid-cols-[minmax(0,1fr)_6rem_5rem_9rem_auto] max-md:grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4';
</script>

<svelte:head><title>{t('nav.templates')} · ProjectList</title></svelte:head>

<PageHeader
  zoomPage="templates"
  icon="fileText"
  title={t('nav.templates')}
  subtitle={t('templates.subtitle')}
  views={[
    { key: 'library', label: t('templates.view.library'), icon: 'fileText', count: templates?.length },
    { key: 'automations', label: t('templates.view.automations'), icon: 'zap', count: rules.length },
    { key: 'variables', label: t('templates.view.variables'), icon: 'braces', count: variables.length }
  ]}
  activeView={view}
  onViewChange={switchView}
  primaryAction={primary}
  reservePrimary={writer}
  searchPlaceholder={t(searchKey[view])}
  searchValue={ps.q}
  onSearch={(v) => (ps.q = v)}
  quickFilters={chips}
  onQuickFilter={(k) => (ps.chip = k)}
  onSaveView={(name) => addPageView(name, 'templates', page.url.search.slice(1))}
  sortOptions={SORTS[view].map((s) => ({ value: s, label: t(`sort.${s}` as Key) }))}
  activeSort={sort}
  onSortChange={(s) => (ps.sort = s)}
  collapsed={groupsCollapsed}
  onToggleCollapse={() => (ps.closedCats = groupsCollapsed ? [] : groups.map((g) => g.id))}
  collapseDisabled={view !== 'library' || groups.length === 0}
  displayOptions={view === 'library' ? Object.keys(DISPLAY).map((k) => ({ key: k, label: displayLabel(k), value: disp[k] })) : []}
  onDisplayChange={(k, v) => setDisplay('templates.library', k, v)} />

<ZoomArea page="templates">
{#if view === 'library'}
  <div class="surface overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800" data-testid="template-library">
    <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
      <span>{t('lib.col.template')}</span>
      <span class="max-md:hidden">{disp.items ? t('lib.col.items') : ''}</span>
      <span class="max-md:hidden">{disp.used ? t('lib.col.used') : ''}</span>
      <span class="max-md:hidden">{disp.lastUsed ? t('lib.col.lastUsed') : ''}</span>
      <span class="text-right">
        {#if admin}
          <button type="button" class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 tracking-normal normal-case hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200" onclick={() => (managing = true)} data-testid="manage-categories">
            <Icon name="settings" size={12} />{t('cat.manageShort')}
          </button>
        {/if}
      </span>
    </div>

    {#if shownTemplates === null}
      <p class="px-4 py-3 text-sm text-zinc-400">{t('common.loading')}</p>
    {:else if shownTemplates.length === 0}
      <p class="px-4 py-3 text-sm text-zinc-400">{t('templates.empty')}</p>
    {:else}
      {#each groups as g (g.id)}
        {@const open = !closed.has(g.id)}
        {#if categories.length}
          <button
            type="button"
            class="flex w-full items-center gap-2.5 border-b border-zinc-100 bg-zinc-50/70 px-4 py-2 text-left font-semibold hover:bg-zinc-100/70 dark:border-zinc-800/70 dark:bg-zinc-900/40 dark:hover:bg-zinc-800/50"
            onclick={() => toggleGroup(g.id)}
            aria-expanded={open}
            data-testid="category-group">
            <Icon name="chevronDown" size={14} class="text-zinc-400 transition-transform {open ? '' : '-rotate-90'}" />
            {#if g.category}<span class="size-2.5 rounded-sm" style:background-color={categoryHex(g.category)}></span>{/if}
            <span>{g.category?.name ?? t('cat.none')}</span>
            <span class="text-sm font-normal text-zinc-400">{g.templates.length}</span>
          </button>
        {/if}
        {#if open}
          {#each g.templates as tpl (tpl.id)}
            <div
              class="{cols} group cursor-pointer border-b border-zinc-100 px-4 py-2.5 hover:bg-zinc-50 dark:border-zinc-800/70 dark:hover:bg-zinc-800/40"
              role="link"
              tabindex="0"
              onclick={() => goto(`/templates/${tpl.id}`)}
              onkeydown={(e) => e.key === 'Enter' && goto(`/templates/${tpl.id}`)}
              oncontextmenu={(e) => openMenu(e, tpl)}
              data-testid="template-row">
              <span class="flex min-w-0 items-center gap-3 {categories.length ? 'pl-5' : ''}">
                <span class="tpl-icon" style:--c={categoryHex(categories.find((c) => c.id === tpl.category_id))}><Icon name={tpl.kind === 'table' ? 'sheet' : 'fileText'} size={16} /></span>
                <span class="min-w-0">
                  <a href="/templates/{tpl.id}" class="block truncate font-semibold hover:underline" onclick={(e) => e.stopPropagation()}>{tpl.name}</a>
                  {#if tpl.description && disp.description}<span class="block truncate text-[13px] text-zinc-500">{tpl.description}</span>{/if}
                </span>
              </span>
              <span class="text-sm max-md:hidden">{disp.items ? templateSizeText(tpl) : ''}</span>
              <span class="text-sm font-semibold tabular-nums max-md:hidden">{disp.used ? `${tpl.used_count ?? 0}×` : ''}</span>
              <span class="text-sm text-zinc-500 max-md:hidden">{disp.lastUsed ? lastUsedText(tpl.last_used) : ''}</span>
              <span class="flex items-center justify-end gap-1.5">
                {#if writer}
                  <button type="button" class="btn-outline py-1!" onclick={(e) => (e.stopPropagation(), (applying = tpl))} data-testid="template-use">{t('lib.use')}</button>
                {/if}
                <button type="button" class="btn-outline px-1.5! py-1!" onclick={(e) => openMenu(e, tpl, true)} aria-label="{t('menu.actions')}: {tpl.name}"><Icon name="dots" size={15} /></button>
              </span>
            </div>
          {/each}
        {/if}
      {/each}
    {/if}

    {#if writer}
      <form onsubmit={create} class="flex items-center gap-2 px-4 py-2.5">
        <Icon name="plus" size={15} class="text-zinc-400" />
        <input class="flex-1 bg-transparent py-0.5 text-sm outline-none placeholder:text-zinc-400" placeholder={t('templates.new')} bind:value={newName} aria-label={t('templates.new')} />
        <!-- checklist (items) or table (spreadsheet) -->
        <select class="rounded-md bg-transparent py-0.5 text-sm text-zinc-500 outline-none hover:text-zinc-800 dark:hover:text-zinc-200" bind:value={newKind} aria-label={t('templates.kindLabel')} data-testid="template-kind">
          <option value="checklist">{t('templates.kind.checklist')}</option>
          <option value="table">{t('templates.kind.table')}</option>
        </select>
        {#if newName.trim()}<button class="btn-primary py-1!" type="submit">{t('common.create')}</button>{/if}
      </form>
    {/if}
  </div>
{:else if view === 'automations'}
  <AutomationsView rules={shownRules} {runs} templates={templates ?? []} {categories} {admin} onedit={(r) => (ruleEdit = { rule: r })} onchange={loadAutomations} />
{:else}
  <VariablesView variables={shownVariables} {admin} onedit={(v) => (varEdit = { variable: v })} />
{/if}
</ZoomArea>

{#if menu}
  {@const m = menu}
  <TemplateMenu
    template={m.template}
    x={m.x}
    y={m.y}
    {categories}
    onclose={() => (menu = null)}
    onuse={() => (applying = m.template)}
    oncategory={(id) => (m.template.category_id = id)}
    onrenamed={(n) => (m.template.name = n)}
    ondeleted={() => (templates = templates?.filter((x) => x.id !== m.template.id) ?? null)} />
{/if}
{#if applying}
  <ApplyTemplateDialog template={applying} onclose={() => ((applying = null), loadTemplates())} />
{/if}
{#if adding}
  <NewTemplateDialog {categories} categoryId={chipCategory} onclose={() => (adding = false)} />
{/if}
{#if managing}
  <CategoryDialog {categories} onchange={() => (loadCategories(), loadTemplates())} onclose={() => (managing = false)} />
{/if}
{#if ruleEdit}
  <RuleDialog rule={ruleEdit.rule} templates={(templates ?? []).filter((x) => x.kind !== 'table')} onsaved={loadAutomations} onclose={() => (ruleEdit = null)} />
{/if}
{#if varEdit}
  <VariableDialog variable={varEdit.variable} onsaved={loadVariables} onclose={() => (varEdit = null)} />
{/if}
