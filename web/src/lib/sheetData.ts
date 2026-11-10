// Workbook data of tables (spreadsheets): the format stored on the server and
// the conversions around Fortune-sheet. Pure functions (tested in sheetData.test.ts).
//
// Stored format: an array of sheets like Fortune-sheet's `Sheet` with the cells
// as sparse `celldata` ([{r, c, v: cell}]), never the dense `data` matrix the
// editor works with, and without view state (selection, scroll position).

/** Cell as Fortune-sheet stores it (v value, m display text, f formula "=…", styles). */
export interface SheetCell {
  v?: string | number | boolean | null;
  m?: string | number | null;
  f?: string;
  ct?: { fa?: string; t?: string; s?: { v?: string }[] };
  mc?: { r: number; c: number; rs?: number; cs?: number };
  bg?: string; // background #rrggbb
  fc?: string; // font color
  bl?: number; // bold
  it?: number; // italic
  cl?: number; // strike-through
  un?: number; // underline
  fs?: number; // font size (pt)
  ff?: string | number; // font family
  ht?: number; // horizontal: 0 center, 1 left, 2 right
  vt?: number; // vertical: 0 middle, 1 top, 2 bottom
  tb?: string; // text wrap: "2" = wrap
  [k: string]: unknown;
}

export interface StoredCell {
  r: number;
  c: number;
  v: SheetCell | null;
}

export interface SheetConfig {
  merge?: Record<string, { r: number; c: number; rs: number; cs: number }>;
  columnlen?: Record<string, number>; // column width (px)
  rowlen?: Record<string, number>; // row height (px)
  [k: string]: unknown;
}

export interface StoredSheet {
  name: string;
  id?: string;
  order?: number;
  status?: number; // 1 = the active sheet
  row?: number; // number of rows shown
  column?: number;
  celldata?: StoredCell[];
  config?: SheetConfig;
  [k: string]: unknown;
}

/** A sheet as the editor hands it out (dense matrix in `data`). */
export interface EditorSheet extends StoredSheet {
  data?: (SheetCell | null)[][];
}

// view state and derived data of the editor that is not stored
const TRANSIENT = new Set([
  'data',
  'luckysheet_select_save',
  'luckysheet_selection_range',
  'jfgird_select_save',
  'status', // active sheet: switching sheets is no change (the first one opens)
  'visibledatarow',
  'visibledatacolumn',
  'ch_width',
  'rh_height',
  'scrollLeft',
  'scrollTop',
  'zoomRatio'
]);

const hasContent = (cell: SheetCell | null | undefined): cell is SheetCell => !!cell && Object.keys(cell).some((k) => cell[k] !== undefined && cell[k] !== null);

/** Editor sheets → stored format (matrix to celldata, view state removed). */
export function toStored(sheets: readonly EditorSheet[]): StoredSheet[] {
  return sheets.map((sheet, i) => {
    const out: StoredSheet = { name: sheet.name || `Sheet${i + 1}` };
    for (const [k, v] of Object.entries(sheet)) if (!TRANSIENT.has(k) && k !== 'celldata' && k !== 'name' && v !== undefined) out[k] = v;
    if (Array.isArray(sheet.data)) {
      const cells: StoredCell[] = [];
      sheet.data.forEach((row, r) => row?.forEach((cell, c) => hasContent(cell) && cells.push({ r, c, v: cell })));
      out.celldata = cells;
    } else {
      out.celldata = (sheet.celldata ?? []).filter((x) => hasContent(x.v));
    }
    return out;
  });
}

/** Stored sheets → editor input: every sheet gets an id and an order, exactly one is active. */
export function toEditor(sheets: readonly StoredSheet[], newId: () => string): StoredSheet[] {
  const list = sheets.length ? sheets : [{ name: 'Sheet1', celldata: [] }];
  const active = Math.max(0, list.findIndex((s) => s.status === 1));
  return list.map((s, i) => ({ ...s, id: s.id || newId(), order: s.order ?? i, status: i === active ? 1 : 0, celldata: s.celldata ?? [] }));
}

/** The text shown in a cell (display text, else the value; rich text joined). */
export function cellText(cell: SheetCell | null | undefined): string {
  if (!cell) return '';
  if (cell.ct?.t === 'inlineStr' && Array.isArray(cell.ct.s)) return cell.ct.s.map((x) => x?.v ?? '').join('');
  const m = cell.m ?? cell.v;
  return m === null || m === undefined ? '' : String(m);
}

/** Display texts of a sheet as rows (trailing empty rows and columns removed). */
export function sheetRows(sheet: StoredSheet): string[][] {
  const rows: string[][] = [];
  for (const { r, c, v } of sheet.celldata ?? []) {
    const text = cellText(v);
    if (!text) continue;
    while (rows.length <= r) rows.push([]);
    rows[r][c] = text;
  }
  const width = Math.max(0, ...rows.map((r) => r.length));
  return rows.map((r) => Array.from({ length: width }, (_, i) => r[i] ?? ''));
}

/** File name from a table title (without characters that file systems reject). */
export function fileName(title: string, ext: string): string {
  const base = title.replace(/[\\/:*?"<>|\x00-\x1f]+/g, ' ').replace(/\s+/g, ' ').trim() || 'table';
  return `${base.slice(0, 120)}.${ext}`;
}

/** Size of the workbook as stored (bytes of the JSON). */
export const storedSize = (sheets: readonly StoredSheet[]) => new TextEncoder().encode(JSON.stringify(sheets)).length;
export const MAX_STORED = 8 << 20; // server limit (internal/httpapi/handlers_tables.go)
