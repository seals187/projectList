// Overview list: steps/progress per project and customer, and how many tags fit
// into a given width. Pure functions (unit tested).
//
// Steps of a project = done / all items of all its checklists (headings do not
// count; the server already aggregates this per project in SQL, see sidebar).

export interface StepsLike {
  done: number;
  total: number;
}
export interface ProjectLike extends StepsLike {
  status: string;
  due_date: string | null;
  include_in_customer_progress: boolean;
}

/** Progress in percent (rounded); no steps = 0. */
export const percent = (done: number, total: number) => (total > 0 ? Math.round((done / total) * 100) : 0);

export interface CustomerSummary {
  projects: number;
  done: number; // steps of the included projects
  total: number;
  pct: number;
  open: number; // open steps of all projects
  next: string | null; // next due date (today or later) of a running project
}

/** Customer: steps and progress over the projects included in the customer progress. */
export function customerSummary(projects: ProjectLike[], today: string): CustomerSummary {
  const included = projects.filter((p) => p.include_in_customer_progress);
  const done = included.reduce((s, p) => s + p.done, 0);
  const total = included.reduce((s, p) => s + p.total, 0);
  const open = projects.reduce((s, p) => s + Math.max(0, p.total - p.done), 0);
  const next =
    projects
      .filter((p) => p.due_date && p.due_date >= today && p.status !== 'done' && p.status !== 'archived')
      .map((p) => p.due_date!)
      .sort()[0] ?? null;
  return { projects: projects.length, done, total, pct: percent(done, total), open, next };
}

/**
 * How many tags fit: all if they fit completely, otherwise as many as fit next
 * to a "+N" marker. widths = rendered tag widths, gap = space between items.
 */
export function fitTags(widths: number[], avail: number, plusWidth: number, gap: number): number {
  const sum = (n: number) => widths.slice(0, n).reduce((s, w) => s + w, 0) + Math.max(0, n - 1) * gap;
  if (sum(widths.length) <= avail) return widths.length;
  let n = widths.length - 1;
  while (n > 0 && sum(n) + gap + plusWidth > avail) n--;
  return n;
}

/** "dd.mm." */
export const dayMonth = (iso: string) => `${iso.slice(8, 10)}.${iso.slice(5, 7)}.`;
