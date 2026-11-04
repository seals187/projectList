// Datenmodell der Übersicht: Sidebar-Baum (Kunden → Projekte → Checklisten)
// ergänzt um Team und nächsten Schritt aus /api/overview.
import type { OverviewExtra, SidebarChecklist, SidebarCustomer, SidebarProject } from './types';
import { customerProgress, daysUntil } from './projectStatus';
import { htmlToText } from './rich';

export interface OvProject extends SidebarProject {
  customer: SidebarCustomer;
  team: { id: number; name: string }[];
  next: string; // Klartext
}

export interface OvCustomer {
  customer: SidebarCustomer;
  projects: OvProject[];
  done: number; // Kundenfortschritt (nur einbezogene Projekte)
  total: number;
  excluded: number; // Projekte, die nicht in den Kundenfortschritt einfliessen
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

/** Fälligkeit hervorheben: nicht fertig und in ≤ 3 Tagen fällig oder überfällig. */
export function dueUrgent(p: SidebarProject, today: string): boolean {
  return !!p.due_date && p.status !== 'done' && daysUntil(p.due_date, today) <= 3;
}

export const pct = (done: number, total: number) => (total ? Math.round((done / total) * 100) : 0);

/** Segmentbreite einer Checkliste im geteilten Balken (mind. etwas sichtbar). */
export const segGrow = (cl: SidebarChecklist) => Math.max(cl.total, 1);
