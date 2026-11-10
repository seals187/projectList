<script lang="ts">
  // Spreadsheet editor (tables and table templates): Fortune-sheet, a React
  // component, mounted into this element. React, Fortune-sheet and its CSS are
  // loaded on demand, only where a table is shown.
  //
  // onchange gets the workbook in the stored format ($lib/sheetData) after a real
  // change by the user: changes the editor makes by itself while loading (ids,
  // calculated values) or view changes (selection, switching sheets) are not
  // reported, so opening a table never saves it.
  import { onMount } from 'svelte';
  import type { WorkbookInstance } from '@fortune-sheet/react';
  import { t } from '$lib/i18n.svelte';
  import { toEditor, toStored, type EditorSheet, type StoredSheet } from '$lib/sheetData';

  let {
    data,
    editable,
    onchange,
    class: cls = ''
  }: {
    data: StoredSheet[]; // initial workbook (remount with {#key} to load another one)
    editable: boolean;
    onchange?: (sheets: StoredSheet[]) => void;
    class?: string;
  } = $props();

  let host = $state<HTMLDivElement>();
  let failed = $state(false);
  let ready = $state(false);
  // React replaces the instance behind the ref on every change of the editor: always read ref.current
  let ref: { readonly current: WorkbookInstance | null } | null = null;
  const wb = () => ref?.current ?? null;

  // toolbar and menus without image/screenshot/chart (not stored, no CSP-safe export)
  const TOOLBAR = [
    'undo', 'redo', 'format-painter', 'clear-format', '|',
    'currency-format', 'percentage-format', 'number-decrease', 'number-increase', 'format', '|',
    'font', '|', 'font-size', '|', 'bold', 'italic', 'strike-through', 'underline', '|',
    'font-color', 'background', 'border', 'merge-cell', '|',
    'horizontal-align', 'vertical-align', 'text-wrap', 'text-rotation', '|',
    'freeze', 'conditionFormat', 'filter', 'link', 'comment', 'quick-formula', 'dataVerification', 'splitColumn', 'search'
  ];
  const CELL_MENU = [
    'copy', 'paste', '|', 'insert-row', 'insert-column', 'delete-row', 'delete-column', 'delete-cell',
    'hide-row', 'hide-column', 'set-row-height', 'set-column-width', '|', 'clear', 'sort', 'orderAZ', 'orderZA', 'filter', 'link', 'data', 'cell-format'
  ];

  // ids for new sheets (crypto.randomUUID needs a secure context, the app may run on plain HTTP)
  const newId = () => Math.random().toString(36).slice(2, 10) + Date.now().toString(36);

  // change detection: compare the stored form; before the first user input only remember it
  let interacted = false;
  let baseline = '';
  let pending: EditorSheet[] | null = null;
  let timer: ReturnType<typeof setTimeout> | undefined;
  function compare() {
    if (!pending) return;
    const stored = toStored(pending);
    pending = null;
    const json = JSON.stringify(stored);
    if (json === baseline) return;
    baseline = json;
    if (interacted) onchange?.(stored);
  }
  function changed(sheets: EditorSheet[]) {
    pending = sheets;
    clearTimeout(timer);
    timer = setTimeout(compare, interacted ? 250 : 0);
  }
  const touch = () => (interacted = true);

  /** The whole workbook in the stored format (also pending changes). */
  export function getSheets(): StoredSheet[] {
    const w = wb();
    return w ? toStored(w.getAllSheets() as EditorSheet[]) : ($state.snapshot(data) as StoredSheet[]);
  }
  /** The sheet currently shown (for the CSV export). */
  export function getActiveSheet(): StoredSheet | null {
    const w = wb();
    if (!w) return data[0] ?? null;
    const s = w.getSheet() as EditorSheet;
    return toStored([s])[0];
  }
  /** Report a change that is still waiting for the comparison right away (before leaving the page). */
  export function flush() {
    clearTimeout(timer);
    compare();
  }

  onMount(() => {
    let unmount: (() => void) | undefined;
    let cancelled = false;
    (async () => {
      try {
        const [React, ReactDOM, { Workbook }] = await Promise.all([import('react'), import('react-dom/client'), import('@fortune-sheet/react')]);
        await import('@fortune-sheet/react/dist/index.css');
        if (cancelled || !host) return;
        const r = React.createRef<WorkbookInstance>();
        ref = r;
        const root = ReactDOM.createRoot(host);
        root.render(
          React.createElement(Workbook, {
            ref: r,
            // a plain copy: the editor freezes and changes its data (immer), a $state proxy would break that
            data: toEditor($state.snapshot(data) as StoredSheet[], newId) as never,
            onChange: (sheets) => changed(sheets as EditorSheet[]),
            allowEdit: editable,
            showToolbar: editable,
            showFormulaBar: true,
            showSheetTabs: true,
            lang: 'en',
            currency: 'CHF',
            toolbarItems: TOOLBAR,
            cellContextMenu: CELL_MENU,
            generateSheetId: newId
          })
        );
        unmount = () => root.unmount();
        // the instance is set after React has rendered; the state the editor has made
        // of the data (before any input) is the baseline for the change detection
        const wait = () => {
          if (cancelled) return;
          if (!r.current) return void requestAnimationFrame(wait);
          ready = true;
          requestAnimationFrame(() => {
            const w = wb();
            if (w && !interacted && !baseline) baseline = JSON.stringify(toStored(w.getAllSheets() as EditorSheet[]));
          });
        };
        wait();
      } catch (e) {
        console.error(e);
        failed = true;
      }
    })();
    return () => {
      cancelled = true;
      flush();
      clearTimeout(timer);
      ref = null;
      unmount?.();
    };
  });
</script>

<div class="spreadsheet relative overflow-hidden rounded-xl border border-zinc-200 bg-white text-zinc-900 dark:border-zinc-700 {cls}" data-testid="spreadsheet">
  <!-- user input inside the editor (also menus and dialogs it opens) marks the following changes as real -->
  <div bind:this={host} class="absolute inset-0" onpointerdown={touch} onkeydown={touch} onpaste={touch} ondrop={touch} role="presentation"></div>
  {#if failed}
    <p class="absolute inset-0 flex items-center justify-center p-6 text-sm text-red-600">{t('table.loadFailed')}</p>
  {:else if !ready}
    <p class="pointer-events-none absolute inset-0 flex items-center justify-center text-sm text-zinc-400">{t('common.loading')}</p>
  {/if}
</div>

<style>
  /* Fortune-sheet brings its own (light) styles; keep the app font in its chrome */
  .spreadsheet :global(.fortune-toolbar),
  .spreadsheet :global(.fortune-sheet-container) {
    font-family: inherit;
  }
</style>
