// Kundenübersicht: Kennzahlen und Status je Kunde, berechnet aus dem
// Sidebar-Baum (Projekte, Checklisten, Fortschritt) und /api/customers.
import type { CustomerListItem, SidebarCustomer, SidebarProject } from './types';
import { dueUrgent } from './overview';
import { customerProgress } from './projectStatus';

/** neu = noch keine Projekte, kritisch = blockiert oder (fast) überfällig, erledigt = alles fertig */
export type CustomerStatus = 'new' | 'onTrack' | 'critical' | 'done';

export const CUSTOMER_STATUSES: readonly CustomerStatus[] = ['onTrack', 'critical', 'new', 'done'];

export const CUSTOMER_STATUS_BADGE: Record<CustomerStatus, { badge: string; dot: string }> = {
  onTrack: { badge: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400', dot: 'bg-emerald-500' },
  critical: { badge: 'bg-red-50 text-red-700 dark:bg-red-950/50 dark:text-red-400', dot: 'bg-red-500' },
  new: { badge: 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-400', dot: 'bg-accent-500' },
  done: { badge: 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300', dot: 'bg-zinc-400' }
};

export interface CustomerRow {
  customer: SidebarCustomer;
  info: CustomerListItem | undefined;
  projects: SidebarProject[]; // ohne archivierte
  activeProjects: number; // weder erledigt noch archiviert
  open: number; // offene Einträge aller Projekte
  done: number; // Kundenfortschritt (nur einbezogene Projekte)
  total: number;
  excluded: number; // Projekte ausserhalb des Kundenfortschritts
  nextDue: string | null; // nächste Fälligkeit eines nicht erledigten Projekts
  nextDueUrgent: boolean;
  status: CustomerStatus;
  lastActivity: string;
}

export function buildCustomerRows(customers: SidebarCustomer[], infos: CustomerListItem[], today: string): CustomerRow[] {
  const byId = new Map(infos.map((i) => [i.id, i]));
  return customers.map((c) => {
    const projects = c.projects.filter((p) => p.status !== 'archived');
    const running = projects.filter((p) => p.status !== 'done');
    const due = running.filter((p) => p.due_date).sort((a, b) => a.due_date!.localeCompare(b.due_date!))[0];
    const { done, total, excluded } = customerProgress(projects);
    const open = projects.reduce((s, p) => s + p.total - p.done, 0);
    const critical = running.some((p) => p.status === 'blocked' || dueUrgent(p, today));
    const status: CustomerStatus = !projects.length ? 'new' : critical ? 'critical' : running.length ? 'onTrack' : 'done';
    return {
      customer: c,
      info: byId.get(c.id),
      projects,
      activeProjects: running.length,
      open,
      done,
      total,
      excluded,
      nextDue: due?.due_date ?? null,
      nextDueUrgent: !!due && dueUrgent(due, today),
      status,
      lastActivity: byId.get(c.id)?.last_activity ?? ''
    };
  });
}

export type CustomerFilter = 'all' | 'active' | 'critical' | 'empty';
export const CUSTOMER_FILTERS: readonly CustomerFilter[] = ['all', 'active', 'critical', 'empty'];

export function matchesFilter(r: CustomerRow, f: CustomerFilter): boolean {
  if (f === 'active') return r.activeProjects > 0;
  if (f === 'critical') return r.status === 'critical';
  if (f === 'empty') return r.projects.length === 0;
  return true;
}

export type CustomerSort = 'activity' | 'name' | 'progress' | 'due' | 'open';
export const CUSTOMER_SORTS: readonly CustomerSort[] = ['activity', 'name', 'progress', 'due', 'open'];

const pctOf = (r: CustomerRow) => (r.total ? r.done / r.total : -1);

export function sortRows(rows: CustomerRow[], s: CustomerSort): CustomerRow[] {
  const byName = (a: CustomerRow, b: CustomerRow) => a.customer.name.localeCompare(b.customer.name, undefined, { sensitivity: 'base' });
  const cmp: Record<CustomerSort, (a: CustomerRow, b: CustomerRow) => number> = {
    activity: (a, b) => b.lastActivity.localeCompare(a.lastActivity) || byName(a, b),
    name: byName,
    progress: (a, b) => pctOf(b) - pctOf(a) || byName(a, b),
    due: (a, b) => (a.nextDue ?? '9999').localeCompare(b.nextDue ?? '9999') || byName(a, b),
    open: (a, b) => b.open - a.open || byName(a, b)
  };
  return [...rows].sort(cmp[s]);
}
