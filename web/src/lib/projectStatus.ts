// Project status (phases on the board) and customer codes/colors for the overview.
// All classes written out so Tailwind finds them.
import type { ProjectStatus } from './types';
import { t } from './i18n.svelte';
import { sidebar } from './workspace.svelte';
import { BOARD_STATUSES } from './projectQuery';

/** Order of the columns on the board (without "archived"), defined in projectQuery. */
export { BOARD_STATUSES };
export const ALL_STATUSES: readonly ProjectStatus[] = [...BOARD_STATUSES, 'archived'];

/** Display name of a status: custom name from the settings, otherwise the translation. */
export function statusLabel(s: ProjectStatus): string {
  return sidebar.statusLabels[s] || t(`status.${s}`);
}

/** Dot (sidebar, board columns). */
export const STATUS_DOT: Record<ProjectStatus, string> = {
  planning: 'bg-zinc-400 dark:bg-zinc-500',
  active: 'bg-emerald-500',
  review: 'bg-orange-500',
  blocked: 'bg-red-600 dark:bg-red-500',
  done: 'bg-emerald-800 dark:bg-emerald-600',
  archived: 'bg-zinc-300 dark:bg-zinc-600'
};

/** Default base color of a status (same hue as its board dot), used for the badge tint. */
const STATUS_BASE: Record<ProjectStatus, string> = {
  planning: 'var(--color-zinc-500)',
  active: 'var(--color-emerald-500)',
  review: 'var(--color-orange-500)',
  blocked: 'var(--color-red-600)',
  done: 'var(--color-emerald-800)',
  archived: 'var(--color-zinc-400)'
};

/** Custom color of a status (#rrggbb, set on the board by admins), '' = default. */
export const statusColor = (s: ProjectStatus): string => sidebar.statusColors[s] ?? '';

// Dot: default classes, or an inline style for a custom color.
export const statusDotCls = (s: ProjectStatus) => (statusColor(s) ? '' : STATUS_DOT[s]);
export const statusDotStyle = (s: ProjectStatus) => (statusColor(s) ? `background-color:${statusColor(s)}` : undefined);
/** Background in the status color (lightened a little so black text stays readable), text always black. */
export function statusBadgeStyle(s: ProjectStatus) {
  const c = statusColor(s) || STATUS_BASE[s];
  return `background-color:color-mix(in srgb, ${c} 55%, white);color:#000`;
}

/** Progress bar: done = green, otherwise accent. */
export const barCls = (done: number, total: number) => (total > 0 && done === total ? 'bg-emerald-500' : 'bg-accent-500');

/** Display name without code prefix: "(KA) Kunde-A" → "Kunde-A". */
export function customerLabel(name: string): string {
  return name.replace(/^\s*\([^)]{1,4}\)\s*/, '') || name;
}
