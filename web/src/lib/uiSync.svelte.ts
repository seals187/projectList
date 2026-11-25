// Appearance settings follow the user to all devices: theme, language and the
// appearance prefs are saved as the server pref "ui" (debounced) and taken over
// whenever the sidebar data arrives (unless a local change is still pending).
// localStorage stays the source before the first server response.
import { parseUi, UI_PREF_KEYS, type UiPayload } from './appearance';
import { i18n, LANGS, setLang } from './i18n.svelte';
import { applyUiPrefs, prefs, uiHooks } from './state.svelte';
import { setTheme, theme } from './theme.svelte';
import { syncedPref } from './syncedPref.svelte';

/** Current values as stored on the server. */
export function uiSnapshot() {
  const p = $state.snapshot(prefs) as unknown as Record<string, unknown>;
  return { theme: theme.value, lang: i18n.lang, prefs: Object.fromEntries(UI_PREF_KEYS.map((k) => [k, p[k]])) };
}

/** Applies a server value to theme, language and appearance prefs. */
function applyUi(ui: UiPayload) {
  if (ui.theme && ui.theme !== theme.value) setTheme(ui.theme, false);
  if (ui.lang && ui.lang !== i18n.lang && LANGS.some((l) => l.code === ui.lang)) setLang(ui.lang as Parameters<typeof setLang>[0]);
  if (ui.prefs) applyUiPrefs(ui.prefs);
}

// failures stay local; the next change tries again
const pref = syncedPref<UiPayload>('ui', { parse: parseUi, debounce: 500, onError: 'silent', onSync: applyUi });

/** Save soon (called after every change of a synced setting). */
export function pushUi() {
  pref.set(uiSnapshot() as UiPayload);
}
uiHooks.changed = pushUi;
