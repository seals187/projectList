import { describe, expect, it } from 'vitest';
import { categoryHex, groupTemplates, lastUsed, validVariableName, variablesIn } from './templateLibrary';
import type { TemplateCategory, TemplateSummary } from './types';

const tpl = (id: number, category_id: number | null): TemplateSummary => ({ id, name: `T${id}`, description: '', item_count: 0, category_id });
const cat = (id: number, color = ''): TemplateCategory => ({ id, name: `C${id}`, color, position: id });

describe('groupTemplates', () => {
  it('groups in category order, uncategorized last, empty groups dropped', () => {
    const g = groupTemplates([tpl(1, 2), tpl(2, null), tpl(3, 1), tpl(4, 99)], [cat(1), cat(2), cat(3)]);
    expect(g.map((x) => [x.id, x.templates.map((t) => t.id)])).toEqual([
      [1, [3]],
      [2, [1]],
      [0, [2, 4]]
    ]);
  });
});

describe('lastUsed', () => {
  const now = new Date(2026, 10, 7, 10, 0);
  it('relative days and weeks', () => {
    expect(lastUsed('', now)).toEqual({ kind: 'never' });
    expect(lastUsed(new Date(2026, 10, 7, 8).toISOString(), now)).toEqual({ kind: 'today' });
    expect(lastUsed(new Date(2026, 10, 6, 23).toISOString(), now)).toEqual({ kind: 'yesterday' });
    expect(lastUsed(new Date(2026, 10, 4).toISOString(), now)).toEqual({ kind: 'days', n: 3 });
    expect(lastUsed(new Date(2026, 9, 30).toISOString(), now)).toEqual({ kind: 'weeks', n: 1 });
    expect(lastUsed(new Date(2026, 8, 2).toISOString(), now)).toEqual({ kind: 'date', date: '02.09.2026' });
  });
});

describe('variables', () => {
  it('finds and validates names', () => {
    expect(variablesIn('{{a.b}} x {{ heute }} {{a.b}} {{-x}}')).toEqual(['a.b', 'heute']);
    expect(validVariableName('server.host_name')).toBe(true);
    expect(validVariableName('Server')).toBe(false);
    expect(validVariableName('a..b')).toBe(false);
  });
  it('category color with gray fallback', () => {
    expect(categoryHex(cat(1, 'red'))).toBe('#ef4444');
    expect(categoryHex(null)).toBe('#71717a');
  });
});
