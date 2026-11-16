// Archive (archive page and the archive tab of a customer): filtering, sorting
// and grouping of archived projects (pure functions, testable).
import type { ArchivedProject } from './types';

export type ArchivePeriod = 'all' | '30d' | '6m' | '12m' | 'older';
export const ARCHIVE_PERIODS: readonly ArchivePeriod[] = ['all', '30d', '6m', '12m', 'older'];
export type ArchiveSort = 'archived' | 'name' | 'customer' | 'progress';
export const ARCHIVE_SORTS: readonly ArchiveSort[] = ['archived', 'name', 'customer', 'progress'];

/** Local calendar day (YYYY-MM-DD) of a timestamp. */
export function localDay(iso: string): string {
  const d = new Date(iso);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

function monthsAgo(now: Date, months: number): Date {
  const d = new Date(now);
  d.setMonth(d.getMonth() - months);
  return d;
}

/** Was the project archived within the period? Unknown dates count as "older". */
export function inPeriod(p: ArchivedProject, period: ArchivePeriod, now: Date): boolean {
  if (period === 'all') return true;
  const at = p.archived_at ? new Date(p.archived_at) : null;
  const yearAgo = monthsAgo(now, 12);
  if (period === 'older') return !at || at < yearAgo;
  if (!at) return false;
  const from = period === '30d' ? new Date(now.getTime() - 30 * 86400_000) : period === '6m' ? monthsAgo(now, 6) : yearAgo;
  return at >= from;
}

/** Not all items checked off (projects without items are not "incomplete"). */
export const incomplete = (p: ArchivedProject) => p.done < p.total;

export interface ArchiveFilter {
  q: string;
  customers: number[] | null; // null = all
  period: ArchivePeriod;
  incompleteOnly: boolean;
}

export function filterArchive(rows: readonly ArchivedProject[], f: ArchiveFilter, now: Date): ArchivedProject[] {
  const q = f.q.trim().toLowerCase();
  const cust = f.customers ? new Set(f.customers) : null;
  return rows.filter(
    (p) =>
      (!q || p.name.toLowerCase().includes(q) || p.customer_name.toLowerCase().includes(q)) &&
      (!cust || cust.has(p.customer_id)) &&
      inPeriod(p, f.period, now) &&
      (!f.incompleteOnly || incomplete(p))
  );
}

const byName = (a: ArchivedProject, b: ArchivedProject) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
const ratio = (p: ArchivedProject) => (p.total ? p.done / p.total : 1);

/** archived: newest first, unknown dates last; progress: least complete first. */
export function sortArchive(rows: readonly ArchivedProject[], sort: ArchiveSort): ArchivedProject[] {
  const out = [...rows];
  if (sort === 'name') return out.sort(byName);
  if (sort === 'customer') return out.sort((a, b) => a.customer_name.localeCompare(b.customer_name, undefined, { sensitivity: 'base' }) || byName(a, b));
  if (sort === 'progress') return out.sort((a, b) => ratio(a) - ratio(b) || byName(a, b));
  return out.sort((a, b) => {
    if (a.archived_at === b.archived_at) return byName(a, b);
    if (!a.archived_at) return 1;
    if (!b.archived_at) return -1;
    return b.archived_at.localeCompare(a.archived_at);
  });
}

export interface YearGroup {
  year: string | null; // null = date unknown
  rows: ArchivedProject[];
}

/** Groups by the year of archiving, newest year first, unknown last; keeps the order within a year. */
export function groupByYear(rows: readonly ArchivedProject[]): YearGroup[] {
  const map = new Map<string | null, ArchivedProject[]>();
  for (const p of rows) {
    const y = p.archived_at ? localDay(p.archived_at).slice(0, 4) : null;
    if (!map.has(y)) map.set(y, []);
    map.get(y)!.push(p);
  }
  return [...map.entries()]
    .sort(([a], [b]) => (a === b ? 0 : a === null ? 1 : b === null ? -1 : b.localeCompare(a)))
    .map(([year, rows]) => ({ year, rows }));
}
