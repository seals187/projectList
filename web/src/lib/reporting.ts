// Reporting: types of GET /api/reports (internal/report; units: checklists as a
// whole and projects, no single items) and pure helpers
// (query string, changes vs. the previous period, CSV). Tested.

export interface ReportFilter {
  period: number; // days, one of the periods of the settings
  customers: number[];
  assignees: number[];
  tags: number[];
}
/** All periods an admin can offer (Settings → Reporting). */
export const ALL_PERIODS = [7, 14, 30, 60, 90, 180, 365] as const;
export type ReportView = 'dashboard' | 'customers' | 'workload' | 'templates';
export type ReportKpi = 'completed' | 'open' | 'overdue' | 'projects_done';

/** Reporting settings (GET/PUT /api/reports/config, internal/report/config.go). */
export interface ReportConfig {
  views: ReportView[];
  default_view: ReportView;
  periods: number[];
  default_period: number;
  progress_basis: 'checklists' | 'items';
  due_basis: 'checklist' | 'item';
  due_soon_days: number;
  include_done: boolean;
  respect_customer_progress: boolean;
  overload_threshold: number; // 0 = off
  kpis: ReportKpi[];
}
export interface ReportConfigResp {
  config: ReportConfig;
  defaults: ReportConfig;
  choices: { views: ReportView[]; periods: number[]; kpis: ReportKpi[] };
}

/** Chosen period if the settings offer it, otherwise their default. */
export function pickPeriod(chosen: string | number, cfg: Pick<ReportConfig, 'periods' | 'default_period'>): number {
  const n = Number(chosen);
  return cfg.periods.includes(n) ? n : cfg.default_period;
}

/** Chosen view if enabled, otherwise the default view. */
export function pickView(chosen: string, cfg: Pick<ReportConfig, 'views' | 'default_view'>): ReportView {
  return cfg.views.includes(chosen as ReportView) ? (chosen as ReportView) : cfg.default_view;
}

/** Moves an entry of a list one place up (-1) or down (+1). */
export function moveItem<T>(list: readonly T[], i: number, dir: -1 | 1): T[] {
  const j = i + dir;
  if (i < 0 || j < 0 || j >= list.length) return [...list];
  const out = [...list];
  [out[i], out[j]] = [out[j], out[i]];
  return out;
}

export interface Week {
  label: string;
  start: string;
  completed: number;
  created: number;
}
export interface DueEntry {
  date: string;
  kind: 'project' | 'checklist';
  id: number;
  title: string;
  customer: string;
}
export interface CustProj {
  id: number;
  name: string;
  status: string;
  done: number;
  total: number;
}
export interface ReportCustomer {
  id: number;
  name: string;
  projects: CustProj[];
  done: number;
  total: number;
  open: number;
  overdue: number;
  next_due: string;
}
export interface ReportPerson {
  id: number;
  name: string;
  open: number;
  due_week: number;
  overdue: number;
  done: number;
  overloaded: boolean; // open checklists reach the overload threshold
}
export interface TplRow {
  id: number;
  name: string;
  category_id: number | null;
  used: number;
  avg_days: number | null;
  planned: number | null;
  on_time: number | null;
  completed: number;
}
export interface Report {
  from: string;
  to: string;
  days: number;
  dashboard: {
    completed: number;
    completed_prev: number;
    open: number;
    open_prev: number;
    overdue: number;
    overdue_prev: number;
    projects_done: number;
    weeks: Week[];
    status: Record<string, number>;
    due: DueEntry[] | null;
  };
  customers: ReportCustomer[];
  workload: {
    people: ReportPerson[];
    unassigned: { total: number; checklists: ClBrief[] };
    done_prev: number;
  };
  templates: {
    rows: TplRow[];
    created: number;
    avg_days: number | null;
    avg_days_prev: number | null;
    on_time: number | null;
    on_time_prev: number | null;
    longest: ClBrief[]; // completed checklists that took longest (days)
    auto_created: number;
    auto_warned: number;
  };
  assignees: { id: number; name: string }[];
  config: ReportConfig;
}
/** A checklist in lists (date: due or completion date; done/total: its progress). */
export interface ClBrief {
  id: number;
  title: string;
  project: string;
  customer: string;
  template?: string;
  date: string;
  done: number;
  total: number;
  days?: number;
}
export interface CustomerReport {
  customer: { id: number; name: string; location: string; contact: string; notes: string };
  from: string;
  to: string;
  projects: CustProj[];
  done: number;
  total: number;
  completed: ClBrief[];
  overdue: ClBrief[];
  upcoming: ClBrief[];
  open: ClBrief[];
  due_soon_days: number;
  progress_basis: 'checklists' | 'items';
}
export interface ReportSchedule {
  id: number;
  recipient_id: number;
  recipient_name: string;
  freq: 'weekly' | 'monthly';
  weekday: number;
  day: number;
  filters: ReportFilter;
  lang: string;
  last_sent_on: string;
}

/** Query string for /api/reports (empty lists are left out). */
export function reportQuery(f: ReportFilter): string {
  const p = new URLSearchParams({ period: String(f.period) });
  if (f.customers.length) p.set('customers', f.customers.join(','));
  if (f.assignees.length) p.set('assignees', f.assignees.join(','));
  if (f.tags.length) p.set('tags', f.tags.join(','));
  return p.toString();
}

/** Change in percent vs. the previous value (null without a base). */
export function pctChange(cur: number, prev: number): number | null {
  if (!prev) return null;
  return Math.round(((cur - prev) / prev) * 100);
}

/** "+18 %" / "−9 %" / "+1" (typographic minus). */
export function signed(n: number, unit = ''): string {
  const s = n > 0 ? '+' : n < 0 ? '−' : '±';
  return `${s}${Math.abs(n)}${unit}`;
}

export const percent = (done: number, total: number) => (total > 0 ? Math.round((done / total) * 100) : 0);

/** "13.11." → day and month of an ISO date. */
export function dayMonth(iso: string): string {
  if (!iso || iso.length < 10) return '–';
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.`;
}

/** "01.10.2026" */
export function swissDate(iso: string): string {
  if (!iso || iso.length < 10) return '';
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`;
}

/** CSV with ";" (Excel in German-speaking regions), quoted where needed, BOM for UTF-8. */
export function toCsv(rows: (string | number | null | undefined)[][]): string {
  const cell = (v: string | number | null | undefined) => {
    const s = v === null || v === undefined ? '' : String(v);
    return /[";\n\r]/.test(s) ? `"${s.replaceAll('"', '""')}"` : s;
  };
  return '﻿' + rows.map((r) => r.map(cell).join(';')).join('\r\n');
}

/** Downloads text (or binary content, e.g. an XLSX file) as a file (browser). */
export function download(name: string, content: BlobPart, type = 'text/csv;charset=utf-8') {
  const url = URL.createObjectURL(new Blob([content], { type }));
  const a = document.createElement('a');
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

/** Points of a line chart: values → SVG coordinates in a w × h box (max at the top). */
export function linePoints(values: number[], w: number, h: number, max: number): [number, number][] {
  const n = values.length;
  return values.map((v, i) => [n > 1 ? (i / (n - 1)) * w : w / 2, max > 0 ? h - (v / max) * h : h]);
}

/** Nice upper bound of an axis (1, 2, 5 × 10ⁿ). */
export function niceMax(v: number): number {
  if (v <= 0) return 4;
  const p = 10 ** Math.floor(Math.log10(v));
  for (const m of [1, 2, 5, 10]) if (v <= m * p) return m * p;
  return 10 * p;
}
