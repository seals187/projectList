// XLSX export and import of tables with ExcelJS (loaded on demand: ~1 MB).
// Values, formulas (with their last result), number formats, fonts, fill
// colors, alignment, merged cells, column widths, row heights and frozen panes
// are converted; borders, comments, conditional formats and charts are not.
import type { Cell, CellValue, Workbook, Worksheet } from 'exceljs';
import { cellText, type SheetCell, type SheetConfig, type StoredCell, type StoredSheet } from './sheetData';

type ExcelModule = typeof import('exceljs');

/** ExcelJS (CommonJS: the module object is the default export after bundling). */
export async function loadExcel(): Promise<ExcelModule> {
  const mod = (await import('exceljs')) as ExcelModule & { default?: ExcelModule };
  return mod.default ?? mod;
}

// ---- colors ------------------------------------------------------------------------

/** "#rgb", "#rrggbb" or "rgb(r, g, b)" → "FFRRGGBB" (null when unknown). */
export function toArgb(color: string | undefined): string | null {
  if (!color) return null;
  let m = /^#([0-9a-f]{3})$/i.exec(color.trim());
  if (m) return ('FF' + [...m[1]].map((ch) => ch + ch).join('')).toUpperCase();
  m = /^#([0-9a-f]{6})$/i.exec(color.trim());
  if (m) return ('FF' + m[1]).toUpperCase();
  const rgb = /^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i.exec(color.trim());
  if (rgb) return 'FF' + rgb.slice(1, 4).map((n) => Math.min(255, Number(n)).toString(16).padStart(2, '0')).join('').toUpperCase();
  return null;
}

/** "FFRRGGBB" → "#rrggbb" (null without an explicit color, e.g. theme colors). */
export function fromArgb(argb: string | undefined): string | null {
  if (!argb || !/^[0-9a-f]{8}$/i.test(argb)) return null;
  return '#' + argb.slice(2).toLowerCase();
}

// ---- dates -------------------------------------------------------------------------

const EXCEL_EPOCH = Date.UTC(1899, 11, 30);
export const dateToSerial = (d: Date) => (d.getTime() - EXCEL_EPOCH) / 86_400_000;

// ---- export ------------------------------------------------------------------------

const H_ALIGN: Record<number, 'center' | 'left' | 'right'> = { 0: 'center', 1: 'left', 2: 'right' };
const V_ALIGN: Record<number, 'middle' | 'top' | 'bottom'> = { 0: 'middle', 1: 'top', 2: 'bottom' };

/** Excel sheet names: max. 31 characters, no []:*?/\ and unique. */
export function sheetName(name: string, used: Set<string>): string {
  let base = (name || 'Sheet').replace(/[[\]:*?/\\]/g, ' ').trim().slice(0, 31) || 'Sheet';
  let n = 2;
  let out = base;
  while (used.has(out.toLowerCase())) out = `${base.slice(0, 31 - String(n).length - 1)} ${n++}`;
  used.add(out.toLowerCase());
  return out;
}

function plainValue(v: SheetCell): string | number | boolean | null {
  if (v.ct?.t === 'inlineStr') return cellText(v);
  if (typeof v.v === 'number' || typeof v.v === 'boolean') return v.v;
  if (typeof v.v === 'string') return v.v;
  if (v.m !== undefined && v.m !== null) return String(v.m);
  return null;
}

function writeCell(cell: Cell, v: SheetCell) {
  const value = plainValue(v);
  if (v.f) cell.value = { formula: v.f.replace(/^=/, ''), result: value ?? undefined } as CellValue;
  else if (value !== null) cell.value = value;
  if (v.ct?.fa && v.ct.fa !== 'General' && v.ct.fa !== '@') cell.numFmt = v.ct.fa;
  const color = toArgb(v.fc);
  const font = {
    ...(v.bl === 1 && { bold: true }),
    ...(v.it === 1 && { italic: true }),
    ...(v.cl === 1 && { strike: true }),
    ...(v.un === 1 && { underline: true }),
    ...(color && { color: { argb: color } }),
    ...(typeof v.fs === 'number' && { size: v.fs }),
    ...(typeof v.ff === 'string' && v.ff && { name: v.ff })
  };
  if (Object.keys(font).length) cell.font = font;
  const bg = toArgb(v.bg);
  if (bg) cell.fill = { type: 'pattern', pattern: 'solid', fgColor: { argb: bg } };
  const alignment = {
    ...(typeof v.ht === 'number' && H_ALIGN[v.ht] && { horizontal: H_ALIGN[v.ht] }),
    ...(typeof v.vt === 'number' && V_ALIGN[v.vt] && { vertical: V_ALIGN[v.vt] }),
    ...(v.tb === '2' && { wrapText: true })
  };
  if (Object.keys(alignment).length) cell.alignment = alignment;
}

function writeSheet(ws: Worksheet, sheet: StoredSheet) {
  for (const { r, c, v } of sheet.celldata ?? []) {
    if (!v || (v.mc && v.mc.rs === undefined)) continue; // covered by a merged cell
    writeCell(ws.getCell(r + 1, c + 1), v);
  }
  const cfg = sheet.config ?? {};
  for (const m of Object.values(cfg.merge ?? {})) if (m.rs > 1 || m.cs > 1) ws.mergeCells(m.r + 1, m.c + 1, m.r + m.rs, m.c + m.cs);
  for (const [c, px] of Object.entries(cfg.columnlen ?? {})) ws.getColumn(Number(c) + 1).width = Math.max(1, Math.round(((px - 5) / 7) * 100) / 100);
  for (const [r, px] of Object.entries(cfg.rowlen ?? {})) ws.getRow(Number(r) + 1).height = Math.round(px * 0.75 * 100) / 100;
  const fz = sheet.frozen as { type?: string; range?: { row_focus: number; column_focus: number } } | undefined;
  if (fz?.type) {
    const rows = fz.type === 'row' || fz.type === 'both' ? 1 : fz.type.startsWith('range') && fz.type !== 'rangeColumn' ? (fz.range?.row_focus ?? 0) + 1 : 0;
    const cols = fz.type === 'column' || fz.type === 'both' ? 1 : fz.type === 'rangeColumn' || fz.type === 'rangeBoth' ? (fz.range?.column_focus ?? 0) + 1 : 0;
    if (rows || cols) ws.views = [{ state: 'frozen', xSplit: cols, ySplit: rows }];
  }
}

/** Workbook → XLSX file. */
export async function toXlsx(sheets: readonly StoredSheet[], Excel?: ExcelModule): Promise<ArrayBuffer> {
  const X = Excel ?? (await loadExcel());
  const wb: Workbook = new X.Workbook();
  wb.creator = 'ProjectList';
  const used = new Set<string>();
  for (const sheet of [...sheets].sort((a, b) => (a.order ?? 0) - (b.order ?? 0))) {
    const ws = wb.addWorksheet(sheetName(sheet.name, used), sheet.hide === 1 ? { state: 'hidden' } : undefined);
    writeSheet(ws, sheet);
  }
  return (await wb.xlsx.writeBuffer()) as ArrayBuffer;
}

// ---- import ------------------------------------------------------------------------

export const MAX_IMPORT_CELLS = 200_000;
export class ImportTooLarge extends Error {}

function readValue(cell: Cell, out: SheetCell) {
  const val = cell.value;
  const fmt = cell.numFmt && cell.numFmt !== 'General' ? cell.numFmt : '';
  const setNumber = (n: number) => {
    out.v = n;
    out.m = String(n);
    out.ct = { fa: fmt || 'General', t: 'n' };
  };
  const setDate = (d: Date) => {
    out.v = dateToSerial(d);
    out.m = d.toISOString().slice(0, 10);
    out.ct = { fa: fmt || 'yyyy-mm-dd', t: 'd' };
  };
  const setText = (s: string) => {
    out.v = s;
    out.m = s;
    out.ct = { fa: 'General', t: 'g' };
  };
  if (val === null || val === undefined) return;
  if (typeof val === 'number') return setNumber(val);
  if (typeof val === 'string') return setText(val);
  if (typeof val === 'boolean') {
    out.v = val;
    out.m = val ? 'TRUE' : 'FALSE';
    out.ct = { fa: 'General', t: 'b' };
    return;
  }
  if (val instanceof Date) return setDate(val);
  if (typeof val === 'object') {
    if ('formula' in val || 'sharedFormula' in val) {
      const formula = cell.formula;
      const result = (val as { result?: unknown }).result;
      if (result instanceof Date) setDate(result);
      else if (typeof result === 'number') setNumber(result);
      else if (typeof result === 'boolean') (out.v = result), (out.m = result ? 'TRUE' : 'FALSE');
      else if (typeof result === 'string') setText(result);
      else if (result && typeof result === 'object' && 'error' in result) (out.v = String(result.error)), (out.m = String(result.error));
      if (formula) out.f = '=' + formula;
      return;
    }
    if ('richText' in val) return setText(val.richText.map((t) => t.text).join(''));
    if ('hyperlink' in val) {
      const text = (val as { text?: unknown }).text;
      return setText(typeof text === 'string' ? text : text && typeof text === 'object' && 'richText' in text ? (text as { richText: { text: string }[] }).richText.map((t) => t.text).join('') : String(val.hyperlink));
    }
    if ('error' in val) return setText(String(val.error));
  }
}

function readStyle(cell: Cell, out: SheetCell) {
  const f = cell.font;
  if (f?.bold) out.bl = 1;
  if (f?.italic) out.it = 1;
  if (f?.underline) out.un = 1;
  if (f?.strike) out.cl = 1;
  const fc = fromArgb(f?.color?.argb);
  if (fc && fc !== '#000000') out.fc = fc;
  if (f?.size && f.size !== 11) out.fs = f.size;
  const fill = cell.fill;
  if (fill?.type === 'pattern' && fill.pattern === 'solid') {
    const bg = fromArgb(fill.fgColor?.argb);
    if (bg) out.bg = bg;
  }
  const a = cell.alignment;
  if (a?.horizontal === 'center' || a?.horizontal === 'centerContinuous') out.ht = 0;
  else if (a?.horizontal === 'left') out.ht = 1;
  else if (a?.horizontal === 'right') out.ht = 2;
  if (a?.vertical === 'middle') out.vt = 0;
  else if (a?.vertical === 'top') out.vt = 1;
  else if (a?.vertical === 'bottom') out.vt = 2;
  if (a?.wrapText) out.tb = '2';
}

/** "B2:D3" → {r, c, rs, cs} (0-based). */
export function parseRange(range: string): { r: number; c: number; rs: number; cs: number } | null {
  const m = /^\$?([A-Z]{1,3})\$?(\d+):\$?([A-Z]{1,3})\$?(\d+)$/i.exec(range);
  if (!m) return null;
  const col = (s: string) => [...s.toUpperCase()].reduce((n, ch) => n * 26 + ch.charCodeAt(0) - 64, 0) - 1;
  const [r1, c1, r2, c2] = [Number(m[2]) - 1, col(m[1]), Number(m[4]) - 1, col(m[3])];
  return { r: Math.min(r1, r2), c: Math.min(c1, c2), rs: Math.abs(r2 - r1) + 1, cs: Math.abs(c2 - c1) + 1 };
}

function readSheet(ws: Worksheet, order: number): StoredSheet {
  const cells = new Map<string, StoredCell>();
  let maxR = 0;
  let maxC = 0;
  const rowlen: Record<string, number> = {};
  ws.eachRow({ includeEmpty: false }, (row, rn) => {
    if (row.height) rowlen[rn - 1] = Math.round(row.height / 0.75);
    row.eachCell({ includeEmpty: false }, (cell, cn) => {
      const out: SheetCell = {};
      readValue(cell, out);
      readStyle(cell, out);
      if (!Object.keys(out).length) return;
      const r = rn - 1;
      const c = cn - 1;
      if (cells.size >= MAX_IMPORT_CELLS) throw new ImportTooLarge();
      cells.set(`${r}_${c}`, { r, c, v: out });
      maxR = Math.max(maxR, r);
      maxC = Math.max(maxC, c);
    });
  });
  const config: SheetConfig = {};
  // merged cells: the first cell carries the size, the covered ones point to it
  const merges = ((ws.model as { merges?: string[] }).merges ?? []).map(parseRange).filter((m) => !!m);
  for (const m of merges) {
    (config.merge ??= {})[`${m.r}_${m.c}`] = m;
    for (let r = m.r; r < m.r + m.rs; r++)
      for (let c = m.c; c < m.c + m.cs; c++) {
        const key = `${r}_${c}`;
        const entry = cells.get(key) ?? { r, c, v: {} };
        entry.v = { ...entry.v, mc: r === m.r && c === m.c ? { ...m } : { r: m.r, c: m.c } };
        cells.set(key, entry);
      }
    maxR = Math.max(maxR, m.r + m.rs - 1);
    maxC = Math.max(maxC, m.c + m.cs - 1);
  }
  const columnlen: Record<string, number> = {};
  (ws.columns ?? []).forEach((col, i) => {
    if (col?.width) columnlen[i] = Math.round(col.width * 7 + 5);
  });
  if (Object.keys(columnlen).length) config.columnlen = columnlen;
  if (Object.keys(rowlen).length) config.rowlen = rowlen;
  const sheet: StoredSheet = {
    name: ws.name,
    order,
    row: Math.max(84, maxR + 21),
    column: Math.max(60, maxC + 6),
    celldata: [...cells.values()],
    config
  };
  if (ws.state === 'hidden' || ws.state === 'veryHidden') sheet.hide = 1;
  const view = ws.views?.[0] as { state?: string; xSplit?: number; ySplit?: number } | undefined;
  if (view?.state === 'frozen' && (view.xSplit || view.ySplit)) {
    const x = view.xSplit ?? 0;
    const y = view.ySplit ?? 0;
    sheet.frozen = { type: x && y ? 'rangeBoth' : y ? 'rangeRow' : 'rangeColumn', range: { row_focus: Math.max(0, y - 1), column_focus: Math.max(0, x - 1) } };
  }
  return sheet;
}

/** XLSX file → workbook (throws ImportTooLarge beyond MAX_IMPORT_CELLS cells). */
export async function fromXlsx(buf: ArrayBuffer, Excel?: ExcelModule): Promise<StoredSheet[]> {
  const X = Excel ?? (await loadExcel());
  const wb: Workbook = new X.Workbook();
  await wb.xlsx.load(buf);
  const sheets = wb.worksheets.map((ws, i) => readSheet(ws, i));
  if (sheets.reduce((n, s) => n + (s.celldata?.length ?? 0), 0) > MAX_IMPORT_CELLS) throw new ImportTooLarge();
  return sheets.slice(0, 50);
}
