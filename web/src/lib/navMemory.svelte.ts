// Sidebar favorites (projects): stored in the database per user (API),
// loaded with /api/sidebar (sidebar.favorites). Changes are shown
// immediately and saved in the background.
import { api, errorText } from './api';
import { session, sidebar, toast } from './state.svelte';

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
