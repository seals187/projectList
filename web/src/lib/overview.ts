// Overview data model: sidebar tree (customers → projects → checklists)
// extended with team and next step from /api/overview.
import type { OverviewExtra, SidebarChecklist, SidebarCustomer, SidebarProject } from './types';
import { customerProgress, daysUntil } from './projectStatus';
import { htmlToText } from './rich';

export interface OvProject extends SidebarProject {
  customer: SidebarCustomer;
  team: { id: number; name: string }[];
  next: string; // plain text
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
        return { ...p, customer: c, team: x?.team ?? [], next: x?.next_step ? htmlToText(x.next_step) : '' };
      });
    return { customer: c, projects, ...customerProgress(projects) };
  });
}

/** Highlight due date: not done and due in ≤ 3 days or overdue. */
export function dueUrgent(p: SidebarProject, today: string): boolean {
  return !!p.due_date && p.status !== 'done' && daysUntil(p.due_date, today) <= 3;
}

export const pct = (done: number, total: number) => (total ? Math.round((done / total) * 100) : 0);

/** Segment width of a checklist in the split bar (at least slightly visible). */
export const segGrow = (cl: SidebarChecklist) => Math.max(cl.total, 1);
