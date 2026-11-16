import { describe, expect, it } from 'vitest';
import { filterArchive, groupByYear, inPeriod, sortArchive } from './archive';
import type { ArchivedProject } from './types';

const p = (id: number, name: string, archived_at: string | null, extra: Partial<ArchivedProject> = {}): ArchivedProject => ({
  id,
  customer_id: 1,
  name,
  customer_name: 'Kunde',
  archived_at,
  archived_by: null,
  archived_by_name: '',
  total: 4,
  done: 4,
  ...extra
});

const now = new Date('2026-11-16T12:00:00Z');
const rows = [
  p(1, 'Server-Migration', '2026-11-12T09:00:00Z'),
  p(2, 'Firewall', '2026-06-01T09:00:00Z', { done: 2, customer_id: 2, customer_name: 'Bern' }),
  p(3, 'Altlast', '2024-03-02T09:00:00Z'),
  p(4, 'Unbekannt', null, { total: 0, done: 0 })
];

describe('archive', () => {
  it('filters by period; unknown dates count as older', () => {
    expect(rows.filter((r) => inPeriod(r, '30d', now)).map((r) => r.id)).toEqual([1]);
    expect(rows.filter((r) => inPeriod(r, '6m', now)).map((r) => r.id)).toEqual([1, 2]);
    expect(rows.filter((r) => inPeriod(r, '12m', now)).map((r) => r.id)).toEqual([1, 2]);
    expect(rows.filter((r) => inPeriod(r, 'older', now)).map((r) => r.id)).toEqual([3, 4]);
    expect(rows.filter((r) => inPeriod(r, 'all', now))).toHaveLength(4);
  });

  it('filters by text, customer and incomplete', () => {
    const f = { q: '', customers: null, period: 'all' as const, incompleteOnly: false };
    expect(filterArchive(rows, { ...f, q: 'bern' }, now).map((r) => r.id)).toEqual([2]);
    expect(filterArchive(rows, { ...f, customers: [2] }, now).map((r) => r.id)).toEqual([2]);
    // projects without items are not incomplete
    expect(filterArchive(rows, { ...f, incompleteOnly: true }, now).map((r) => r.id)).toEqual([2]);
  });

  it('sorts newest archiving first, unknown last', () => {
    expect(sortArchive(rows, 'archived').map((r) => r.id)).toEqual([1, 2, 3, 4]);
    expect(sortArchive(rows, 'name').map((r) => r.id)).toEqual([3, 2, 1, 4]);
    expect(sortArchive(rows, 'progress')[0].id).toBe(2);
  });

  it('groups by year, unknown last', () => {
    expect(groupByYear(sortArchive(rows, 'archived')).map((g) => [g.year, g.rows.length])).toEqual([
      ['2026', 2],
      ['2024', 1],
      [null, 1]
    ]);
  });
});
