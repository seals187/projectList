// UI state of this browser (Svelte 5 runes): sidebar chrome (collapsed,
// width, mobile drawer), the open checklist's sections, personal view prefs
// and the overview view. Other global state has its own module: session
// (session.svelte), read model + writes (workspace.svelte), theme
// (theme.svelte), toasts (toasts.svelte), dialogs (dialogs.svelte), save
// indicator (saveState.svelte), zoom steps (zoom.ts).
import { applyFonts, DEFAULT_FONT, FONT_IDS, type FontId, type HeadingFont } from './fonts';
import { CUSTOMER_SORTS, type CustomerSort } from './customers';
import { parsePageZoom, type PageZoom } from './pageZoom';
import { applyAppearance, readAppearance, UI_PREF_KEYS, type Accent, type Density, type DimContent, type UiPrefKey } from './appearance';

/** Sidebar chrome: mobile drawer shown, desktop collapsed (remembered per browser). */
export const sidebarUi = $state<{ open: boolean; collapsed: boolean }>({
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

export function toggleSidebar(collapsed = !sidebarUi.collapsed) {
  sidebarUi.collapsed = collapsed;
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

/** Hooks for settings that sync across devices (set by uiSync.svelte.ts). */
export const uiHooks: { changed: () => void } = { changed: () => {} };

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
  noteWidth: number; // checklist note column in px, 0 = default (dragged at its left edge)
  accent: Accent; // accent color (buttons, selection, progress)
  glass: boolean; // sidebar and header translucent over a background image
  dimBg: boolean; // darken the background image
  dimContent: DimContent; // darken the main window (veil over all pages)
  density: Density; // row spacing of lists and checklists
  scale: number; // font size of the whole app (1 = 100 %)
  pageZoom: PageZoom; // content size of the main pages (70–110 %, see pageZoom.ts)
}

// width range of the checklist note column (px, before zoom)
export const NOTE_MIN = 160;
export const NOTE_MAX = 900;

const PREFS_KEY = 'view-prefs';
const defaultPrefs: ViewPrefs = { checkPos: 'right', checkShape: 'square', gridLines: true, zoom: 1, codeBg: 'dark', overview: 'cards', overviewDefault: 'last', customersView: 'list', customersSort: 'activity', font: DEFAULT_FONT, headingFont: 'serif', noteWidth: 0, pageZoom: {}, ...readAppearance({}) };

function readPrefs(): ViewPrefs {
  try {
    return readPrefsFrom(JSON.parse(localStorage.getItem(PREFS_KEY) ?? '{}'));
  } catch {
    return { ...defaultPrefs };
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function readPrefsFrom(p: any): ViewPrefs {
  if (!p || typeof p !== 'object') return { ...defaultPrefs };
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
      headingFont: p.headingFont === 'sans' ? 'sans' : 'serif',
      noteWidth: Number.isFinite(p.noteWidth) && p.noteWidth >= NOTE_MIN && p.noteWidth <= NOTE_MAX ? Math.round(p.noteWidth) : 0,
      pageZoom: parsePageZoom(p.pageZoom),
      ...readAppearance(p)
  };
}

export const prefs = $state<ViewPrefs>(readPrefs());

export function setPref<K extends keyof ViewPrefs>(key: K, value: ViewPrefs[K], sync = true) {
  prefs[key] = value;
  if (key === 'font' || key === 'headingFont') applyFonts(prefs.font, prefs.headingFont);
  if (key === 'accent' || key === 'glass' || key === 'density' || key === 'scale') applyAppearance(prefs);
  try {
    localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore */
  }
  if (sync && (UI_PREF_KEYS as readonly string[]).includes(key)) uiHooks.changed();
}

/** Takes over synced values from another device (validated, no re-sync). */
export function applyUiPrefs(values: Partial<Record<UiPrefKey, unknown>>) {
  const merged = { ...$state.snapshot(prefs), ...values } as Record<string, unknown>;
  const check = readPrefsFrom(merged);
  for (const k of UI_PREF_KEYS) if (k in values) (prefs as unknown as Record<string, unknown>)[k] = check[k];
  applyFonts(prefs.font, prefs.headingFont);
  applyAppearance(prefs);
  try {
    localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore */
  }
}

/** Classes for the checkbox shape. */
export const checkRadius = () => (prefs.checkShape === 'round' ? 'rounded-full' : 'rounded-[4px]');

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

