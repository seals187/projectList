// Project changes from anywhere in the UI (context menu, tag picker):
// update the sidebar store immediately, revert on error.
import { api, errorText } from './api';
import { refreshSidebar, sidebar, toast } from './state.svelte';
import type { Assignee, Project, SidebarProject } from './types';

/** The project in the sidebar store (source of the overview), if loaded. */
export const storeProject = (id: number): SidebarProject | undefined => sidebar.customers.flatMap((c) => c.projects).find((p) => p.id === id);

type ProjectPatch = Partial<Pick<Project, 'name' | 'status' | 'start_date' | 'due_date' | 'owner_id'>>;

/** PATCH /projects/{id}; returns the saved project or null on error (with toast). */
export async function patchProject(id: number, body: ProjectPatch, ownerName?: string): Promise<Project | null> {
  const p = storeProject(id);
  const prev = p ? { ...p } : undefined;
  if (p) {
    Object.assign(p, body);
    if ('owner_id' in body) p.owner_name = body.owner_id ? (ownerName ?? '') : '';
  }
  try {
    const saved = await api.patch<Project>(`/projects/${id}`, body);
    refreshSidebar(0);
    return saved;
  } catch (e) {
    if (p && prev) Object.assign(p, prev);
    toast(errorText(e));
    return null;
  }
}

/** PUT /projects/{id}/tags; returns the saved tag IDs or null on error (with toast). */
export async function setProjectTags(id: number, tagIds: number[]): Promise<number[] | null> {
  const p = storeProject(id);
  const prev = p?.tag_ids;
  if (p) p.tag_ids = tagIds;
  try {
    const r = await api.put<{ tag_ids: number[] }>(`/projects/${id}/tags`, { tag_ids: tagIds });
    if (p) p.tag_ids = r.tag_ids;
    refreshSidebar(0);
    return r.tag_ids;
  } catch (e) {
    if (p && prev) p.tag_ids = prev;
    toast(errorText(e));
    return null;
  }
}

// People per customer who can be made responsible (cached for the session).
const assigneeCache = new Map<number, Promise<Assignee[]>>();
export function customerAssignees(customerId: number): Promise<Assignee[]> {
  let p = assigneeCache.get(customerId);
  if (!p) {
    p = api.get<Assignee[]>(`/customers/${customerId}/assignees`);
    p.catch(() => assigneeCache.delete(customerId));
    assigneeCache.set(customerId, p);
  }
  return p;
}

export interface CustomerColorPatch {
  color?: string;
  everywhere?: boolean;
}

/** Shared customer color (only the given fields; optimistic). */
export async function setCustomerColors(id: number, patch: CustomerColorPatch) {
  const c = sidebar.customers.find((x) => x.id === id);
  const prev = c ? { color: c.color, color_everywhere: c.color_everywhere } : null;
  if (c) {
    if (patch.color !== undefined) c.color = patch.color;
    if (patch.everywhere !== undefined) c.color_everywhere = patch.everywhere;
  }
  try {
    await api.put(`/customers/${id}/colors`, patch);
    return true;
  } catch (e) {
    if (c && prev) Object.assign(c, prev);
    toast(errorText(e));
    return false;
  }
}
