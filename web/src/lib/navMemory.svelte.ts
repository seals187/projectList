// Sidebar: favorites (projects) and recently visited pages.
// Stored in the database per user (API), loaded with /api/sidebar
// (sidebar.favorites / sidebar.recent). Changes are shown immediately
// and saved in the background.
import { api, errorText } from './api';
import { refreshSidebar, session, sidebar, toast } from './state.svelte';
import type { RecentEntry } from './types';

export type RecentKind = RecentEntry['kind'];

export const isFavorite = (projectId: number) => sidebar.favorites.includes(projectId);

export async function toggleFavorite(projectId: number) {
  const was = isFavorite(projectId);
  const prev = sidebar.favorites;
  sidebar.favorites = was ? prev.filter((id) => id !== projectId) : [...prev, projectId];
  try {
    if (was) await api.del(`/favorites/${projectId}`);
    else await api.put(`/favorites/${projectId}`, null);
  } catch (e) {
    sidebar.favorites = prev;
    toast(errorText(e));
  }
}

let lastPosted = ''; // guard: don't send the same visit repeatedly (e.g. after reload)

/** Put the visited page at the front and remember it on the server. */
export function rememberVisit(kind: RecentKind, id: number) {
  const idx = sidebar.recent.findIndex((r) => r.kind === kind && r.id === id);
  if (idx === 0) return; // already at the front
  const key = `${kind}:${id}`;
  if (key === lastPosted) return;
  lastPosted = key;
  const post = api.post('/recent', { kind, id });
  if (idx > 0) {
    // known: move to the front immediately
    const item = sidebar.recent[idx];
    sidebar.recent = [item, ...sidebar.recent.filter((_, i) => i !== idx)];
    post.catch(() => {});
  } else {
    // new: the server knows the title → reload the sidebar afterwards
    post.then(() => refreshSidebar(0), () => {});
  }
}

/** Clear "Recently visited". */
export async function clearRecent() {
  lastPosted = '';
  const prev = sidebar.recent;
  sidebar.recent = [];
  try {
    await api.del('/recent');
  } catch (e) {
    sidebar.recent = prev;
    toast(errorText(e));
  }
}

/**
 * One-time migration: transfer favorites that an earlier version stored
 * in the browser (localStorage) to the server.
 */
export async function migrateLocalFavorites() {
  const key = `favorites:${session.user?.id ?? 0}`;
  let ids: unknown;
  try {
    ids = JSON.parse(localStorage.getItem(key) ?? 'null');
    localStorage.removeItem(key);
    localStorage.removeItem(`recent:${session.user?.id ?? 0}`);
  } catch {
    return;
  }
  if (!Array.isArray(ids)) return;
  for (const id of ids) {
    if (Number.isInteger(id) && !isFavorite(id)) {
      try {
        await api.put(`/favorites/${id}`, null);
        sidebar.favorites = [...sidebar.favorites, id];
      } catch {
        /* project deleted or no longer accessible */
      }
    }
  }
}
