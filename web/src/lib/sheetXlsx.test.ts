// @vitest-environment node
import { describe, expect, it } from 'vitest';
import type { StoredSheet } from './sheetData';
import { fromArgb, fromXlsx, parseRange, sheetName, toArgb, toXlsx } from './sheetXlsx';

describe('colors and names', () => {
  it('converts colors', () => {
    expect(toArgb('#0e7490')).toBe('FF0E7490');
    expect(toArgb('#abc')).toBe('FFAABBCC');
    expect(toArgb('rgb(255, 0, 16)')).toBe('FFFF0010');
    expect(toArgb('red')).toBeNull();
    expect(fromArgb('FF0E7490')).toBe('#0e7490');
    expect(fromArgb(undefined)).toBeNull();
  });

  it('Excel sheet names are cleaned and unique', () => {
    const used = new Set<string>();
    expect(sheetName('Budget [2026]', used)).toBe('Budget  2026');
    expect(sheetName('budget  2026', used)).toBe('budget  2026 2');
    expect(sheetName('x'.repeat(40), used)).toHaveLength(31);
  });

  it('ranges', () => {
    expect(parseRange('B2:D3')).toEqual({ r: 1, c: 1, rs: 2, cs: 3 });
    expect(parseRange('A1')).toBeNull();
  });
});

describe('XLSX round trip', () => {
  const sheets: StoredSheet[] = [
    {
      name: 'Hardware',
      order: 0,
      celldata: [
        { r: 0, c: 0, v: { v: 'Artikel', m: 'Artikel', bl: 1, bg: '#e0f2f1', ht: 0 } },
        { r: 0, c: 1, v: { v: 'Preis', m: 'Preis', fc: '#dc2626', it: 1 } },
        { r: 1, c: 0, v: { v: 'Server', m: 'Server' } },
        { r: 1, c: 1, v: { v: 8450, m: "8'450", ct: { fa: '#,##0', t: 'n' } } },
        { r: 2, c: 1, v: { v: 3120, m: '3120' } },
        { r: 3, c: 1, v: { v: 11570, m: '11570', f: '=SUM(B2:B3)' } },
        { r: 4, c: 0, v: { v: 'Netto', m: 'Netto', mc: { r: 4, c: 0, rs: 1, cs: 2 } } },
        { r: 4, c: 1, v: { mc: { r: 4, c: 0 } } }
      ],
      config: { merge: { '4_0': { r: 4, c: 0, rs: 1, cs: 2 } }, columnlen: { 0: 152 }, rowlen: { 0: 32 } },
      frozen: { type: 'rangeRow', range: { row_focus: 0, column_focus: 0 } }
    },
    { name: 'Budget', order: 1, celldata: [{ r: 0, c: 0, v: { v: true } }] }
  ];

  it('keeps values, formulas, formats, styles, merges, sizes and frozen rows', async () => {
    const back = await fromXlsx(await toXlsx(sheets));
    expect(back.map((s) => s.name)).toEqual(['Hardware', 'Budget']);
    const cell = (r: number, c: number) => back[0].celldata?.find((x) => x.r === r && x.c === c)?.v;
    expect(cell(0, 0)).toMatchObject({ v: 'Artikel', bl: 1, bg: '#e0f2f1', ht: 0 });
    expect(cell(0, 1)).toMatchObject({ v: 'Preis', fc: '#dc2626', it: 1 });
    expect(cell(1, 1)).toMatchObject({ v: 8450, ct: { fa: '#,##0', t: 'n' } });
    expect(cell(3, 1)).toMatchObject({ f: '=SUM(B2:B3)', v: 11570 });
    expect(cell(4, 0)).toMatchObject({ v: 'Netto', mc: { r: 4, c: 0, rs: 1, cs: 2 } });
    expect(cell(4, 1)).toMatchObject({ mc: { r: 4, c: 0 } });
    expect(back[0].config?.merge).toEqual({ '4_0': { r: 4, c: 0, rs: 1, cs: 2 } });
    expect(back[0].config?.columnlen?.[0]).toBe(152);
    expect(back[0].config?.rowlen?.[0]).toBe(32);
    expect(back[0].frozen).toEqual({ type: 'rangeRow', range: { row_focus: 0, column_focus: 0 } });
    expect(back[1].celldata?.[0].v).toMatchObject({ v: true, m: 'TRUE' });
  });

  it('shows enough rows and columns for the data', async () => {
    const back = await fromXlsx(await toXlsx([{ name: 'Big', celldata: [{ r: 199, c: 69, v: { v: 1 } }] }]));
    expect(back[0].row).toBe(220);
    expect(back[0].column).toBe(75);
  });
});
