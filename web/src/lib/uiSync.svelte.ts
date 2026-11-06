// Appearance settings follow the user to all devices: theme, language and the
// appearance prefs are saved as the server pref "ui" (debounced) and taken over
// whenever the sidebar data arrives (unless a local change is still pending).
// localStorage stays the source before the first server response.
import { api } from './api';
import { parseUi, UI_PREF_KEYS } from './appearance';
import { i18n, LANGS, setLang } from './i18n.svelte';
import { applyUiPrefs, prefs, setTheme, sidebar, theme, uiHooks } from './state.svelte';

let timer: ReturnType<typeof setTimeout> | undefined;

/** Current values as stored on the server. */
export function uiSnapshot() {
  const p = $state.snapshot(prefs) as unknown as Record<string, unknown>;
  return { theme: theme.value, lang: i18n.lang, prefs: Object.fromEntries(UI_PREF_KEYS.map((k) => [k, p[k]])) };
}

/** Save soon (called after every change of a synced setting). */
export function pushUi() {
  clearTimeout(timer);
  timer = setTimeout(async () => {
    const value = uiSnapshot();
    try {
      await api.put('/prefs/ui', value);
      sidebar.prefs.ui = value;
    } catch {
      /* stays local; next change tries again */
    } finally {
      timer = undefined;
    }
  }, 500);
}
uiHooks.changed = pushUi;

/** Take over the server value (called when the sidebar data arrives). */
export function syncUi() {
  if (timer) return;
  const ui = parseUi(sidebar.prefs.ui);
  if (ui.theme && ui.theme !== theme.value) setTheme(ui.theme, false);
  if (ui.lang && ui.lang !== i18n.lang && LANGS.some((l) => l.code === ui.lang)) setLang(ui.lang as Parameters<typeof setLang>[0]);
  if (ui.prefs) applyUiPrefs(ui.prefs);
}
