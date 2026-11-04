import type { ProjectStatus, RecentItem, SidebarCustomer, User } from './types';
import { applyFonts, DEFAULT_FONT, FONT_IDS, type FontId, type HeadingFont } from './fonts';
import { CUSTOMER_SORTS, type CustomerSort } from './customers';

// Globale Zustände (Svelte 5 Runes).

export const session = $state<{ user: User | null; ready: boolean; subtitle: string }>({ user: null, ready: false, subtitle: '' });

export const sidebar = $state<{
  customers: SidebarCustomer[];
  myTasks: number;
  favorites: number[]; // Projekt-IDs (DB, pro Benutzer)
  recent: RecentItem[]; // zuletzt besucht, neuester zuerst (DB, pro Benutzer)
  open: boolean;
  collapsed: boolean;
}>({
  customers: [],
  myTasks: 0,
  favorites: [],
  recent: [],
  open: false, // mobil: eingeblendet
  collapsed: readCollapsed() // Desktop: eingeklappt (pro Browser gemerkt)
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
    /* ignorieren */
  }
}

// Breite der Seitenleiste (Desktop), per Ziehen am rechten Rand einstellbar.
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
    /* ignorieren */
  }
}

export const palette = $state({ open: false });

/** Dialog „Neues Projekt“ (Seitenleiste, Übersicht); null = geschlossen. */
export const newProject = $state<{ value: { customerId: number | null; status: ProjectStatus } | null }>({ value: null });
export function openNewProject(customerId: number | null = null, status: ProjectStatus = 'planning') {
  newProject.value = { customerId, status };
}

/** Speicherstatus für die "saved"-Anzeige. */
export const saveState = $state({ pending: 0, saved: false, error: false });

/** Abschnitte der gerade geöffneten Checkliste (für die Sidebar). */
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

/** Lädt die Sidebar neu (entprellt, damit schnelle Klickfolgen nur eine Anfrage auslösen). */
export function refreshSidebar(delay = 400) {
  clearTimeout(refreshTimer);
  refreshTimer = setTimeout(async () => {
    const { api } = await import('./api');
    try {
      const r = await api.get<{ customers: SidebarCustomer[]; my_tasks: number; favorites: number[]; recent: RecentItem[] }>('/sidebar');
      sidebar.customers = r.customers;
      sidebar.myTasks = r.my_tasks;
      sidebar.favorites = r.favorites ?? [];
      sidebar.recent = r.recent ?? [];
    } catch {
      /* Fehler werden beim nächsten Laden sichtbar */
    }
  }, delay);
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
    /* ignorieren */
  }
  const dark = value === 'dark' || value === 'dim' || (value === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', dark);
  document.documentElement.classList.toggle('dim', value === 'dim');
}

// ---- Darstellung der Checklisten ------------------------------------------------
// Persönliche Ansichtseinstellungen (pro Browser, wie das Theme).

export type CheckPos = 'left' | 'right';
export type CheckShape = 'square' | 'round';
export type CodeBg = 'light' | 'dark' | 'black';
export const CODE_BGS: readonly CodeBg[] = ['light', 'dark', 'black'];
export type OverviewView = 'cards' | 'list' | 'board';
export const OVERVIEW_VIEWS: readonly OverviewView[] = ['cards', 'list', 'board'];
// 80 % bis 160 % in 5 %-Schritten (gerundet, damit 1.05 usw. exakt vergleichbar sind)
export const ZOOM_STEPS: readonly number[] = Array.from({ length: 17 }, (_, i) => Math.round(80 + i * 5) / 100);

interface ViewPrefs {
  checkPos: CheckPos;
  checkShape: CheckShape;
  gridLines: boolean; // Rahmenlinien um jede Zeile (wie in Excel)
  zoom: number;
  codeBg: CodeBg; // Hintergrund der Codeblöcke
  overview: OverviewView; // zuletzt genutzte Ansicht der Übersicht
  overviewDefault: OverviewView | 'last'; // Ansicht beim Start (Einstellungen)
  customersView: 'list' | 'cards'; // Kundenübersicht
  customersSort: CustomerSort;
  font: FontId; // Schrift der App
  headingFont: HeadingFont; // Überschriften: Serif oder wie der Text
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
    /* ignorieren */
  }
}

/** Größe der Checkliste um eine Stufe ändern (dir = +1 / -1); 0 setzt auf 100 % zurück. */
export function stepZoom(dir: -1 | 0 | 1) {
  const i = ZOOM_STEPS.indexOf(prefs.zoom);
  const next = dir === 0 ? 1 : ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, (i < 0 ? ZOOM_STEPS.indexOf(1) : i) + dir))];
  setPref('zoom', next);
}

/** Klassen für die Form der Checkbox. */
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

// ---- Ansicht der Übersicht ------------------------------------------------------
// Während der Sitzung (Tab) bleibt die zuletzt gewählte Ansicht erhalten
// (sessionStorage). Beim nächsten Start gilt die Standardansicht aus den
// Einstellungen – oder bei „Zuletzt genutzte“ die zuletzt verwendete.

function startView(): OverviewView {
  try {
    const s = sessionStorage.getItem('overview-view') as OverviewView | null;
    if (s && OVERVIEW_VIEWS.includes(s)) return s;
  } catch {
    /* ignorieren */
  }
  return prefs.overviewDefault === 'last' ? prefs.overview : prefs.overviewDefault;
}

export const overviewView = $state<{ value: OverviewView }>({ value: startView() });

export function setOverviewView(v: OverviewView) {
  overviewView.value = v;
  try {
    sessionStorage.setItem('overview-view', v);
  } catch {
    /* ignorieren */
  }
  setPref('overview', v); // „zuletzt genutzt“ merken
}

/** Standardansicht ändern (Einstellungen) – gilt sofort, nicht erst beim nächsten Start. */
export function setOverviewDefault(v: OverviewView | 'last') {
  setPref('overviewDefault', v);
  try {
    sessionStorage.removeItem('overview-view');
  } catch {
    /* ignorieren */
  }
  overviewView.value = startView();
}
