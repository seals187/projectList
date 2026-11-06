// In-app navigation history for the back bar (pure functions, testable):
// a stack of visited pages with the current position, kept in step with the
// browser history (back/forward = popstate with delta), and the description of
// each entry (name, type icon) plus the breadcrumb of the current page.
import type { EntityType } from './entityTypes';
import type { SidebarCustomer, TemplateSummary } from './types';

export interface HistEntry {
  path: string;
  search: string;
}
export interface Hist {
  entries: HistEntry[];
  index: number;
}

const MAX = 50;
const same = (a: HistEntry | undefined, b: HistEntry) => !!a && a.path === b.path && a.search === b.search;

/** Next state after a navigation (SvelteKit afterNavigate: type and, for popstate, delta). */
export function applyNav(h: Hist | null, nav: { type: string; delta?: number } & HistEntry): Hist {
  const e: HistEntry = { path: nav.path, search: nav.search };
  if (!h || !h.entries.length) return { entries: [e], index: 0 };
  if (nav.type === 'enter') {
    // reload: keep the stored history if it still points at this page
    return h.entries[h.index]?.path === e.path ? { entries: h.entries.map((x, i) => (i === h.index ? e : x)), index: h.index } : { entries: [e], index: 0 };
  }
  if (nav.type === 'popstate' && typeof nav.delta === 'number') {
    const index = Math.min(h.entries.length - 1, Math.max(0, h.index + nav.delta));
    if (h.entries[index]?.path === e.path) return { entries: h.entries.map((x, i) => (i === index ? e : x)), index };
    // out of step (e.g. history from before the app): start over from here
    return { entries: [...h.entries.slice(0, h.index + 1), e].slice(-MAX), index: Math.min(h.index + 1, MAX - 1) };
  }
  if (same(h.entries[h.index], e)) return h;
  const entries = [...h.entries.slice(0, h.index + 1), e].slice(-MAX);
  return { entries, index: entries.length - 1 };
}

/** Earlier entries, newest first, with the step for history.go() (negative). */
export function backList(h: Hist, max = 8): { entry: HistEntry; delta: number }[] {
  const out: { entry: HistEntry; delta: number }[] = [];
  for (let i = h.index - 1; i >= 0 && out.length < max; i--) out.push({ entry: h.entries[i], delta: i - h.index });
  return out;
}

// ---- Descriptions ------------------------------------------------------------------

export type PageIcon = 'home' | 'tasks' | 'building2' | 'template' | 'settings';
export interface Described {
  label: string;
  type?: EntityType; // object page: type icon (colored)
  icon?: PageIcon; // list page: plain icon
}
export interface NavData {
  customers: readonly SidebarCustomer[];
  templates: readonly TemplateSummary[] | null;
}
export interface NavTexts {
  overview: string;
  myTasks: string;
  customers: string;
  templates: string;
  settings: string;
  view: (v: string) => string | null; // overview view name ("Liste")
  settingsPage: (slug: string) => string | null;
}

function lookup(data: NavData) {
  const projects = new Map<number, { name: string; customer: SidebarCustomer }>();
  const lists = new Map<number, { title: string; projectId: number }>();
  for (const c of data.customers)
    for (const p of c.projects) {
      projects.set(p.id, { name: p.name, customer: c });
      for (const cl of p.checklists) lists.set(cl.id, { title: cl.title, projectId: p.id });
    }
  return { projects, lists };
}

/** Breadcrumb of a page: always starts at the overview (the start page), then the page or the object with its parents. */
export function crumbsFor(path: string, search: string, data: NavData, tx: NavTexts): (Described & { href?: string })[] {
  const rest = pageCrumbs(path, search, data, tx);
  return path === '/' ? rest : [{ label: tx.overview, icon: 'home', href: '/' }, ...rest];
}

function pageCrumbs(path: string, search: string, data: NavData, tx: NavTexts): (Described & { href?: string })[] {
  const m = path.match(/^\/(customers|projects|checklists|templates)\/(\d+)/);
  const { projects, lists } = lookup(data);
  if (m) {
    const id = Number(m[2]);
    // customers, projects and checklists sit below the customers list:
    // customers / customer / project / checklist
    const customersCrumb = { label: tx.customers, icon: 'building2' as const, href: '/customers' };
    const customerCrumb = (c: SidebarCustomer) => ({ label: c.name, type: 'customer' as const, href: `/customers/${c.id}` });
    if (m[1] === 'customers') {
      const c = data.customers.find((x) => x.id === id);
      return [customersCrumb, c ? { label: c.name, type: 'customer' } : { label: '…', type: 'customer' }];
    }
    if (m[1] === 'projects') {
      const p = projects.get(id);
      return p ? [customersCrumb, customerCrumb(p.customer), { label: p.name, type: 'project' }] : [customersCrumb, { label: '…', type: 'project' }];
    }
    if (m[1] === 'checklists') {
      const l = lists.get(id);
      const p = l ? projects.get(l.projectId) : undefined;
      return l && p
        ? [customersCrumb, customerCrumb(p.customer), { label: p.name, type: 'project', href: `/projects/${l.projectId}` }, { label: l.title, type: 'checklist' }]
        : [customersCrumb, { label: '…', type: 'checklist' }];
    }
    const tpl = data.templates?.find((x) => x.id === id);
    return [{ label: tx.templates, icon: 'template', href: '/templates' }, { label: tpl?.name ?? '…', type: 'template' }];
  }
  return [describe(path, search, data, tx)];
}

/** One line of the history menu: name and type icon / page icon. */
export function describe(path: string, search: string, data: NavData, tx: NavTexts): Described {
  const m = path.match(/^\/(customers|projects|checklists|templates)\/(\d+)/);
  if (m) {
    const crumbs = pageCrumbs(path, search, data, tx);
    const last = crumbs[crumbs.length - 1];
    return { label: last.label, type: last.type };
  }
  if (path === '/') {
    const v = new URLSearchParams(search).get('view');
    const name = v ? tx.view(v) : null;
    return { label: name ? `${tx.overview} · ${name}` : tx.overview, icon: 'home' };
  }
  if (path.startsWith('/tasks')) return { label: tx.myTasks, icon: 'tasks' };
  if (path.startsWith('/customers')) return { label: tx.customers, icon: 'building2' };
  if (path.startsWith('/templates')) return { label: tx.templates, icon: 'template' };
  if (path.startsWith('/settings')) {
    const sub = tx.settingsPage(path.split('/')[2] ?? '');
    return { label: sub ? `${tx.settings} · ${sub}` : tx.settings, icon: 'settings' };
  }
  return { label: path, icon: 'home' };
}

/** Clears the history: only the current page remains (no back / forward). */
export function clearHistory(h: Hist): Hist {
  const cur = h.entries[h.index];
  return cur ? { entries: [cur], index: 0 } : { entries: [], index: 0 };
}
