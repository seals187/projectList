// Navigation history of this tab (back bar): updated after every navigation,
// kept in sessionStorage so a reload keeps it.
import { applyNav, clearHistory, type Hist } from './navHistory';

const KEY = 'nav-history';

function load(): Hist | null {
  try {
    const h = JSON.parse(sessionStorage.getItem(KEY) ?? 'null');
    return h && Array.isArray(h.entries) && Number.isInteger(h.index) ? h : null;
  } catch {
    return null;
  }
}

export const navHist = $state<{ h: Hist }>({ h: { entries: [], index: 0 } });
let restored: Hist | null = typeof sessionStorage === 'undefined' ? null : load();

/** Called by the layout after each navigation (SvelteKit afterNavigate). */
export function recordNav(type: string, delta: number | undefined, url: URL) {
  const base = restored ?? navHist.h;
  restored = null;
  navHist.h = applyNav(base.entries.length ? base : null, { type, delta, path: url.pathname, search: url.search });
  try {
    sessionStorage.setItem(KEY, JSON.stringify(navHist.h));
  } catch {
    /* ignore */
  }
}

/** Clears the history of this tab (the browser's own history stays as it is). */
export function clearNav() {
  navHist.h = clearHistory(navHist.h);
  try {
    sessionStorage.setItem(KEY, JSON.stringify(navHist.h));
  } catch {
    /* ignore */
  }
}
