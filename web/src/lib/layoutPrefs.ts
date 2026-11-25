// Own order and folders of sidebar sections (prefs "fav_layout" and
// "views_layout"), synced per user; saved immediately, errors as a toast.
import { parseLayout, type FavLayout } from './favLayout';
import { syncedPref, type SyncedPref } from './syncedPref.svelte';

export type LayoutPrefKey = 'fav_layout' | 'views_layout';

export const layoutPrefs: Record<LayoutPrefKey, SyncedPref<FavLayout>> = {
  fav_layout: syncedPref('fav_layout', { parse: parseLayout, debounce: 0 }),
  views_layout: syncedPref('views_layout', { parse: parseLayout, debounce: 0 })
};
