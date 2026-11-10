import { describe, expect, it } from 'vitest';
import { cellText, fileName, sheetRows, toEditor, toStored, type EditorSheet } from './sheetData';

describe('toStored', () => {
  it('turns the editor matrix into sparse celldata and drops the view state', () => {
    const sheet: EditorSheet = {
      name: 'Hardware',
      id: 'a',
      status: 1,
      config: { merge: {} },
      luckysheet_select_save: [{ row: [0, 0], column: [0, 0] }],
      zoomRatio: 1.2,
      data: [
        [{ v: 'Artikel', m: 'Artikel', bl: 1 }, null],
        [null, { v: 2, m: '2', f: '=1+1' }],
        [{}, { v: undefined }]
      ]
    };
    expect(toStored([sheet])).toEqual([
      {
        name: 'Hardware',
        id: 'a',
        config: { merge: {} },
        celldata: [
          { r: 0, c: 0, v: { v: 'Artikel', m: 'Artikel', bl: 1 } },
          { r: 1, c: 1, v: { v: 2, m: '2', f: '=1+1' } }
        ]
      }
    ]);
  });

  it('keeps celldata of sheets without a matrix and names unnamed sheets', () => {
    const out = toStored([{ name: '', celldata: [{ r: 0, c: 0, v: { v: 1 } }, { r: 0, c: 1, v: null }] }]);
    expect(out).toEqual([{ name: 'Sheet1', celldata: [{ r: 0, c: 0, v: { v: 1 } }] }]);
  });

  it('is stable: storing the stored data again changes nothing', () => {
    const once = toStored([{ name: 'A', data: [[{ v: 1, m: '1' }]] }]);
    expect(toStored(once)).toEqual(once);
  });
});

describe('toEditor', () => {
  it('gives every sheet an id and an order and activates the first one', () => {
    let n = 0;
    const out = toEditor([{ name: 'A' }, { name: 'B', id: 'x', order: 5 }], () => `id${++n}`);
    expect(out).toEqual([
      { name: 'A', id: 'id1', order: 0, status: 1, celldata: [] },
      { name: 'B', id: 'x', order: 5, status: 0, celldata: [] }
    ]);
  });

  it('an empty workbook gets one sheet', () => {
    expect(toEditor([], () => 's').map((s) => s.name)).toEqual(['Sheet1']);
  });
});

describe('texts', () => {
  it('cell text: display text, value, rich text', () => {
    expect(cellText({ v: 16900, m: "16'900" })).toBe("16'900");
    expect(cellText({ v: true })).toBe('true');
    expect(cellText({ ct: { t: 'inlineStr', s: [{ v: 'über ' }, { v: 'Budget' }] } })).toBe('über Budget');
    expect(cellText(null)).toBe('');
  });

  it('rows of a sheet are rectangular without trailing empty cells', () => {
    const rows = sheetRows({
      name: 'S',
      celldata: [
        { r: 0, c: 0, v: { v: 'A' } },
        { r: 2, c: 2, v: { v: 3, m: '3' } },
        { r: 4, c: 5, v: { v: '' } }
      ]
    });
    expect(rows).toEqual([
      ['A', '', ''],
      ['', '', ''],
      ['', '', '3']
    ]);
  });
});

describe('helpers', () => {
  it('file names without forbidden characters', () => {
    expect(fileName('Kalkulation: HW/SW?', 'xlsx')).toBe('Kalkulation HW SW.xlsx');
    expect(fileName('  ', 'csv')).toBe('table.csv');
  });
});
