// Workspace store: the app's main read model (/api/sidebar: customers →
// projects → checklists/tables, favorites, tags, status settings) plus the
// named write operations that change it. Each operation owns its API URL, the
// optimistic update of the read model, the rollback on error and the
// invalidation (debounced reload of /api/sidebar). Operations reject with the
// API error after rolling back; callers show it (toast) as before.
//
// createWorkspace() takes the transport and timers, so the tests run against
// an in-memory fake (workspace.test.ts); the app uses the singleton below.
import { api } from './api';
import { syncPrefs } from './syncedPref.svelte';
import type {
  Checklist,
  Customer,
  Favorite,
  FavoriteGroup,
  Project,
  ProjectStatus,
  SidebarChecklist,
  SidebarCustomer,
  SidebarData,
  SidebarProject,
  Table,
  Tag
} from './types';

export interface WorkspaceTransport {
  get<T>(url: string): Promise<T>;
  post<T>(url: string, body?: unknown): Promise<T>;
  put<T>(url: string, body: unknown): Promise<T>;
  patch<T>(url: string, body: unknown): Promise<T>;
  del(url: string): Promise<void>;
}

export interface WorkspaceTimers {
  set(fn: () => void, ms: number): unknown;
  clear(handle: unknown): void;
}

/** The read model as the UI reads it. */
export interface WorkspaceState {
  customers: SidebarCustomer[];
  myTasks: number;
  favorites: Favorite[]; // customers, projects, checklists, templates (DB, per user)
  favoriteGroups: FavoriteGroup[]; // user-defined favorites groups (DB, per user)
  tags: Tag[]; // tag catalog (global)
  statusLabels: Partial<Record<ProjectStatus, string>>; // custom status names (global, set by admins)
  statusColors: Partial<Record<ProjectStatus, string>>; // custom status colors #rrggbb (global, set by admins)
  statusWip: Partial<Record<ProjectStatus, number>>; // WIP limit per board column (global, set by admins)
  prefs: Record<string, unknown>; // synced personal settings as last received (live values: syncedPref modules)
  loaded: boolean; // first /sidebar response arrived
}

export const emptyWorkspace = (): WorkspaceState => ({
  customers: [],
  myTasks: 0,
  favorites: [],
  favoriteGroups: [],
  tags: [],
  statusLabels: {},
  statusColors: {},
  statusWip: {},
  prefs: {},
  loaded: false
});

/** Copies a /api/sidebar response into the read model. */
export function applySidebarData(s: WorkspaceState, r: SidebarData) {
  s.customers = r.customers;
  s.myTasks = r.my_tasks;
  s.favorites = r.favorites ?? [];
  s.favoriteGroups = r.favorite_groups ?? [];
  s.tags = r.tags ?? [];
  s.statusLabels = r.status_labels ?? {};
  s.statusColors = r.status_colors ?? {};
  s.statusWip = r.status_wip ?? {};
  s.prefs = r.prefs ?? {};
  s.loaded = true;
}

export type ProjectPatch = Partial<Omit<Project, 'id' | 'customer_id' | 'created_at'>>;
export type CustomerPatch = Partial<Pick<Customer, 'name' | 'location' | 'notes' | 'contact_name' | 'contact_email' | 'phones'>>;
export interface CustomerColorPatch {
  color?: string;
  everywhere?: boolean;
}
export interface FromTemplate {
  template_id: number;
  title: string;
  values?: Record<string, string>;
}

const reorder = <T extends { id: number }>(list: T[], ids: number[]) =>
  [...list].sort((a, b) => (ids.indexOf(a.id) + 1 || Infinity) - (ids.indexOf(b.id) + 1 || Infinity));

const defaultTimers: WorkspaceTimers = {
  set: (fn, ms) => setTimeout(fn, ms),
  clear: (h) => clearTimeout(h as ReturnType<typeof setTimeout>)
};

export function createWorkspace(o: { transport: WorkspaceTransport; timers?: WorkspaceTimers; onLoaded?: (r: SidebarData) => void }) {
  const http = o.transport;
  const timers = o.timers ?? defaultTimers;
  const state = $state<WorkspaceState>(emptyWorkspace());

  // ---- Loading and invalidation ------------------------------------------------------
  let refreshTimer: unknown;
  /** Reloads the read model (debounced so rapid click sequences trigger only one request). */
  function refresh(delay = 400) {
    if (refreshTimer !== undefined) timers.clear(refreshTimer);
    refreshTimer = timers.set(async () => {
      refreshTimer = undefined;
      try {
        const r = await http.get<SidebarData>('/sidebar');
        applySidebarData(state, r);
        o.onLoaded?.(r);
      } catch {
        /* errors become visible on the next load */
      }
    }, delay);
  }

  let batchDepth = 0;
  let batchDirty = false;
  /** Something changed on the server: reload soon (once per batch). */
  function invalidate(delay = 0) {
    if (batchDepth > 0) batchDirty = true;
    else refresh(delay);
  }
  /** Runs several writes and reloads once at the end (also after an error). */
  async function batch<T>(run: () => Promise<T>): Promise<T> {
    batchDepth++;
    try {
      return await run();
    } finally {
      if (--batchDepth === 0 && batchDirty) {
        batchDirty = false;
        refresh(0);
      }
    }
  }

  /**
   * One write: optional optimistic change (returns its undo), the request, an
   * optional follow-up on success, then invalidation. On error: undo, rethrow.
   */
  async function write<T>(request: () => Promise<T>, w: { apply?: () => (() => void) | void; after?: (r: T) => void; invalidate?: boolean | number } = {}): Promise<T> {
    const undo = w.apply?.();
    let r: T;
    try {
      r = await request();
    } catch (e) {
      undo?.();
      throw e;
    }
    w.after?.(r);
    if (w.invalidate !== false) invalidate(typeof w.invalidate === 'number' ? w.invalidate : 0);
    return r;
  }

  // ---- Lookups ----------------------------------------------------------------------
  const projects = () => state.customers.flatMap((c) => c.projects);
  const findProject = (id: number): SidebarProject | undefined => projects().find((p) => p.id === id);
  const findChecklist = (id: number): SidebarChecklist | undefined => projects().flatMap((p) => p.checklists).find((c) => c.id === id);
  const findCustomer = (id: number): SidebarCustomer | undefined => state.customers.find((c) => c.id === id);
  const projectOfChecklist = (id: number) => projects().find((p) => p.checklists.some((c) => c.id === id));
  const projectOfTable = (id: number) => projects().find((p) => p.tables.some((t) => t.id === id));

  /** Optimistic field changes on an object of the read model; returns the undo. */
  function assign<T extends object>(target: T | undefined, patch: Partial<T>): (() => void) | undefined {
    if (!target) return;
    const prev = Object.fromEntries(Object.keys(patch).map((k) => [k, target[k as keyof T]])) as Partial<T>;
    Object.assign(target, patch);
    return () => Object.assign(target, prev);
  }

  // ---- Projects ----------------------------------------------------------------------
  const ops = {
    createProject: (customerId: number, body: { name: string; status?: ProjectStatus; due_date?: string | null }) =>
      write(() => http.post<Project>(`/customers/${customerId}/projects`, body)),

    /** PATCH /projects/{id}; shown at once in the read model (owner name too). */
    updateProject: (id: number, patch: ProjectPatch, ownerName?: string) =>
      write(() => http.patch<Project>(`/projects/${id}`, patch), {
        apply: () => {
          const p = findProject(id);
          const view: Partial<SidebarProject> = {};
          for (const k of ['name', 'status', 'start_date', 'due_date', 'owner_id', 'include_in_customer_progress'] as const) if (k in patch) (view as Record<string, unknown>)[k] = patch[k];
          if ('owner_id' in patch) view.owner_name = patch.owner_id ? (ownerName ?? '') : '';
          return assign(p, view);
        }
      }),

    setProjectTags: (id: number, tagIds: number[]) =>
      write(() => http.put<{ tag_ids: number[] }>(`/projects/${id}/tags`, { tag_ids: tagIds }), {
        apply: () => assign(findProject(id), { tag_ids: tagIds }),
        after: (r) => assign(findProject(id), { tag_ids: r.tag_ids })
      }).then((r) => r.tag_ids),

    /** Board drag & drop: new status and order (IDs) of the target column. */
    moveProject: (status: ProjectStatus, order: number[]) =>
      write(() => http.put('/board/order', { status, ids: order }), {
        apply: () => {
          const touched = projects().filter((x) => order.includes(x.id));
          const undo = touched.map((x) => assign(x, { status, board_position: order.indexOf(x.id) }));
          return () => undo.forEach((u) => u?.());
        },
        invalidate: false // the board already shows the result
      }),

    /** Background of a project ("", "g<n>" or "img:<id>"). */
    setProjectBackground: (id: number, background: string) =>
      write(() => http.put(`/projects/${id}/background`, { background }), { apply: () => assign(findProject(id), { background }), invalidate: false }),

    duplicateProject: (id: number, name: string) => write(() => http.post<Project>(`/projects/${id}/duplicate`, { name })),

    deleteProject: (id: number) =>
      write(() => http.del(`/projects/${id}`), {
        // drop it from the tree right away; the reload brings the rest in line
        after: () => {
          for (const c of state.customers) c.projects = c.projects.filter((x) => x.id !== id);
        }
      }),

    // ---- Checklists ----------------------------------------------------------------
    createChecklist: (projectId: number, title: string) => write(() => http.post<Checklist>(`/projects/${projectId}/checklists`, { title })),

    createChecklistFromTemplate: (projectId: number, body: FromTemplate) => write(() => http.post<Checklist>(`/projects/${projectId}/checklists/from-template`, body)),

    renameChecklist: (id: number, title: string) =>
      write(() => http.patch(`/checklists/${id}`, { title }), { apply: () => assign(findChecklist(id), { title }) }),

    assignChecklist: (id: number, assigneeId: number | null) => write(() => http.patch(`/checklists/${id}`, { assignee_id: assigneeId })),

    reorderChecklists: (projectId: number, ids: number[]) =>
      write(() => http.put(`/projects/${projectId}/checklists/order`, { ids }), {
        apply: () => {
          const p = findProject(projectId);
          return assign(p, p ? { checklists: reorder(p.checklists, ids) } : {});
        }
      }),

    deleteChecklist: (id: number) =>
      write(() => http.del(`/checklists/${id}`), {
        after: () => {
          const p = projectOfChecklist(id);
          if (p) p.checklists = p.checklists.filter((x) => x.id !== id);
        }
      }),

    // ---- Tables --------------------------------------------------------------------
    createTable: (projectId: number, body: { title: string; data?: unknown }) => write(() => http.post<Table>(`/projects/${projectId}/tables`, body)),

    createTableFromTemplate: (projectId: number, body: FromTemplate) => write(() => http.post<Table>(`/projects/${projectId}/tables/from-template`, body)),

    renameTable: (id: number, title: string) =>
      write(() => http.patch(`/tables/${id}`, { title }), {
        apply: () => assign(projectOfTable(id)?.tables.find((t) => t.id === id), { title })
      }),

    reorderTables: (projectId: number, ids: number[]) =>
      write(() => http.put(`/projects/${projectId}/tables/order`, { ids }), {
        apply: () => {
          const p = findProject(projectId);
          return assign(p, p ? { tables: reorder(p.tables, ids) } : {});
        }
      }),

    deleteTable: (id: number) =>
      write(() => http.del(`/tables/${id}`), {
        after: () => {
          const p = projectOfTable(id);
          if (p) p.tables = p.tables.filter((x) => x.id !== id);
        }
      }),

    // ---- Customers -----------------------------------------------------------------
    createCustomer: (body: { name: string; location?: string; contact_name?: string; contact_email?: string }) => write(() => http.post<Customer>('/customers', body)),

    /** PATCH /customers/{id}; reloads only when a field of the read model (name, location) changed. */
    updateCustomer: (id: number, patch: CustomerPatch) =>
      write(() => http.patch<Customer>(`/customers/${id}`, patch), {
        apply: () => {
          const view: Partial<SidebarCustomer> = {};
          if (patch.name !== undefined) view.name = patch.name;
          if (patch.location !== undefined) view.location = patch.location;
          return assign(findCustomer(id), view);
        },
        invalidate: 'name' in patch || 'location' in patch
      }),

    /** Shared customer color (only the given fields). */
    setCustomerColors: (id: number, patch: CustomerColorPatch) =>
      write(() => http.put(`/customers/${id}/colors`, patch), {
        apply: () => {
          const view: Partial<SidebarCustomer> = {};
          if (patch.color !== undefined) view.color = patch.color;
          if (patch.everywhere !== undefined) view.color_everywhere = patch.everywhere;
          return assign(findCustomer(id), view);
        },
        invalidate: false
      }),

    /** Background of a customer (also behind its projects unless they have their own). */
    setCustomerBackground: (id: number, background: string) =>
      write(() => http.put(`/customers/${id}/background`, { background }), { apply: () => assign(findCustomer(id), { background }), invalidate: false }),

    deleteCustomer: (id: number) =>
      write(() => http.del(`/customers/${id}`), {
        after: () => {
          state.customers = state.customers.filter((c) => c.id !== id);
        }
      }),

    // ---- Tags and items --------------------------------------------------------------
    deleteTag: (id: number) =>
      write(() => http.del(`/tags/${id}`), {
        after: () => {
          state.tags = state.tags.filter((x) => x.id !== id);
        }
      }),

    /** Check off an item (My tasks); progress in the tree follows with the debounced reload. */
    setItemDone: (itemId: number, done: boolean) => write(() => http.post(`/items/${itemId}/done`, { done }), { invalidate: 400 })
  };

  return { state, refresh, invalidate, batch, findProject, findChecklist, findCustomer, ...ops };
}

export type Workspace = ReturnType<typeof createWorkspace>;

// ---- App singleton ---------------------------------------------------------------------

/** The workspace of the signed-in user; synced prefs take over their values on every load. */
export const workspace = createWorkspace({
  transport: api,
  onLoaded: (r) => syncPrefs({ prefs: r.prefs ?? {}, board_hidden: r.board_hidden ?? [] })
});

/** The read model (/api/sidebar). Read it; change it through the operations of `workspace`. */
export const sidebar = workspace.state;

/** Reloads the read model (debounced). Use after login or when a page wants fresh data. */
export const refreshSidebar = (delay = 400) => workspace.refresh(delay);
