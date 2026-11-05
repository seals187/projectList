// Overview data model: sidebar tree (customers → projects → checklists)
// extended with team and next step from /api/overview.
import type { OverviewExtra, SidebarChecklist, SidebarCustomer, SidebarProject } from './types';
import { customerProgress, daysUntil } from './projectStatus';
import { htmlToText } from './rich';

export interface OvProject extends SidebarProject {
  customer: SidebarCustomer;
  team: { id: number; name: string }[];
  next: string; // plain text
  updated: string; // latest activity (ISO), otherwise the creation date
  clExtra: Record<number, { assignee_id: number | null; assignee_name: string; overdue: number }>;
}

export interface OvCustomer {
  customer: SidebarCustomer;
  projects: OvProject[];
  done: number; // customer progress (included projects only)
  total: number;
  excluded: number; // projects not counted in the customer progress
}

export function buildOverview(customers: SidebarCustomer[], extra: Record<string, OverviewExtra>): OvCustomer[] {
  return customers.map((c) => {
    const projects = c.projects
      .filter((p) => p.status !== 'archived')
      .map((p) => {
        const x = extra[p.id];
        const clExtra = Object.fromEntries(Object.entries(x?.checklists ?? {}).map(([k, v]) => [Number(k), v]));
        return { ...p, customer: c, team: x?.team ?? [], next: x?.next_step ? htmlToText(x.next_step) : '', updated: x?.updated_at ?? p.created_at, clExtra };
      });
    return { customer: c, projects, ...customerProgress(projects) };
  });
}

/** Highlight due date: not done and due in ≤ 3 days or overdue. */
export function dueUrgent(p: Pick<SidebarProject, 'due_date' | 'status'>, today: string): boolean {
  return !!p.due_date && p.status !== 'done' && daysUntil(p.due_date, today) <= 3;
}

export const pct = (done: number, total: number) => (total ? Math.round((done / total) * 100) : 0);

/** Segment width of a checklist in the split bar (at least slightly visible). */
export const segGrow = (cl: SidebarChecklist) => Math.max(cl.total, 1);

/**
 * Time vs. progress of a project: share of the time elapsed between start
 * (start date or creation date) and due date, compared with the share of items
 * done. Without a due date there is nothing to compare.
 */
export interface Schedule {
  start: string;
  due: string;
  elapsed: number; // % of the time elapsed today (0–100)
  progress: number; // % of the items done
  state: 'done' | 'onTrack' | 'behind' | 'overdue';
  behind: number; // percentage points the progress lags behind the time
}

const BEHIND_TOLERANCE = 10; // up to 10 points behind still counts as "on track"

export function schedule(p: SidebarProject, today: string): Schedule | null {
  if (!p.due_date) return null;
  const start = p.start_date ?? p.created_at;
  const len = Math.max(1, daysUntil(p.due_date, start)); // days from start to due
  const elapsed = Math.min(100, Math.max(0, Math.round((daysUntil(today, start) / len) * 100)));
  const progress = pct(p.done, p.total);
  const behind = Math.max(0, elapsed - progress);
  let state: Schedule['state'] = 'onTrack';
  if (p.status === 'done' || (p.total > 0 && p.done === p.total)) state = 'done';
  else if (daysUntil(p.due_date, today) < 0) state = 'overdue';
  else if (behind > BEHIND_TOLERANCE) state = 'behind';
  return { start, due: p.due_date, elapsed, progress, state, behind };
}
