// Display options of the page headers per user and page/view: saved on the
// server (pref "display"), localStorage as fallback (offline, before the sidebar
// data has arrived, or when saving fails).
import { session } from './session.svelte';
import { sidebar } from './workspace.svelte';
import { parseStore, resolveDisplay, withOption, type DisplayStore, type DisplayValues } from './displayCore';
import { t } from './i18n.svelte';
import { syncedPref } from './syncedPref.svelte';

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

// server value, or localStorage when the server has nothing; failures stay local
const pref = syncedPref<DisplayStore>('display', {
  parse: (raw) => {
    const server = parseStore(raw);
    return Object.keys(server).length ? server : readLocal();
  },
  debounce: 400,
  onError: 'silent',
  onChange: (v) => writeLocal($state.snapshot(v) as DisplayStore)
});

/** All stored options; `loaded` = a server value has been taken over. */
export const displayStore = {
  get all() {
    return pref.value;
  },
  set all(v: DisplayStore) {
    pref.value = v;
  },
  get loaded() {
    return pref.synced;
  },
  set loaded(v: boolean) {
    pref.synced = v;
  }
};

/** Take over the server value from the sidebar data (unless a change is pending). */
export function syncDisplay() {
  pref.sync(sidebar);
}

/** Current options of a page/view (defaults where nothing is stored). */
export function displayOf(page: string, defaults: DisplayValues): DisplayValues {
  const all = pref.synced ? pref.value : readLocal(); // read only (may run inside $derived)
  return resolveDisplay(defaults, all[page]);
}

/** Change one option: shown at once, localStorage immediately, server debounced. */
export function setDisplay(page: string, key: string, value: boolean) {
  pref.set((all) => withOption(all, page, key, value));
}

/** Label of an option ("disp.<key>" in the translations). */
export const displayLabel = (key: string) => t(`disp.${key}` as TKey);
