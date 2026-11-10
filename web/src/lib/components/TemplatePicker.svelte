<script lang="ts" generics="K extends TemplateKind = 'checklist'">
  // "New checklist / table from template" with a preview (items or sheets).
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { htmlToText } from '$lib/rich';
  import { sheetsText } from '$lib/templateText';
  import type { StoredSheet } from '$lib/sheetData';
  import type { Checklist, Table, Template, TemplateItem, TemplateKind, TemplateSummary, TemplateVariable } from '$lib/types';
  import EntityIcon from './EntityIcon.svelte';

  let {
    projectId,
    kind = 'checklist' as K,
    oncreated,
    oncancel
  }: {
    projectId: number;
    kind?: K;
    oncreated: (created: K extends 'table' ? Table : Checklist) => void;
    oncancel: () => void;
  } = $props();

  const isTable = $derived(kind === 'table');
  let templates = $state<TemplateSummary[] | null>(null);
  let selected = $state<number | null>(null);
  let preview = $state<TemplateItem[]>([]);
  let sheets = $state<StoredSheet[]>([]);
  let title = $state('');
  // custom variables of the selected template (asked, default prefilled)
  let vars = $state<TemplateVariable[]>([]);
  let values = $state<Record<string, string>>({});
  let busy = $state(false);

  $effect(() => {
    api.get<TemplateSummary[]>('/templates').then(
      (tpl) => (templates = tpl.filter((x) => (x.kind ?? 'checklist') === kind)),
      (e) => toast(errorText(e))
    );
  });

  async function select(tpl: TemplateSummary) {
    selected = tpl.id;
    title = tpl.name;
    preview = [];
    sheets = [];
    vars = [];
    try {
      const [r, v] = await Promise.all([
        api.get<{ template: Template; items: TemplateItem[] }>(`/templates/${tpl.id}`),
        api.get<TemplateVariable[]>(`/templates/${tpl.id}/variables`)
      ]);
      if (selected !== tpl.id) return;
      preview = r.items;
      try {
        sheets = JSON.parse(r.template.sheet_data || '[]');
      } catch {
        sheets = [];
      }
      vars = v.filter((x) => x.source !== 'system' && x.source !== 'unknown');
      values = Object.fromEntries(vars.map((x) => [x.name, x.default_value]));
    } catch (e) {
      toast(errorText(e));
    }
  }

  async function create() {
    if (!selected) return;
    busy = true;
    try {
      const url = `/projects/${projectId}/${isTable ? 'tables' : 'checklists'}/from-template`;
      oncreated(await api.post<K extends 'table' ? Table : Checklist>(url, { template_id: selected, title, values }));
    } catch (e) {
      toast(errorText(e));
    } finally {
      busy = false;
    }
  }
</script>

<div class="rounded-lg border border-zinc-200 p-4 dark:border-zinc-800">
  <div class="mb-3 flex items-center justify-between">
    <h3 class="font-serif text-[17px] font-bold">{t(isTable ? 'table.pickerTitle' : 'picker.title')}</h3>
    <button type="button" class="btn-ghost" onclick={oncancel}>{t('common.cancel')}</button>
  </div>

  {#if templates === null}
    <p class="text-sm text-zinc-400">{t('common.loading')}</p>
  {:else if templates.length === 0}
    <p class="text-sm text-zinc-500">
      {t(isTable ? 'table.noTemplates' : 'picker.noTemplates')} <a href="/templates" class="text-accent-600 hover:underline">{t('picker.createTemplates')}</a>
    </p>
  {:else}
    <div class="grid gap-4 sm:grid-cols-2">
      <ul class="max-h-72 space-y-px overflow-y-auto">
        {#each templates as tpl (tpl.id)}
          <li>
            <button
              type="button"
              onclick={() => select(tpl)}
              class="flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-left text-sm {selected === tpl.id
                ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'}">
              <span class="flex min-w-0 items-center gap-2"><EntityIcon type="template" size="xs" /><span class="truncate">{tpl.name}</span></span>
              {#if !isTable}<span class="text-xs text-zinc-400 tabular-nums">{tpl.item_count}</span>{/if}
            </button>
          </li>
        {/each}
      </ul>
      <div class="rounded-md bg-zinc-50 p-3 dark:bg-zinc-900">
        {#if selected}
          <p class="label">{t('picker.preview')}</p>
          {#if isTable}
            <p class="mb-1 text-xs text-zinc-500">{sheetsText(sheets.length)}</p>
            <ul class="max-h-48 space-y-0.5 overflow-y-auto text-sm">
              {#each sheets as s, i (i)}
                <li class="flex items-center gap-2"><EntityIcon type="table" size="xs" /><span class="truncate">{s.name}</span></li>
              {/each}
            </ul>
          {:else}
            <ul class="max-h-48 space-y-0.5 overflow-y-auto text-sm">
              {#each preview as it (it.id)}
                {#if it.kind === 'h1' || it.kind === 'h2' || it.kind === 'h3' || it.kind === 'h4'}
                  <li class="pt-1.5 font-serif font-bold">{htmlToText(it.title)}</li>
                {:else}
                  <li class="flex gap-2"><span class="mt-[5px] size-2.5 shrink-0 rounded-full border border-zinc-300 dark:border-zinc-600"></span>{htmlToText(it.title)}</li>
                {/if}
              {:else}
                <li class="text-zinc-400">{t('picker.empty')}</li>
              {/each}
            </ul>
          {/if}
          <div class="mt-3">
            <label class="label" for="tp-title">{t(isTable ? 'table.pickerTitleLabel' : 'picker.titleLabel')}</label>
            <input id="tp-title" class="input" bind:value={title} />
          </div>
          {#each vars as v (v.name)}
            <label class="mt-2 block">
              <span class="mb-0.5 flex items-center gap-2 text-[13px]"><code class="var-chip">{`{{${v.name}}}`}</code><span class="truncate text-zinc-500">{v.description}</span></span>
              <input class="input py-1! text-[13px]!" bind:value={values[v.name]} maxlength="500" />
            </label>
          {/each}
          <button type="button" class="btn-primary mt-3 w-full" disabled={busy} onclick={create}>{t(isTable ? 'table.pickerCreate' : 'picker.create')}</button>
          <p class="mt-2 text-xs text-zinc-500">{t(isTable ? 'table.pickerSnapshot' : 'picker.snapshot')}</p>
        {:else}
          <p class="text-sm text-zinc-400">{t('picker.select')}</p>
        {/if}
      </div>
    </div>
  {/if}
</div>
