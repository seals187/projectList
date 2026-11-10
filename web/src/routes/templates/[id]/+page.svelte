<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { Ops, Template, TemplateItem } from '$lib/types';
  import DetailHeader from '$lib/components/DetailHeader.svelte';
  import BlockEditor from '$lib/components/BlockEditor.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import FavoriteStar from '$lib/components/FavoriteStar.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import Spreadsheet from '$lib/components/Spreadsheet.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import type { StoredSheet } from '$lib/sheetData';
  import { readXlsx } from '$lib/xlsxImport';
  import { tick } from 'svelte';

  let template = $state<Template | null>(null);
  let items = $state<TemplateItem[]>([]);
  let notFound = $state(false);

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    try {
      const r = await api.get<{ template: Template; items: TemplateItem[] }>(`/templates/${id}`);
      template = r.template;
      items = r.items;
    } catch {
      notFound = true;
    }
  }

  async function save(body: Partial<Template>) {
    template = await api.patch<Template>(`/templates/${id}`, body);
  }

  // table templates: the workbook (read when the editor opens), saved debounced on changes
  const sheets = $derived.by((): StoredSheet[] => {
    try {
      return JSON.parse(template?.sheet_data || '[]');
    } catch {
      return [];
    }
  });
  let sheetTimer: ReturnType<typeof setTimeout> | undefined;
  let ignoreChanges = false; // while the editor is replaced after an import
  function sheetChanged(data: StoredSheet[]) {
    if (ignoreChanges) return;
    clearTimeout(sheetTimer);
    sheetTimer = setTimeout(() => api.put(`/templates/${id}/sheet`, { data }).catch((e) => toast(errorText(e))), 800);
  }

  // Excel file → replaces the workbook of the table template (the editor is reloaded;
  // what the old editor still reports while closing is not saved)
  let fileInput = $state<HTMLInputElement>();
  let importing = $state(false);
  let editorKey = $state(0);
  async function importXlsx(file: File | undefined) {
    if (!file || !template) return;
    importing = true;
    try {
      const data = await readXlsx(file);
      if (!data) return;
      clearTimeout(sheetTimer);
      await api.put(`/templates/${id}/sheet`, { data });
      ignoreChanges = true;
      template.sheet_data = JSON.stringify(data);
      editorKey++;
      await tick();
      toast(t('template.imported'), 'info');
    } catch (e) {
      toast(errorText(e));
    } finally {
      ignoreChanges = false;
      importing = false;
      if (fileInput) fileInput.value = '';
    }
  }

  const ops: Ops<TemplateItem> = {
    create: (body) => api.post<TemplateItem>(`/templates/${id}/items`, body),
    patch: (itemId, body) => api.patch<TemplateItem>(`/template-items/${itemId}`, body),
    remove: (itemId) => api.del(`/template-items/${itemId}`),
    reorder: (ids) => api.put(`/templates/${id}/items/order`, { ids })
  };

  async function remove() {
    try {
      await api.del(`/templates/${id}`);
      goto('/templates');
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{template?.name ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if template}
  <DetailHeader type="template">
    {#snippet actions()}
      <SaveStatus />
      {#if template!.kind === 'table' && editable}
        <!-- first click asks (the content is replaced), the second opens the file selection -->
        <ConfirmButton class="btn-outline ml-2" onconfirm={() => fileInput?.click()} label={t('template.importXlsx')} confirmLabel={t('template.importConfirm')}>
          <Icon name="download" size={15} class="rotate-180" />{importing ? t('table.importing') : t('template.importXlsx')}
        </ConfirmButton>
        <input bind:this={fileInput} type="file" accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" class="hidden" onchange={(e) => importXlsx(e.currentTarget.files?.[0])} data-testid="template-import-file" />
      {/if}
    {/snippet}
    {#snippet meta()}<FavoriteStar kind="template" id={template!.id} />{/snippet}
    {#snippet title()}
      <Editable value={template!.name} required disabled={!editable} onsave={(name) => save({ name })} inputClass="font-serif text-2xl! font-bold" />
    {/snippet}
  </DetailHeader>

  <div class="mb-6 max-w-2xl text-sm text-zinc-600 dark:text-zinc-300">
    <Editable value={template.description} multiline placeholder={t('template.descPlaceholder')} disabled={!editable} onsave={(description) => save({ description })} />
  </div>

  {#if template.kind === 'table'}
    {#key `${template.id}:${editorKey}`}
      <Spreadsheet data={sheets} {editable} onchange={sheetChanged} class="h-[calc(100dvh-18rem)] min-h-[24rem]" />
    {/key}
    <p class="mt-2 text-xs text-zinc-400">{t('template.tableHint')} {t('table.formulaHint')}</p>
  {:else}
    {#key template.id}
      <BlockEditor bind:items {ops} {editable} collapseKey="template:{id}" onerror={() => load(id)} />
    {/key}

    <p class="mt-6 text-xs text-zinc-400">{t('template.hint')}</p>
  {/if}

  {#if editable}
    <div class="mt-16 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <ConfirmButton onconfirm={remove} label={t('template.delete')} confirmLabel={t('template.deleteConfirm')} />
    </div>
  {/if}
{/if}
