// Project query: the one place that turns the sidebar tree (customers →
// projects → checklists) into what the overview views and "My tasks" render.
// It owns the project concepts (progress, open, due, overdue), filtering,
// search, sorting and grouping, the per-view orderings, the board drop order
// and the customer-filter transitions. Pure functions only (no stores, no
// i18n): labels are passed in by the caller. Unit tested in projectQuery.test.ts.
import type { OverviewExtra, ProjectStatus, SidebarChecklist, SidebarCustomer, SidebarProject, Task } from './types';
import { daysUntil, toDay } from './dates';
import { htmlToText } from './rich';

// ---- Concepts ---------------------------------------------------------------------

/** Order of the columns on the board (without "archived"). */
export const BOARD_STATUSES = ['planning', 'active', 'review', 'blocked', 'done'] as const satisfies readonly ProjectStatus[];

/** Person responsible "nobody" (owner filter, owner columns). */
export const NO_OWNER = 0;

type Steps = { done: number; total: number };

/** Progress in percent (rounded); no items = 0. */
export const pct = (done: number, total: number) => (total ? Math.round((done / total) * 100) : 0);

/** Progress as a ratio for sorting; no items = -1 (sorts after 0 %). */
export const progressRatio = (s: Steps) => (s.total ? s.done / s.total : -1);

/** Open items of a project (or customer sum). */
export const openItems = (s: Steps) => Math.max(0, s.total - s.done);

/** A checklist still has work: no items yet or not everything checked. */
export const checklistOpen = (cl: Pick<SidebarChecklist, 'done' | 'total'>) => cl.total === 0 || cl.done < cl.total;

/** A project is complete: status "done" or all its items checked. */
export const projectComplete = (p: Pick<SidebarProject, 'status' | 'done' | 'total'>) => p.status === 'done' || (p.total > 0 && p.done === p.total);

/**
 * Project due date ("due" on the overview): the project's own due date while it
 * is not done; '' otherwise. Used by the overview sort, urgency and the cards.
 */
export const projectDue = (p: { due_date: string | null; status: string }): string => (p.status !== 'done' ? (p.due_date ?? '') : '');

/** All upcoming deadlines of a project: open checklists' next due dates plus its project due date. */
export const dueDates = (p: Pick<SidebarProject, 'due_date' | 'status' | 'checklists'>): string[] =>
  [...p.checklists.filter((c) => checklistOpen(c) && c.next_due).map((c) => c.next_due), projectDue(p)].filter(Boolean);

/**
 * Next due date ("due" on My tasks): the earliest open checklist due date, else
 * the project due date; '' = none. Differs from projectDue on purpose: My
 * tasks is about the next thing to do, the overview about project deadlines.
 */
export const nextDue = (p: Pick<SidebarProject, 'due_date' | 'status' | 'checklists'>): string => dueDates(p).sort()[0] ?? '';

/** Highlight due date: not done and due in ≤ 3 days or overdue. */
export function dueUrgent(p: Pick<SidebarProject, 'due_date' | 'status'>, today: string): boolean {
  const d = projectDue(p);
  return !!d && daysUntil(d, today) <= 3;
}

/** Overdue items of an overview project (server count per checklist). */
export const overdueItems = (p: Pick<OvProject, 'clExtra'>) => Object.values(p.clExtra).reduce((n, c) => n + c.overdue, 0);

export type DueBucket = 'overdue' | 'today' | 'week' | 'later' | 'none';
export const DUE_BUCKETS: readonly DueBucket[] = ['overdue', 'today', 'week', 'later', 'none'];

/** Due date bucket relative to today (filter "Due"). */
export function dueBucket(d: string | null | undefined, today: string): DueBucket {
  if (!d) return 'none';
  const n = daysUntil(d, today);
  return n < 0 ? 'overdue' : n === 0 ? 'today' : n <= 6 ? 'week' : 'later';
}

/** Segment width of a checklist in the split bar (at least slightly visible). */
export const segGrow = (cl: SidebarChecklist) => Math.max(cl.total, 1);

/**
 * Overall progress of a customer: done / all items, only across projects
 * that count toward the customer progress (include_in_customer_progress).
 * Archived projects never count. `excluded` = number of deliberately excluded projects.
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

export const byName = (a: string, b: string) => a.localeCompare(b, undefined, { sensitivity: 'base' });

// ---- Overview model ---------------------------------------------------------------

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

/** Sidebar tree extended with team and next step from /api/overview (archived projects left out). */
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
  if (projectComplete(p)) state = 'done';
  else if (daysUntil(p.due_date, today) < 0) state = 'overdue';
  else if (behind > BEHIND_TOLERANCE) state = 'behind';
  return { start, due: p.due_date, elapsed, progress, state, behind };
}

// ---- Overview: filter, search, sort -------------------------------------------------

export type OverviewSort = 'activity' | 'name' | 'progress' | 'due';
export const OVERVIEW_SORTS: readonly OverviewSort[] = ['activity', 'name', 'progress', 'due'];

/** Label and owner filter (empty list = all). */
export interface ProjectFilter {
  tags: number[]; // projects with at least one of these labels
  owners: number[]; // person responsible (NO_OWNER = nobody)
}

/** Whether a project passes the label and owner filters (customer filter is separate). */
export function matchesProjectFilter(p: Pick<SidebarProject, 'tag_ids' | 'owner_id'>, f: ProjectFilter): boolean {
  if (f.tags.length && !p.tag_ids.some((id) => f.tags.includes(id))) return false;
  if (f.owners.length && !f.owners.includes(p.owner_id ?? NO_OWNER)) return false;
  return true;
}

/** Hidden customers that still exist (IDs of deleted customers are ignored). */
export const effectiveHidden = (groups: OvCustomer[], hidden: number[]) => new Set(hidden.filter((id) => groups.some((g) => g.customer.id === id)));

/** Search: customer name (keeps all its projects) or project name. */
export function searchOverview(groups: OvCustomer[], q: string): OvCustomer[] {
  const s = q.trim().toLowerCase();
  if (!s) return groups;
  return groups
    .map((g) => (g.customer.name.toLowerCase().includes(s) ? g : { ...g, projects: g.projects.filter((p) => p.name.toLowerCase().includes(s)) }))
    .filter((g) => g.projects.length > 0);
}

/** Customers and their projects in the chosen order (the board keeps its own column order). */
export function sortOverview(groups: OvCustomer[], sort: OverviewSort): OvCustomer[] {
  const latest = (g: OvCustomer) => g.projects.reduce((m, p) => (p.updated > m ? p.updated : m), '');
  const firstDue = (ps: OvProject[]) => ps.map(projectDue).filter(Boolean).sort()[0] ?? '9999';
  const pc: Record<OverviewSort, (a: OvProject, b: OvProject) => number> = {
    activity: (a, b) => b.updated.localeCompare(a.updated) || byName(a.name, b.name),
    name: (a, b) => byName(a.name, b.name),
    progress: (a, b) => progressRatio(b) - progressRatio(a) || byName(a.name, b.name),
    due: (a, b) => firstDue([a]).localeCompare(firstDue([b])) || byName(a.name, b.name)
  };
  const gc: Record<OverviewSort, (a: OvCustomer, b: OvCustomer) => number> = {
    activity: (a, b) => latest(b).localeCompare(latest(a)) || byName(a.customer.name, b.customer.name),
    name: (a, b) => byName(a.customer.name, b.customer.name),
    progress: (a, b) => progressRatio(b) - progressRatio(a) || byName(a.customer.name, b.customer.name),
    due: (a, b) => firstDue(a.projects).localeCompare(firstDue(b.projects)) || byName(a.customer.name, b.customer.name)
  };
  return groups.map((g) => ({ ...g, projects: [...g.projects].sort(pc[sort]) })).sort(gc[sort]);
}

export interface OverviewQuery extends ProjectFilter {
  hidden: number[]; // hidden customers
  q: string;
  sort: OverviewSort;
}

/**
 * What all overview views show: visible customers with their matching projects
 * (with a label/person filter, customers without a matching project are left
 * out), then search and sorting.
 */
export function queryOverview(groups: OvCustomer[], query: OverviewQuery): OvCustomer[] {
  const hidden = effectiveHidden(groups, query.hidden);
  const projectFilter = query.tags.length > 0 || query.owners.length > 0;
  const visible = groups
    .filter((g) => !hidden.has(g.customer.id))
    .map((g) => ({ ...g, projects: g.projects.filter((p) => matchesProjectFilter(p, query)) }))
    .filter((g) => !projectFilter || g.projects.length > 0);
  return sortOverview(searchOverview(visible, query.q), query.sort);
}

export type CustomerFilterMode = 'only' | 'toggle' | 'all';

/** Customer filter from a customer menu: new list of hidden customers. */
export function nextHidden(hidden: number[], allIds: number[], id: number, mode: CustomerFilterMode): number[] {
  if (mode === 'all') return [];
  if (mode === 'only') return allIds.filter((x) => x !== id);
  return hidden.includes(id) ? hidden.filter((x) => x !== id) : [...hidden, id];
}

// ---- Board ----------------------------------------------------------------------------

export type BoardGroup = 'status' | 'customer' | 'owner';
export const BOARD_GROUPS: readonly BoardGroup[] = ['status', 'customer', 'owner'];

export interface BoardColumn {
  key: string;
  title: string;
  status?: ProjectStatus; // grouped by status: drop target, WIP limit, status menu
  customer?: SidebarCustomer;
  owner?: { id: number; name: string };
  items: OvProject[];
}

// order within a status column: board position (stable sort keeps the incoming order)
const byPosition = (a: SidebarProject, b: SidebarProject) => a.board_position - b.board_position;
// customer/person columns: projects in board order (status, then position)
const statusIndex = (s: ProjectStatus) => (BOARD_STATUSES as readonly ProjectStatus[]).indexOf(s);
const byStatus = (a: SidebarProject, b: SidebarProject) => statusIndex(a.status) - statusIndex(b.status) || byPosition(a, b);

/** Board columns for the grouping; labels come from the caller (i18n). */
export function boardColumns(groups: OvCustomer[], group: BoardGroup, labels: { status: (s: ProjectStatus) => string; noOwner: string }): BoardColumn[] {
  const projects = groups.flatMap((g) => g.projects);
  if (group === 'customer') return groups.map((g) => ({ key: `c${g.customer.id}`, title: g.customer.name, customer: g.customer, items: [...g.projects].sort(byStatus) }));
  if (group === 'owner') {
    const m = new Map<number, BoardColumn>();
    for (const p of projects) {
      const id = p.owner_id ?? NO_OWNER;
      if (!m.has(id)) m.set(id, { key: `o${id}`, title: id ? p.owner_name : labels.noOwner, owner: { id, name: p.owner_name }, items: [] });
      m.get(id)!.items.push(p);
    }
    // people alphabetically, "nobody" last
    return [...m.values()]
      .sort((a, b) => (a.owner!.id === NO_OWNER ? 1 : b.owner!.id === NO_OWNER ? -1 : a.title.localeCompare(b.title)))
      .map((c) => ({ ...c, items: c.items.sort(byStatus) }));
  }
  return BOARD_STATUSES.map((s) => ({ key: s, title: labels.status(s), status: s, items: projects.filter((p) => p.status === s).sort(byPosition) }));
}

/**
 * Drop on the board: new order (IDs) of the whole target column, including
 * filtered-out projects (they keep their place). `before` = card before which
 * the project is inserted (null = after the last visible card). null = nothing changed.
 */
export function boardDropOrder(
  everything: SidebarProject[],
  visibleIds: Set<number>,
  dragged: SidebarProject,
  status: ProjectStatus,
  before: number | null
): number[] | null {
  const column = everything.filter((x) => x.status === status && x.id !== dragged.id).sort(byPosition);
  const visible = column.filter((x) => visibleIds.has(x.id));
  let at = before === null ? -1 : column.findIndex((x) => x.id === before);
  if (at < 0) at = visible.length ? column.indexOf(visible[visible.length - 1]) + 1 : column.length;
  const ids = column.map((x) => x.id);
  ids.splice(at, 0, dragged.id);
  const old = everything.filter((x) => x.status === status).sort(byPosition).map((x) => x.id);
  if (dragged.status === status && old.join() === ids.join()) return null;
  return ids;
}

// ---- Cards, timeline, calendar ------------------------------------------------------------

export interface CustomerCard extends OvCustomer {
  overdue: number; // overdue items
  open: number; // open items
  next: string; // next deadline today or later (open checklists or project), '' = none
  team: { id: number; name: string }[];
}

/** Cards view: per customer overdue/open items, next deadline, team; overdue first, then customers with projects. */
export function customerCards(groups: OvCustomer[], today: string): CustomerCard[] {
  return groups
    .map((g) => {
      const overdue = g.projects.reduce((n, p) => n + overdueItems(p), 0);
      const open = g.projects.reduce((n, p) => n + openItems(p), 0);
      const dates = g.projects
        .filter((p) => p.status !== 'done')
        .flatMap(dueDates)
        .filter((d) => d >= today)
        .sort();
      const team = new Map<number, { id: number; name: string }>();
      for (const p of g.projects) for (const m of p.team) team.set(m.id, m);
      return { ...g, overdue, open, next: dates[0] ?? '', team: [...team.values()] };
    })
    .sort((a, b) => Number(b.overdue > 0) - Number(a.overdue > 0) || Number(b.projects.length > 0) - Number(a.projects.length > 0));
}

/** Timeline bar of a project as day numbers [start, end]: start (or creation) date to due date. */
export function timelineRange(p: Pick<SidebarProject, 'start_date' | 'created_at' | 'due_date'>): [number, number] {
  const s = toDay(p.start_date ?? p.created_at);
  const e = p.due_date ? toDay(p.due_date) : s;
  return s <= e ? [s, e] : [e, s];
}

/** Timeline rows: per customer, projects by start, then name. */
export function timelineRows(groups: OvCustomer[]): { customer: SidebarCustomer; projects: OvProject[] }[] {
  return groups.map((g) => ({
    customer: g.customer,
    projects: [...g.projects].sort((a, b) => timelineRange(a)[0] - timelineRange(b)[0] || a.name.localeCompare(b.name))
  }));
}

/** Calendar: projects per due day (day number) and those without a due date. */
export function calendarDays<P extends Pick<SidebarProject, 'due_date'>>(projects: P[]): { byDay: Map<number, P[]>; undated: P[] } {
  const byDay = new Map<number, P[]>();
  for (const p of projects) {
    if (!p.due_date) continue;
    const d = toDay(p.due_date);
    if (!byDay.has(d)) byDay.set(d, []);
    byDay.get(d)!.push(p);
  }
  return { byDay, undated: projects.filter((p) => !p.due_date) };
}

// ---- My tasks -----------------------------------------------------------------------------

export type TaskChip = 'open' | 'today' | 'done';
export const TASK_CHIPS: readonly TaskChip[] = ['open', 'today', 'done'];
export type TaskSort = 'due' | 'name' | 'progress';
export const TASK_SORTS: readonly TaskSort[] = ['due', 'name', 'progress'];

/** I am responsible for the project. */
export const isOwner = (p: Pick<SidebarProject, 'owner_id'>, userId: number | null) => userId !== null && p.owner_id === userId;

/** Mine: I am responsible, or something in it (checklist or item) is assigned to me. */
export const isMine = (p: SidebarProject, userId: number | null, extra: Record<string, OverviewExtra>) =>
  isOwner(p, userId) || (userId !== null && !!extra[p.id]?.team.some((m) => m.id === userId));

/** Quick filter chip of My tasks. */
export function matchesChip(p: SidebarProject, chip: TaskChip, today: string): boolean {
  if (chip === 'done') return projectComplete(p);
  if (p.status === 'done') return false;
  if (chip === 'today') return p.checklists.some((cl) => !!cl.next_due && cl.next_due <= today);
  return p.total === 0 || p.done < p.total || p.checklists.some(checklistOpen);
}

export interface TaskGroup {
  customer: SidebarCustomer;
  projects: SidebarProject[];
  checklists: number;
}

/** Customers with my projects passing the chip (before the header filters: customer filter entries). */
export function myCustomers(
  customers: SidebarCustomer[],
  o: { userId: number | null; extra: Record<string, OverviewExtra>; chip: TaskChip; today: string }
): { customer: SidebarCustomer; projects: SidebarProject[] }[] {
  return customers
    .map((c) => ({ customer: c, projects: c.projects.filter((p) => p.status !== 'archived' && isMine(p, o.userId, o.extra) && matchesChip(p, o.chip, o.today)) }))
    .filter((g) => g.projects.length > 0);
}

export interface TaskQuery {
  cust: (string | number)[] | null; // customer filter (null = all)
  due: (string | number)[] | null; // due buckets (null = all)
  q: string;
  sort: TaskSort;
  today: string;
}

/** Text search: any of the strings contains the query. */
const hit = (q: string, ...s: (string | undefined | null)[]) => {
  const n = q.trim().toLowerCase();
  return !n || s.some((x) => x?.toLowerCase().includes(n));
};

const taskProjectCmp: Record<TaskSort, (a: SidebarProject, b: SidebarProject) => number> = {
  due: (a, b) => (nextDue(a) || '9999').localeCompare(nextDue(b) || '9999') || byName(a.name, b.name),
  name: (a, b) => byName(a.name, b.name),
  progress: (a, b) => progressRatio(b) - progressRatio(a) || byName(a.name, b.name)
};

/** My tasks, customer view: header filters (customer, due bucket of the next due date, search), sorted. */
export function taskGroups(mine: { customer: SidebarCustomer; projects: SidebarProject[] }[], o: TaskQuery): TaskGroup[] {
  const cmp = taskProjectCmp[o.sort];
  const list = mine
    .filter((g) => o.cust === null || o.cust.includes(g.customer.id))
    .map((g) => {
      const projects = g.projects
        .filter((p) => (o.due === null || o.due.includes(dueBucket(nextDue(p), o.today))) && (hit(o.q, g.customer.name, p.name) || p.checklists.some((c) => hit(o.q, c.title))))
        .sort(cmp);
      return { customer: g.customer, projects, checklists: projects.reduce((s, p) => s + p.checklists.length, 0) };
    })
    .filter((g) => g.projects.length > 0);
  return list.sort((a, b) => cmp(a.projects[0], b.projects[0]) || byName(a.customer.name, b.customer.name));
}

/** My tasks, personal view: source list for the chip ("today" = due today or earlier). */
export const personalSource = (tasks: Task[], chip: TaskChip, today: string) => (chip === 'today' ? tasks.filter((x) => x.due_date && x.due_date <= today) : tasks);

/** My tasks, personal view: header filters and sorting ("progress" keeps the server order). */
export function personalTasks(src: Task[], o: TaskQuery): Task[] {
  const list = src.filter(
    (x) =>
      (o.cust === null || o.cust.includes(x.customer_id)) &&
      (o.due === null || o.due.includes(dueBucket(x.due_date, o.today))) &&
      hit(o.q, x.title, x.project_name, x.customer_name, x.checklist_title)
  );
  if (o.sort === 'due') list.sort((a, b) => (a.due_date ?? '9999').localeCompare(b.due_date ?? '9999'));
  if (o.sort === 'name') list.sort((a, b) => byName(a.title, b.title));
  return list;
}
