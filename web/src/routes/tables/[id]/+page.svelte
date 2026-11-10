<script lang="ts">
  // Table (spreadsheet) of a project: title, last change, export (CSV of the
  // sheet shown, XLSX of all sheets), save as template; the editor saves on its
  // own (debounced). Saving sends the version that was loaded: when someone else
  // saved in between, the server answers 409 and the user chooses between their
  // version (overwrite) and the current one (reload).
  import { page } from '$app/state';
  import { beforeNavigate, goto } from '$app/navigation';
  import { api, ApiError, errorText } from '$lib/api';
  import { canWrite, refreshSidebar, saveState, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { relativeTime } from '$lib/activity';
  import { download, toCsv } from '$lib/reporting';
  import { fileName, MAX_STORED, sheetRows, storedSize, type StoredSheet } from '$lib/sheetData';
  import type { Template, TableDetail } from '$lib/types';
  import DetailHeader from '$lib/components/DetailHeader.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import Spreadsheet from '$lib/components/Spreadsheet.svelte';
  import Icon from '$lib/components/Icon.svelte';

  let data = $state<TableDetail | null>(null);
  // the workbook as loaded: not deeply reactive (can be large; the editor keeps its own state)
  let workbook = $state.raw<StoredSheet[]>([]);
  let notFound = $state(false);
  let sheet = $state<Spreadsheet>();
  let saveAs = $state(false);
  let templateName = $state('');
  let conflict = $state(false);
  let loadKey = $state(0); // remounts the editor after a reload

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    saveAs = false;
    conflict = false;
    unsaved = null;
    try {
      const r = await api.get<TableDetail>(`/tables/${id}`);
      workbook = r.table.data;
      r.table.data = [];
      data = r;
      loadKey++;
    } catch {
      data = null;
      notFound = true;
    }
  }

  // ---- saving --------------------------------------------------------------------------
  let unsaved: StoredSheet[] | null = null; // latest change not saved yet
  let saving = false;
  let timer: ReturnType<typeof setTimeout> | undefined;

  function changed(sheets: StoredSheet[]) {
    unsaved = sheets;
    clearTimeout(timer);
    timer = setTimeout(save, 800);
  }

  async function save() {
    clearTimeout(timer);
    if (!data || !unsaved || saving || conflict) return;
    const sheets = unsaved;
    if (storedSize(sheets) > MAX_STORED) {
      toast(t('table.tooLarge'));
      return;
    }
    const tableId = data.table.id;
    saving = true;
    unsaved = null;
    try {
      const r = await api.put<{ version: number; updated_at: string; updated_by_name: string }>(`/tables/${tableId}/data`, { data: sheets, version: data.table.version });
      if (data?.table.id === tableId) Object.assign(data.table, { version: r.version, updated_at: r.updated_at, updated_by_name: r.updated_by_name });
    } catch (e) {
      unsaved ??= sheets; // try again with the next change
      if (e instanceof ApiError && e.status === 409) {
        conflict = true;
        saveState.error = false; // shown by the banner
      } else toast(errorText(e));
    } finally {
      saving = false;
    }
    if (unsaved && !conflict) timer = setTimeout(save, 800); // changed while saving
  }

  // conflict: overwrite with my version (on top of the current one) or load the current one
  async function keepMine() {
    if (!data) return;
    try {
      const cur = await api.get<TableDetail>(`/tables/${data.table.id}`);
      data.table.version = cur.table.version;
      unsaved = sheet?.getSheets() ?? unsaved;
      conflict = false;
      await save();
    } catch (e) {
      toast(errorText(e));
    }
  }

  // leaving: save what is pending (the request continues in the background)
  beforeNavigate(() => {
    sheet?.flush();
    if (unsaved) save();
  });
  function beforeUnload(e: BeforeUnloadEvent) {
    sheet?.flush();
    if (unsaved || saving) {
      save();
      e.preventDefault();
    }
  }

  // ---- header actions ------------------------------------------------------------------
  async function rename(title: string) {
    await api.patch(`/tables/${id}`, { title });
    if (data) data.table.title = title;
    refreshSidebar(0);
  }

  function exportCsv() {
    const s = sheet?.getActiveSheet();
    if (!s || !data) return;
    download(fileName(`${data.table.title} - ${s.name}`, 'csv'), toCsv(sheetRows(s)));
  }

  let exporting = $state(false);
  async function exportXlsx() {
    if (!sheet || !data) return;
    exporting = true;
    try {
      const { toXlsx } = await import('$lib/sheetXlsx');
      const buf = await toXlsx(sheet.getSheets());
      download(fileName(data.table.title, 'xlsx'), buf, 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet');
    } catch (e) {
      console.error(e);
      toast(t('errors.generic'));
    } finally {
      exporting = false;
    }
  }

  async function saveAsTemplate(e: SubmitEvent) {
    e.preventDefault();
    sheet?.flush();
    await save(); // the template is made from the saved table
    try {
      const tpl = await api.post<Template>(`/tables/${id}/save-as-template`, { name: templateName });
      toast(t('checklist.templateSaved', { name: tpl.name }), 'info');
      saveAs = false;
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function remove() {
    if (!data) return;
    const projectId = data.project.id;
    clearTimeout(timer);
    unsaved = null;
    try {
      await api.del(`/tables/${id}`);
      refreshSidebar(0);
      goto(`/projects/${projectId}#tables`);
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{data?.table.title ?? ''} · ProjectList</title></svelte:head>
<svelte:window onbeforeunload={beforeUnload} />

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if data}
  {@const tb = data.table}
  <DetailHeader type="table">
    {#snippet title()}
      <Editable value={tb.title} required disabled={!editable} onsave={rename} inputClass="font-serif text-2xl! font-bold" />
    {/snippet}
    {#snippet meta()}
      <span class="text-[13px] text-zinc-500 dark:text-zinc-400">
        <a href="/projects/{data!.project.id}#tables" class="hover:underline">{data!.project.name}</a>
        {#if tb.updated_at && tb.updated_by_name}· {t('table.updated', { time: relativeTime(tb.updated_at), name: tb.updated_by_name })}{/if}
      </span>
    {/snippet}
    {#snippet actions()}
      <SaveStatus />
      <button type="button" class="btn-outline ml-2" onclick={exportCsv} title={t('table.csvHint')} data-testid="table-export-csv">
        <Icon name="download" size={15} />{t('table.exportCsv')}
      </button>
      <button type="button" class="btn-outline" onclick={exportXlsx} disabled={exporting} data-testid="table-export-xlsx">
        <Icon name="download" size={15} />{t('table.exportXlsx')}
      </button>
      {#if editable}
        <button type="button" class="btn-outline" onclick={() => ((saveAs = !saveAs), (templateName = tb.title))}>{t('checklist.saveAsTemplate')}</button>
      {/if}
    {/snippet}
  </DetailHeader>

  {#if saveAs}
    <form onsubmit={saveAsTemplate} class="mb-4 flex max-w-xl gap-2">
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" placeholder={t('checklist.templateName')} bind:value={templateName} required autofocus />
      <button class="btn-primary" type="submit">{t('common.save')}</button>
      <button class="btn-ghost" type="button" onclick={() => (saveAs = false)}>{t('common.cancel')}</button>
    </form>
  {/if}

  {#if conflict}
    <div class="mb-4 flex flex-wrap items-center gap-3 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700/60 dark:bg-amber-950/40 dark:text-amber-200" role="alert" data-testid="table-conflict">
      <Icon name="alert" size={16} />
      <span class="min-w-0 flex-1">{t('table.conflict')}</span>
      <button type="button" class="btn-outline py-1!" onclick={() => load(id)}>{t('table.conflictReload')}</button>
      <button type="button" class="btn-primary py-1!" onclick={keepMine}>{t('table.conflictKeep')}</button>
    </div>
  {/if}

  {#key `${tb.id}:${loadKey}`}
    <Spreadsheet bind:this={sheet} data={workbook} {editable} onchange={changed} class="h-[calc(100dvh-15rem)] min-h-[24rem]" />
  {/key}
  <p class="mt-2 text-xs text-zinc-400">{t('table.formulaHint')}</p>

  {#if editable}
    <div class="mt-10 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <ConfirmButton onconfirm={remove} label={t('table.delete')} confirmLabel={t('table.deleteConfirm')} />
    </div>
  {/if}
{/if}
