// Sidebar favorites (customers, projects, checklists): stored in the database
// per user (API), loaded with /api/sidebar (sidebar.favorites, in the order
// they were added). Changes are shown immediately and saved in the background.
import { api, errorText } from './api';
import { session, sidebar, toast } from './state.svelte';
import type { Favorite, FavoriteKind } from './types';

export const isFavorite = (kind: FavoriteKind, id: number) => sidebar.favorites.some((f) => f.kind === kind && f.id === id);

export async function toggleFavorite(kind: FavoriteKind, id: number) {
  const was = isFavorite(kind, id);
  const prev = sidebar.favorites;
  sidebar.favorites = was ? prev.filter((f) => !(f.kind === kind && f.id === id)) : [...prev, { kind, id } satisfies Favorite];
  try {
    if (was) await api.del(`/favorites/${kind}/${id}`);
    else await api.put(`/favorites/${kind}/${id}`, null);
  } catch (e) {
    sidebar.favorites = prev;
    toast(errorText(e));
  }
}

/**
 * One-time migration: transfer project favorites that an earlier version
 * stored in the browser (localStorage) to the server.
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
    if (Number.isInteger(id) && !isFavorite('project', id)) {
      try {
        await api.put(`/favorites/project/${id}`, null);
        sidebar.favorites = [...sidebar.favorites, { kind: 'project', id }];
      } catch {
        /* project deleted or no longer accessible */
      }
    }
  }
}
