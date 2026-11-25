// Overview filters (customer, label, person responsible), board grouping and
// compact cards, plus saved views. Everything is stored per user in the
// database and therefore the same on all devices. The customer filter keeps its
// own key (pref "board_hidden", PUT /prefs/board-hidden) so existing settings stay valid.
import { goto } from '$app/navigation';
import { OVERVIEW_VIEWS, setOverviewView, type OverviewView } from './state.svelte';
import { sidebar } from './workspace.svelte';
import { syncedPref } from './syncedPref.svelte';
import { BOARD_GROUPS, nextHidden, NO_OWNER, type BoardGroup, type CustomerFilterMode } from './projectQuery';
import { normQuery } from './pageStateCore';

export { BOARD_GROUPS, NO_OWNER, type BoardGroup };

interface Filter {
  tags: number[]; // projects with at least one of these labels (empty = all)
  owners: number[]; // person responsible (NO_OWNER = nobody; empty = all)
  group: BoardGroup;
  compact: boolean; // board: compact cards
}

/** Pages whose header state can be saved as a view. */
export type ViewPage = 'overview' | 'tasks' | 'customers' | 'templates' | 'reports' | 'archive';
export const VIEW_PAGES: Record<ViewPage, string> = { overview: '/', tasks: '/tasks', customers: '/customers', templates: '/templates', reports: '/reports', archive: '/archive' };

export interface SavedView extends Omit<Filter, 'compact'> {
  id: string;
  name: string;
  page: ViewPage; // overview: filters below; other pages: their URL query
  query: string; // URL query of the page header (without "?")
  view: OverviewView;
  hidden: number[]; // hidden customers
}

const ids = (v: unknown): number[] => (Array.isArray(v) ? v.filter((x) => Number.isInteger(x)) : []);

function parseFilter(v: unknown): Filter {
  const o = (v && typeof v === 'object' ? v : {}) as Record<string, unknown>;
  return {
    tags: ids(o.tags),
    owners: ids(o.owners),
    group: BOARD_GROUPS.includes(o.group as BoardGroup) ? (o.group as BoardGroup) : 'status',
    compact: o.compact === true
  };
}

function parseViews(v: unknown): SavedView[] {
  if (!Array.isArray(v)) return [];
  return v
    .filter((x) => x && typeof x === 'object' && typeof x.id === 'string' && typeof x.name === 'string')
    .map((x) => ({
      ...parseFilter(x),
      id: x.id,
      name: x.name,
      page: x.page in VIEW_PAGES ? (x.page as ViewPage) : 'overview',
      query: typeof x.query === 'string' ? x.query : '',
      view: OVERVIEW_VIEWS.includes(x.view) ? x.view : 'board',
      hidden: ids(x.hidden)
    }));
}

// Synced prefs (see syncedPref.svelte.ts): filters debounced, saved views
// immediately with rollback, hidden customers via their own endpoint.
const filterPref = syncedPref('overview', { parse: parseFilter, debounce: 400 });
const viewsPref = syncedPref('saved_views', { parse: parseViews, debounce: 0, rollback: true });
/** Customers hidden in the overview (all views); key kept from the board ("board-hidden"). */
export const boardHidden = syncedPref<number[]>('board_hidden', {
  parse: ids,
  read: (s) => s.board_hidden,
  url: '/prefs/board-hidden',
  body: (v) => ({ customer_ids: v }),
  debounce: 400
});

/** Current filters (reactive, read only; change them with setFilter). */
export const filter: Readonly<Filter> = {
  get tags() {
    return filterPref.value.tags;
  },
  get owners() {
    return filterPref.value.owners;
  },
  get group() {
    return filterPref.value.group;
  },
  get compact() {
    return filterPref.value.compact;
  }
};
export const savedViews: { readonly list: SavedView[] } = {
  get list() {
    return viewsPref.value;
  }
};

/** Customer filter: shown immediately, saved debounced (rapid clicks = one request). */
export function setBoardHidden(list: number[]) {
  boardHidden.set(list);
}

/** Change filters; shown immediately, saved debounced. */
export function setFilter(patch: Partial<Filter>) {
  filterPref.set((f) => ({ ...f, ...patch }));
}

const saveViews = (list: SavedView[]) => viewsPref.set(list);

const newId = () => Math.random().toString(36).slice(2, 10);

/** Saves the header state of another page (its URL query) under a name. */
export function addPageView(name: string, page: Exclude<ViewPage, 'overview'>, query: string) {
  const v: SavedView = { id: newId(), name: name.trim().slice(0, 60), page, query, view: 'cards', hidden: [], tags: [], owners: [], group: 'status' };
  return saveViews([...savedViews.list, v]);
}

/** Saves the current overview view and filters under a name (query: sorting/search). */
export function addSavedView(name: string, view: OverviewView, query = '') {
  const v: SavedView = {
    id: newId(),
    name: name.trim().slice(0, 60),
    page: 'overview',
    query,
    view,
    hidden: [...boardHidden.value],
    tags: [...filter.tags],
    owners: [...filter.owners],
    group: filter.group
  };
  return saveViews([...savedViews.list, v]);
}

export const removeSavedView = (id: string) => saveViews(savedViews.list.filter((v) => v.id !== id));

/** Renames a saved view (empty names are ignored). */
export function renameSavedView(id: string, name: string) {
  const n = name.trim().slice(0, 60);
  const v = savedViews.list.find((x) => x.id === id);
  if (!n || !v || v.name === n) return;
  return saveViews(savedViews.list.map((x) => (x.id === id ? { ...x, name: n } : x)));
}

/** Applies a saved view (view, filters) and opens the overview. */
export function applySavedView(v: SavedView) {
  if (v.page !== 'overview') {
    goto(VIEW_PAGES[v.page] + (v.query ? `?${v.query}` : ''));
    return;
  }
  setBoardHidden(v.hidden);
  setFilter({ tags: v.tags, owners: v.owners, group: v.group });
  setOverviewView(v.view);
  // the overview reads its filters from the URL as well (see routes/+page.svelte)
  const q = new URLSearchParams(v.query);
  if (v.view !== 'cards') q.set('view', v.view);
  if (v.tags.length) q.set('tags', v.tags.join(','));
  if (v.owners.length) q.set('owners', v.owners.join(','));
  if (v.hidden.length) q.set('hide', v.hidden.join(','));
  if (v.group !== 'status') q.set('group', v.group);
  q.sort();
  goto('/' + (q.toString() ? `?${q}` : ''));
}

/** Whether a saved view of another page matches the current page and URL query. */
export function isActivePageView(v: SavedView, path: string, search: string): boolean {
  return v.page !== 'overview' && VIEW_PAGES[v.page] === path && normQuery(v.query) === normQuery(search);
}

/** Resets customer, label and owner filters to "All". */
export function resetFilters() {
  setBoardHidden([]);
  setFilter({ tags: [], owners: [] });
}

/** Whether the current filters equal a saved view (highlight in the sidebar). */
export function isActiveView(v: SavedView, view: OverviewView): boolean {
  const same = (a: number[], b: number[]) => a.length === b.length && a.every((x) => b.includes(x));
  return v.page === 'overview' && v.view === view && v.group === filter.group && same(v.hidden, boardHidden.value) && same(v.tags, filter.tags) && same(v.owners, filter.owners);
}

/** Customer filter from a customer menu (all overview views): only / toggle / all. */
export function applyCustomerFilter(id: number, mode: CustomerFilterMode) {
  setBoardHidden(nextHidden(boardHidden.value, sidebar.customers.map((c) => c.id), id, mode));
}

/** Number of active filters (customer, label, person responsible). */
export const activeFilters = () => (boardHidden.value.length ? 1 : 0) + (filter.tags.length ? 1 : 0) + (filter.owners.length ? 1 : 0);
