// Projektstatus (Phasen auf dem Board) und Kunden-Kürzel/-Farben für die Übersicht.
// Alle Klassen ausgeschrieben, damit Tailwind sie findet.
import type { ProjectStatus } from './types';

/** Reihenfolge der Spalten auf dem Board (ohne „archiviert“). */
export const BOARD_STATUSES = ['planning', 'active', 'review', 'blocked', 'done'] as const satisfies readonly ProjectStatus[];
export const ALL_STATUSES: readonly ProjectStatus[] = [...BOARD_STATUSES, 'archived'];

/** Punkt (Sidebar, Board-Spalten). */
export const STATUS_DOT: Record<ProjectStatus, string> = {
  planning: 'bg-zinc-400 dark:bg-zinc-500',
  active: 'bg-accent-500',
  review: 'bg-amber-600 dark:bg-amber-500',
  blocked: 'bg-red-600 dark:bg-red-500',
  done: 'bg-emerald-600 dark:bg-emerald-500',
  archived: 'bg-zinc-300 dark:bg-zinc-600'
};

/** Badge mit Text. */
export const STATUS_BADGE: Record<ProjectStatus, string> = {
  planning: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300',
  active: 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400',
  review: 'bg-amber-50 text-amber-700 dark:bg-amber-950/50 dark:text-amber-400',
  blocked: 'bg-red-50 text-red-700 dark:bg-red-950/50 dark:text-red-400',
  done: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400',
  archived: 'bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400'
};

/** Fortschrittsbalken: fertig = grün, sonst Akzent. */
export const barCls = (done: number, total: number) => (total > 0 && done === total ? 'bg-emerald-500' : 'bg-accent-500');

/** Kürzel eines Kunden: "(KA) Kunde-A" → "KA", sonst Initialen. */
export function customerCode(name: string): string {
  const m = name.match(/^\s*\(([^)]{1,4})\)/);
  if (m) return m[1].toUpperCase();
  const parts = name.trim().split(/\s+/);
  return ((parts[0]?.[0] ?? '') + (parts[1]?.[0] ?? '')).toUpperCase() || '?';
}

const CUSTOMER_TONES = [
  'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400',
  'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400',
  'bg-amber-50 text-amber-700 dark:bg-amber-950/50 dark:text-amber-400',
  'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-400',
  'bg-rose-50 text-rose-700 dark:bg-rose-950/50 dark:text-rose-400',
  'bg-violet-50 text-violet-700 dark:bg-violet-950/50 dark:text-violet-400'
];
/** Farbe des Kunden-Kürzels (stabil je Kunden-ID). */
export const customerTone = (id: number) => CUSTOMER_TONES[(id - 1 + CUSTOMER_TONES.length * 100) % CUSTOMER_TONES.length];

/** Tage von heute bis zum Datum (YYYY-MM-DD); negativ = überfällig. */
export function daysUntil(iso: string, today: string): number {
  return Math.round((Date.parse(iso + 'T00:00:00') - Date.parse(today + 'T00:00:00')) / 86_400_000);
}

/** Anzeigename ohne Kürzel-Präfix: "(KA) Kunde-A" → "Kunde-A". */
export function customerLabel(name: string): string {
  return name.replace(/^\s*\([^)]{1,4}\)\s*/, '') || name;
}

/**
 * Gesamtfortschritt eines Kunden: erledigte / alle Einträge, nur über Projekte,
 * die in den Kundenfortschritt einfliessen (include_in_customer_progress).
 * Archivierte Projekte zählen nie. `excluded` = Anzahl bewusst ausgenommener Projekte.
 */
export function customerProgress(projects: { status: ProjectStatus; done: number; total: number; include_in_customer_progress: boolean }[]) {
  let done = 0,
    total = 0,
    excluded = 0;
  for (const p of projects) {
    if (p.status === 'archived') continue;
    if (!p.include_in_customer_progress) {
      excluded++;
      continue;
    }
    done += p.done;
    total += p.total;
  }
  return { done, total, excluded };
}
