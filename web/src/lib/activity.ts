// Project activity log: grouping and texts.
import { fmtDate, locale, t } from './i18n.svelte';
import { statusLabel } from './projectStatus';
import type { ActivityEntry, ProjectStatus } from './types';

/** An activity entry; checked-off items of the same person in the same checklist are combined. */
export interface ActivityGroup extends ActivityEntry {
  count: number; // number of combined "item_done" entries
}

const GROUP_WINDOW = 2 * 60 * 60 * 1000; // items checked off within 2 hours are shown as one entry

/** Combines consecutive "item_done" entries of the same person and checklist (input newest first). */
export function groupActivity(list: ActivityEntry[]): ActivityGroup[] {
  const out: ActivityGroup[] = [];
  for (const a of list) {
    const last = out[out.length - 1];
    if (
      last &&
      a.kind === 'item_done' &&
      last.kind === 'item_done' &&
      last.user_id === a.user_id &&
      last.subject === a.subject &&
      Date.parse(last.created_at) - Date.parse(a.created_at) < GROUP_WINDOW
    ) {
      last.count++;
      continue;
    }
    out.push({ ...a, count: 1 });
  }
  return out;
}

const range = (start: string, due: string) =>
  start && due ? `${fmtDate(start)} – ${fmtDate(due)}` : due ? fmtDate(due) : start ? `${fmtDate(start)} –` : t('activity.noDates');

/** Text after the person's name, e.g. "checked off 3 items in Configuration". */
export function activityText(a: ActivityGroup): string {
  const s = { subject: a.subject, detail: a.detail };
  switch (a.kind) {
    case 'project_created':
      return a.detail === 'copy' ? t('activity.copied', s) : t('activity.created');
    case 'renamed':
      return t('activity.renamed', s);
    case 'status':
      return t('activity.status', { status: statusLabel(a.detail as ProjectStatus) });
    case 'dates':
      return t('activity.dates', { range: range(a.subject, a.detail) });
    case 'owner':
      return a.detail ? t('activity.owner', s) : t('activity.ownerRemoved');
    case 'tags':
      return a.detail ? t('activity.tags', s) : t('activity.tagsRemoved');
    case 'redmine':
      return a.detail ? t('activity.redmine', s) : t('activity.redmineRemoved');
    case 'checklist_created':
      return t('activity.checklistCreated', s);
    case 'checklist_completed':
      return t('activity.checklistCompleted', s);
    case 'checklist_deleted':
      return t('activity.checklistDeleted', s);
    case 'item_done':
      return a.count > 1 ? t('activity.itemsDone', { n: a.count, subject: a.subject }) : t('activity.itemDone', s);
    case 'file_added':
      return t('activity.fileAdded', s);
    case 'file_deleted':
      return t('activity.fileDeleted', s);
    default:
      return a.kind;
  }
}

/** "2 hours ago", "yesterday", otherwise the date. */
export function relativeTime(iso: string): string {
  const then = new Date(iso);
  const diff = (Date.now() - then.getTime()) / 1000;
  const rtf = new Intl.RelativeTimeFormat(locale(), { numeric: 'auto', style: 'short' });
  if (diff < 60) return rtf.format(0, 'second');
  if (diff < 3600) return rtf.format(-Math.floor(diff / 60), 'minute');
  const days = Math.round((startOfDay(new Date()) - startOfDay(then)) / 86_400_000);
  if (days === 0) return rtf.format(-Math.floor(diff / 3600), 'hour');
  if (days < 7) return rtf.format(-days, 'day');
  return then.toLocaleDateString(locale(), { day: '2-digit', month: '2-digit', year: 'numeric' });
}
const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
