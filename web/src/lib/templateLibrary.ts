// Template library (templates page): grouping by category, "last used" as a
// relative day, colors of categories. Pure functions (tested).
import type { TemplateCategory, TemplateSummary } from './types';
import { cardHex } from './cardColors';

export const GRAY = '#71717a';

/** Hex color of a category ("" → gray). */
export const categoryHex = (c: Pick<TemplateCategory, 'color'> | undefined | null) => (c && cardHex(c.color)) || GRAY;

export interface TemplateGroup {
  id: number; // category id, 0 = without category
  category: TemplateCategory | null;
  templates: TemplateSummary[];
}

/** Templates grouped by category (category order; "without category" last; empty groups dropped). */
export function groupTemplates(list: TemplateSummary[], categories: TemplateCategory[]): TemplateGroup[] {
  const known = new Set(categories.map((c) => c.id));
  const groups: TemplateGroup[] = categories.map((c) => ({ id: c.id, category: c, templates: list.filter((t) => t.category_id === c.id) }));
  groups.push({ id: 0, category: null, templates: list.filter((t) => !t.category_id || !known.has(t.category_id)) });
  return groups.filter((g) => g.templates.length > 0);
}

/** "Last used" relative to today: today / yesterday / n days / n weeks ago, else the date. */
export type LastUsed = { kind: 'never' } | { kind: 'today' } | { kind: 'yesterday' } | { kind: 'days'; n: number } | { kind: 'weeks'; n: number } | { kind: 'date'; date: string };

export function lastUsed(iso: string | undefined, now: Date): LastUsed {
  if (!iso) return { kind: 'never' };
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return { kind: 'never' };
  const day = (x: Date) => Date.UTC(x.getFullYear(), x.getMonth(), x.getDate());
  const days = Math.round((day(now) - day(d)) / 86_400_000);
  if (days <= 0) return { kind: 'today' };
  if (days === 1) return { kind: 'yesterday' };
  if (days < 7) return { kind: 'days', n: days };
  if (days < 28) return { kind: 'weeks', n: Math.floor(days / 7) };
  const p = (n: number) => String(n).padStart(2, '0');
  return { kind: 'date', date: `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()}` };
}

/** Variables written in a text ({{name}}), unique. */
export function variablesIn(text: string): string[] {
  return [...new Set([...text.matchAll(/\{\{\s*([a-zA-Z][a-zA-Z0-9_]*(?:\.[a-zA-Z0-9_]+)*)\s*\}\}/g)].map((m) => m[1]))];
}

/** Valid name of a custom variable (lower case, digits, "_", dots). */
export const validVariableName = (s: string) => s.length <= 60 && /^[a-z][a-z0-9_]*(\.[a-z0-9_]+)*$/.test(s);
