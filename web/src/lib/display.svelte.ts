// Display options of the page headers per user and page/view: saved on the
// server (pref "display"), localStorage as fallback (offline, before the sidebar
// data has arrived, or when saving fails).
import { api } from './api';
import { session, sidebar } from './state.svelte';
import { parseStore, resolveDisplay, withOption, type DisplayStore, type DisplayValues } from './displayCore';
import { t } from './i18n.svelte';

type TKey = Parameters<typeof t>[0];

const localKey = () => `display:${session.user?.id ?? 0}`;

function readLocal(): DisplayStore {
  try {
    return parseStore(localStorage.getItem(localKey()));
  } catch {
    return {};
  }
}
function writeLocal(s: DisplayStore) {
  try {
    localStorage.setItem(localKey(), JSON.stringify(s));
  } catch {
    /* ignore */
  }
}

export const displayStore = $state<{ all: DisplayStore; loaded: boolean }>({ all: {}, loaded: false });

let saveTimer: ReturnType<typeof setTimeout> | undefined;

/** Take over the server value when the sidebar data arrives (unless a change is pending). */
export function syncDisplay() {
  if (saveTimer) return;
  const server = parseStore(sidebar.prefs.display);
  displayStore.all = Object.keys(server).length ? server : readLocal();
  displayStore.loaded = true;
}

/** Current options of a page/view (defaults where nothing is stored). */
export function displayOf(page: string, defaults: DisplayValues): DisplayValues {
  const all = displayStore.loaded ? displayStore.all : readLocal(); // read only (may run inside $derived)
  return resolveDisplay(defaults, all[page]);
}

/** Change one option: shown at once, localStorage immediately, server debounced. */
export function setDisplay(page: string, key: string, value: boolean) {
  displayStore.all = withOption(displayStore.all, page, key, value);
  writeLocal(displayStore.all);
  clearTimeout(saveTimer);
  saveTimer = setTimeout(async () => {
    try {
      const v = $state.snapshot(displayStore.all);
      await api.put('/prefs/display', v);
      sidebar.prefs.display = v;
    } catch {
      /* stays in localStorage */
    } finally {
      saveTimer = undefined;
    }
  }, 400);
}

/** Label of an option ("disp.<key>" in the translations). */
export const displayLabel = (key: string) => t(`disp.${key}` as TKey);
