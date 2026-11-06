// Sidebar favorites (customers, projects, checklists, templates) and their
// groups: stored in the database per user (API), loaded with /api/sidebar
// (sidebar.favorites, sidebar.favoriteGroups). Changes are shown immediately
// and saved in the background; on errors the previous state comes back.
import { api, errorText } from './api';
import { refreshSidebar, session, sidebar, toast } from './state.svelte';
import type { Favorite, FavoriteGroup, FavoriteKind } from './types';
import { favoritesLayout, layoutToOrder } from './favoritesModel';
import type { FavLayout } from './favLayout';

export const isFavorite = (kind: FavoriteKind, id: number) => sidebar.favorites.some((f) => f.kind === kind && f.id === id);

async function guarded(fn: () => Promise<unknown>, rollback: () => void) {
  try {
    await fn();
  } catch (e) {
    rollback();
    toast(errorText(e));
  }
}

export async function toggleFavorite(kind: FavoriteKind, id: number) {
  const was = isFavorite(kind, id);
  const prev = sidebar.favorites;
  const last = Math.max(-1, ...prev.filter((f) => f.group_id === null).map((f) => f.position));
  sidebar.favorites = was ? prev.filter((f) => !(f.kind === kind && f.id === id)) : [...prev, { kind, id, group_id: null, position: last + 1 } satisfies Favorite];
  await guarded(
    () => (was ? api.del(`/favorites/${kind}/${id}`) : api.put(`/favorites/${kind}/${id}`, null)),
    () => (sidebar.favorites = prev)
  );
}

/** Saves a new order (layout of groups and favorites, see favoritesModel). */
export async function saveFavoritesOrder(layout: FavLayout) {
  const prevF = sidebar.favorites;
  const prevG = sidebar.favoriteGroups;
  const order = layoutToOrder(layout);
  // optimistic: positions and groups as in the new layout
  const groupPos = new Map(order.groups.map((id, i) => [id, i]));
  sidebar.favoriteGroups = prevG.map((g) => ({ ...g, position: groupPos.get(g.id) ?? g.position }));
  const counters = new Map<number | null, number>();
  const placed = new Map<string, { group_id: number | null; position: number }>();
  for (const it of order.items) {
    const n = counters.get(it.group_id) ?? 0;
    counters.set(it.group_id, n + 1);
    placed.set(`${it.kind}:${it.id}`, { group_id: it.group_id, position: n });
  }
  sidebar.favorites = prevF.map((f) => ({ ...f, ...(placed.get(`${f.kind}:${f.id}`) ?? {}) }));
  await guarded(
    () => api.put('/favorites/order', order),
    () => ((sidebar.favorites = prevF), (sidebar.favoriteGroups = prevG))
  );
}

/** Creates a group; optionally moves a favorite into it. */
export async function createFavoriteGroup(name: string, move?: { kind: FavoriteKind; id: number }) {
  try {
    const g = await api.post<FavoriteGroup>('/favorite-groups', { name });
    sidebar.favoriteGroups = [...sidebar.favoriteGroups, g];
    if (move) await moveFavoriteToGroup(move.kind, move.id, g.id);
    return g;
  } catch (e) {
    toast(errorText(e));
    return null;
  }
}

export async function updateFavoriteGroup(id: number, patch: Partial<Pick<FavoriteGroup, 'name' | 'collapsed'>>) {
  const prev = sidebar.favoriteGroups;
  sidebar.favoriteGroups = prev.map((g) => (g.id === id ? { ...g, ...patch } : g));
  await guarded(
    () => api.patch(`/favorite-groups/${id}`, patch),
    () => (sidebar.favoriteGroups = prev)
  );
}

/** Deletes a group; its favorites move to "without group" (end of the list). */
export async function deleteFavoriteGroup(id: number) {
  const prevF = sidebar.favorites;
  const prevG = sidebar.favoriteGroups;
  const last = Math.max(-1, ...prevF.filter((f) => f.group_id === null).map((f) => f.position));
  let n = last;
  sidebar.favoriteGroups = prevG.filter((g) => g.id !== id);
  sidebar.favorites = prevF
    .slice()
    .sort((a, b) => a.position - b.position)
    .map((f) => (f.group_id === id ? { ...f, group_id: null, position: ++n } : f));
  await guarded(
    () => api.del(`/favorite-groups/${id}`),
    () => ((sidebar.favorites = prevF), (sidebar.favoriteGroups = prevG))
  );
}

/** Moves a favorite to the end of a group (null = without group). */
export function moveFavoriteToGroup(kind: FavoriteKind, id: number, groupId: number | null) {
  const layout = favoritesLayout(sidebar.favorites, sidebar.favoriteGroups);
  const key = `${kind}:${id}`;
  const next = groupId === null ? moveToRoot(layout, key) : moveInto(layout, key, `g${groupId}`);
  return saveFavoritesOrder(next);
}

function moveInto(layout: FavLayout, key: string, folder: string): FavLayout {
  const rest = layout
    .filter((n) => !(n.t === 'i' && n.k === key))
    .map((n) => (n.t === 'f' ? { ...n, items: n.items.filter((k) => k !== key) } : n));
  return rest.map((n) => (n.t === 'f' && n.id === folder ? { ...n, open: true, items: [...n.items, key] } : n));
}

function moveToRoot(layout: FavLayout, key: string): FavLayout {
  const rest = layout
    .filter((n) => !(n.t === 'i' && n.k === key))
    .map((n) => (n.t === 'f' ? { ...n, items: n.items.filter((k) => k !== key) } : n));
  const firstFolder = rest.findIndex((n) => n.t === 'f');
  const at = firstFolder < 0 ? rest.length : firstFolder;
  return [...rest.slice(0, at), { t: 'i', k: key }, ...rest.slice(at)];
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
  let added = false;
  for (const id of ids) {
    if (Number.isInteger(id) && !isFavorite('project', id)) {
      try {
        await api.put(`/favorites/project/${id}`, null);
        added = true;
      } catch {
        /* project deleted or no longer accessible */
      }
    }
  }
  if (added) refreshSidebar(0);
}
