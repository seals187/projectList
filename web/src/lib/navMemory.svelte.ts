// Seitenleiste: Favoriten (Projekte) und zuletzt besuchte Seiten.
// Gespeichert in der Datenbank pro Benutzer (API), geladen mit /api/sidebar
// (sidebar.favorites / sidebar.recent). Änderungen werden sofort angezeigt
// und im Hintergrund gespeichert.
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

let lastPosted = ''; // Schutz: denselben Besuch nicht wiederholt senden (z. B. nach Neuladen)

/** Besuchte Seite vorne einreihen und auf dem Server merken. */
export function rememberVisit(kind: RecentKind, id: number) {
  const idx = sidebar.recent.findIndex((r) => r.kind === kind && r.id === id);
  if (idx === 0) return; // schon vorne
  const key = `${kind}:${id}`;
  if (key === lastPosted) return;
  lastPosted = key;
  const post = api.post('/recent', { kind, id });
  if (idx > 0) {
    // bekannt: sofort nach vorne schieben
    const item = sidebar.recent[idx];
    sidebar.recent = [item, ...sidebar.recent.filter((_, i) => i !== idx)];
    post.catch(() => {});
  } else {
    // neu: Titel kennt der Server → danach Seitenleiste neu laden
    post.then(() => refreshSidebar(0), () => {});
  }
}

/** „Zuletzt besucht“ leeren. */
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
 * Einmalige Übernahme: Favoriten, die eine frühere Version im Browser
 * (localStorage) gespeichert hat, auf den Server übertragen.
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
        /* Projekt gelöscht oder kein Zugriff mehr */
      }
    }
  }
}
