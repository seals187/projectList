// Overview filters (customer, label, person responsible), board grouping and
// compact cards, plus saved views. Everything is stored per user in the
// database and therefore the same on all devices. The customer filter keeps its
// own key (sidebar.boardHidden) so existing settings stay valid.
import { goto } from '$app/navigation';
import { api, errorText } from './api';
import { OVERVIEW_VIEWS, setBoardHidden, setOverviewView, sidebar, toast, type OverviewView } from './state.svelte';
import type { SidebarProject } from './types';
import { normQuery } from './pageStateCore';
import { syncDisplay } from './display.svelte';
import { syncUi } from './uiSync.svelte';
import { syncMyColors } from './myColors.svelte';

export type BoardGroup = 'status' | 'customer' | 'owner';
export const BOARD_GROUPS: readonly BoardGroup[] = ['status', 'customer', 'owner'];

/** Person responsible "nobody" in the owner filter. */
export const NO_OWNER = 0;

interface Filter {
  tags: number[]; // projects with at least one of these labels (empty = all)
  owners: number[]; // person responsible (NO_OWNER = nobody; empty = all)
  group: BoardGroup;
  compact: boolean; // board: compact cards
}

/** Pages whose header state can be saved as a view. */
export type ViewPage = 'overview' | 'tasks' | 'customers' | 'templates' | 'reports';
export const VIEW_PAGES: Record<ViewPage, string> = { overview: '/', tasks: '/tasks', customers: '/customers', templates: '/templates', reports: '/reports' };

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

export const filter = $state<Filter>(parseFilter(null));
export const savedViews = $state<{ list: SavedView[] }>({ list: [] });

// Take over the stored values whenever the sidebar data arrives, unless a
// local change is still being saved.
let filterTimer: ReturnType<typeof setTimeout> | undefined;
let viewsSaving = false;
export function syncFromServer() {
  if (!filterTimer) Object.assign(filter, parseFilter(sidebar.prefs.overview));
  if (!viewsSaving) savedViews.list = parseViews(sidebar.prefs.saved_views);
  syncDisplay();
  syncMyColors();
  syncUi();
}

/** Change filters; shown immediately, saved debounced. */
export function setFilter(patch: Partial<Filter>) {
  Object.assign(filter, patch);
  clearTimeout(filterTimer);
  filterTimer = setTimeout(async () => {
    try {
      const value = { ...filter };
      await api.put('/prefs/overview', value);
      sidebar.prefs.overview = value;
    } catch (e) {
      toast(errorText(e));
    } finally {
      filterTimer = undefined;
    }
  }, 400);
}

async function saveViews(list: SavedView[]) {
  const prev = savedViews.list;
  savedViews.list = list;
  viewsSaving = true;
  try {
    await api.put('/prefs/saved_views', list);
    sidebar.prefs.saved_views = list;
  } catch (e) {
    savedViews.list = prev;
    toast(errorText(e));
  } finally {
    viewsSaving = false;
  }
}

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
    hidden: [...sidebar.boardHidden],
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
  return v.page === 'overview' && v.view === view && v.group === filter.group && same(v.hidden, sidebar.boardHidden) && same(v.tags, filter.tags) && same(v.owners, filter.owners);
}

/** Whether a project passes the label and owner filters (customer filter is separate). */
export function matchesFilter(p: SidebarProject): boolean {
  if (filter.tags.length && !p.tag_ids.some((id) => filter.tags.includes(id))) return false;
  if (filter.owners.length && !filter.owners.includes(p.owner_id ?? NO_OWNER)) return false;
  return true;
}

/** Number of active filters (customer, label, person responsible). */
export const activeFilters = () => (sidebar.boardHidden.length ? 1 : 0) + (filter.tags.length ? 1 : 0) + (filter.owners.length ? 1 : 0);
