import type { ProjectStatus, SidebarCustomer, Tag, User } from './types';
import { applyFonts, DEFAULT_FONT, FONT_IDS, type FontId, type HeadingFont } from './fonts';
import { CUSTOMER_SORTS, type CustomerSort } from './customers';

// Global state (Svelte 5 runes).

export const session = $state<{ user: User | null; ready: boolean; subtitle: string }>({ user: null, ready: false, subtitle: '' });

export const sidebar = $state<{
  customers: SidebarCustomer[];
  myTasks: number;
  favorites: number[]; // project IDs (DB, per user)
  tags: Tag[]; // tag catalog (global)
  statusLabels: Partial<Record<ProjectStatus, string>>; // custom status names (global, set by admins)
  statusColors: Partial<Record<ProjectStatus, string>>; // custom status colors #rrggbb (global, set by admins)
  statusWip: Partial<Record<ProjectStatus, number>>; // WIP limit per board column (global, set by admins)
  prefs: { overview?: unknown; saved_views?: unknown; background?: unknown }; // synced personal settings (DB, per user)
  loaded: boolean; // first /sidebar response arrived
  boardHidden: number[]; // customers hidden on the board (DB, per user, synced across devices)
  open: boolean;
  collapsed: boolean;
}>({
  customers: [],
  myTasks: 0,
  favorites: [],
  tags: [],
  statusLabels: {},
  statusColors: {},
  statusWip: {},
  prefs: {},
  loaded: false,
  boardHidden: [],
  open: false, // mobile: shown
  collapsed: readCollapsed() // desktop: collapsed (remembered per browser)
});

function readCollapsed(): boolean {
  try {
    return localStorage.getItem('sidebar-collapsed') === '1';
  } catch {
    return false;
  }
}

export function toggleSidebar(collapsed = !sidebar.collapsed) {
  sidebar.collapsed = collapsed;
  try {
    if (collapsed) localStorage.setItem('sidebar-collapsed', '1');
    else localStorage.removeItem('sidebar-collapsed');
  } catch {
    /* ignore */
  }
}

// Sidebar width (desktop), adjustable by dragging the right edge.
export const SIDEBAR_MIN = 200;
export const SIDEBAR_MAX = 480;
export const SIDEBAR_DEFAULT = 240;

export const sidebarWidth = $state({ value: readSidebarWidth() });

function readSidebarWidth(): number {
  try {
    const n = Number(localStorage.getItem('sidebar-width'));
    return n >= SIDEBAR_MIN && n <= SIDEBAR_MAX ? n : SIDEBAR_DEFAULT;
  } catch {
    return SIDEBAR_DEFAULT;
  }
}

export function setSidebarWidth(px: number, save = true) {
  sidebarWidth.value = Math.round(Math.min(SIDEBAR_MAX, Math.max(SIDEBAR_MIN, px)));
  if (!save) return;
  try {
    if (sidebarWidth.value === SIDEBAR_DEFAULT) localStorage.removeItem('sidebar-width');
    else localStorage.setItem('sidebar-width', String(sidebarWidth.value));
  } catch {
    /* ignore */
  }
}

export const palette = $state({ open: false });

/** "New project" dialog (sidebar, overview); null = closed. */
export const newProject = $state<{ value: { customerId: number | null; status: ProjectStatus } | null }>({ value: null });
export function openNewProject(customerId: number | null = null, status: ProjectStatus = 'planning') {
  newProject.value = { customerId, status };
}

/** Save status for the "saved" indicator. */
export const saveState = $state({ pending: 0, saved: false, error: false });

/** Sections of the currently open checklist (for the sidebar). */
export interface Section {
  id: number;
  title: string;
  done: number;
  total: number;
}
export const current = $state<{ checklistId: number | null; projectId: number | null; customerId: number | null; sections: Section[] }>({
  checklistId: null,
  projectId: null,
  customerId: null,
  sections: []
});

export function canWrite() {
  return session.user?.role === 'admin' || session.user?.role === 'member';
}

export function isAdmin() {
  return session.user?.role === 'admin';
}

let refreshTimer: ReturnType<typeof setTimeout> | undefined;

/** Reloads the sidebar (debounced so rapid click sequences trigger only one request). */
export function refreshSidebar(delay = 400) {
  clearTimeout(refreshTimer);
  refreshTimer = setTimeout(async () => {
    const { api } = await import('./api');
    try {
      const r = await api.get<{
        customers: SidebarCustomer[];
        my_tasks: number;
        favorites: number[];
        tags: Tag[];
        status_labels: Partial<Record<ProjectStatus, string>>;
        status_colors?: Partial<Record<ProjectStatus, string>>;
        status_wip?: Partial<Record<ProjectStatus, number>>;
        prefs?: { overview?: unknown; saved_views?: unknown; background?: unknown };
        board_hidden: number[];
      }>('/sidebar');
      sidebar.customers = r.customers;
      sidebar.myTasks = r.my_tasks;
      sidebar.favorites = r.favorites ?? [];
      sidebar.tags = r.tags ?? [];
      sidebar.statusLabels = r.status_labels ?? {};
      sidebar.statusColors = r.status_colors ?? {};
      sidebar.statusWip = r.status_wip ?? {};
      sidebar.prefs = r.prefs ?? {};
      sidebar.loaded = true;
      // do not overwrite a filter change that is still being saved
      if (!boardHiddenTimer) sidebar.boardHidden = r.board_hidden ?? [];
    } catch {
      /* errors become visible on the next load */
    }
  }, delay);
}

// Board customer filter: show immediately, save debounced (rapid clicks = one request).
let boardHiddenTimer: ReturnType<typeof setTimeout> | undefined;
export function setBoardHidden(ids: number[]) {
  sidebar.boardHidden = ids;
  clearTimeout(boardHiddenTimer);
  boardHiddenTimer = setTimeout(async () => {
    const { api, errorText } = await import('./api');
    try {
      await api.put('/prefs/board-hidden', { customer_ids: sidebar.boardHidden });
    } catch (e) {
      toast(errorText(e));
    } finally {
      boardHiddenTimer = undefined;
    }
  }, 400);
}

// ---- Theme ----------------------------------------------------------------------

export type Theme = 'system' | 'light' | 'dark' | 'dim';

function readTheme(): Theme {
  try {
    const t = localStorage.getItem('theme');
    return t === 'light' || t === 'dark' || t === 'dim' ? t : 'system';
  } catch {
    return 'system';
  }
}

export const theme = $state<{ value: Theme }>({ value: readTheme() });

export function setTheme(value: Theme) {
  theme.value = value;
  try {
    if (value === 'system') localStorage.removeItem('theme');
    else localStorage.setItem('theme', value);
  } catch {
    /* ignore */
  }
  const dark = value === 'dark' || value === 'dim' || (value === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', dark);
  document.documentElement.classList.toggle('dim', value === 'dim');
}

// ---- Checklist display ----------------------------------------------------------
// Personal view settings (per browser, like the theme).

export type CheckPos = 'left' | 'right';
export type CheckShape = 'square' | 'round';
export type CodeBg = 'light' | 'dark' | 'black';
export const CODE_BGS: readonly CodeBg[] = ['light', 'dark', 'black'];
export type OverviewView = 'cards' | 'board' | 'list' | 'calendar' | 'timeline';
export const OVERVIEW_VIEWS: readonly OverviewView[] = ['cards', 'board', 'list', 'calendar', 'timeline'];
// 80 % to 160 % in 5 % steps (rounded so 1.05 etc. compare exactly)
export const ZOOM_STEPS: readonly number[] = Array.from({ length: 17 }, (_, i) => Math.round(80 + i * 5) / 100);

interface ViewPrefs {
  checkPos: CheckPos;
  checkShape: CheckShape;
  gridLines: boolean; // border lines around each row (like in Excel)
  zoom: number;
  codeBg: CodeBg; // background of code blocks
  overview: OverviewView; // last used overview view
  overviewDefault: OverviewView | 'last'; // view at startup (settings)
  customersView: 'list' | 'cards'; // customer overview
  customersSort: CustomerSort;
  font: FontId; // app font
  headingFont: HeadingFont; // headings: serif or same as the text
}

const PREFS_KEY = 'view-prefs';
const defaultPrefs: ViewPrefs = { checkPos: 'right', checkShape: 'square', gridLines: true, zoom: 1, codeBg: 'dark', overview: 'cards', overviewDefault: 'last', customersView: 'list', customersSort: 'activity', font: DEFAULT_FONT, headingFont: 'serif' };

function readPrefs(): ViewPrefs {
  try {
    const p = JSON.parse(localStorage.getItem(PREFS_KEY) ?? '{}');
    return {
      checkPos: p.checkPos === 'left' ? 'left' : 'right',
      checkShape: p.checkShape === 'round' ? 'round' : 'square',
      gridLines: p.gridLines !== false,
      zoom: ZOOM_STEPS.includes(p.zoom) ? p.zoom : 1,
      codeBg: CODE_BGS.includes(p.codeBg) ? p.codeBg : 'dark',
      overview: OVERVIEW_VIEWS.includes(p.overview) ? p.overview : 'cards',
      overviewDefault: OVERVIEW_VIEWS.includes(p.overviewDefault) ? p.overviewDefault : 'last',
      customersView: p.customersView === 'cards' ? 'cards' : 'list',
      customersSort: CUSTOMER_SORTS.includes(p.customersSort) ? p.customersSort : 'activity',
      font: FONT_IDS.includes(p.font) ? p.font : DEFAULT_FONT,
      headingFont: p.headingFont === 'sans' ? 'sans' : 'serif'
    };
  } catch {
    return { ...defaultPrefs };
  }
}

export const prefs = $state<ViewPrefs>(readPrefs());

export function setPref<K extends keyof ViewPrefs>(key: K, value: ViewPrefs[K]) {
  prefs[key] = value;
  if (key === 'font' || key === 'headingFont') applyFonts(prefs.font, prefs.headingFont);
  try {
    localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore */
  }
}

/** Change the checklist size by one step (dir = +1 / -1); 0 resets to 100 %. */
export function stepZoom(dir: -1 | 0 | 1) {
  const i = ZOOM_STEPS.indexOf(prefs.zoom);
  const next = dir === 0 ? 1 : ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, (i < 0 ? ZOOM_STEPS.indexOf(1) : i) + dir))];
  setPref('zoom', next);
}

/** Classes for the checkbox shape. */
export const checkRadius = () => (prefs.checkShape === 'round' ? 'rounded-full' : 'rounded-[4px]');

// ---- Toasts -------------------------------------------------------------------

export interface Toast {
  id: number;
  text: string;
  kind: 'error' | 'info';
}

export const toasts = $state<{ list: Toast[] }>({ list: [] });
let toastId = 0;

export function toast(text: string, kind: Toast['kind'] = 'error') {
  const id = ++toastId;
  toasts.list.push({ id, text, kind });
  setTimeout(() => {
    toasts.list = toasts.list.filter((t) => t.id !== id);
  }, 4500);
}

// ---- Overview view --------------------------------------------------------------
// During the session (tab) the last selected view is kept
// (sessionStorage). On the next start the default view from the
// settings applies – or, with "Last used", the most recently used one.

function startView(): OverviewView {
  try {
    const s = sessionStorage.getItem('overview-view') as OverviewView | null;
    if (s && OVERVIEW_VIEWS.includes(s)) return s;
  } catch {
    /* ignore */
  }
  return prefs.overviewDefault === 'last' ? prefs.overview : prefs.overviewDefault;
}

export const overviewView = $state<{ value: OverviewView }>({ value: startView() });

export function setOverviewView(v: OverviewView) {
  overviewView.value = v;
  try {
    sessionStorage.setItem('overview-view', v);
  } catch {
    /* ignore */
  }
  setPref('overview', v); // remember "last used"
}

/** Change the default view (settings) – applies immediately, not only on the next start. */
export function setOverviewDefault(v: OverviewView | 'last') {
  setPref('overviewDefault', v);
  try {
    sessionStorage.removeItem('overview-view');
  } catch {
    /* ignore */
  }
  overviewView.value = startView();
}
