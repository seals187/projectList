// Project changes from anywhere in the UI (context menu, tag picker): the
// workspace store shows them at once and reverts on error; these wrappers add
// the error toast and return null on failure.
import { api, errorText } from './api';
import { toast } from './toasts.svelte';
import { workspace, type CustomerColorPatch, type ProjectPatch } from './workspace.svelte';
import type { Assignee, Project, SidebarProject } from './types';

export type { CustomerColorPatch, ProjectPatch };

/** The project in the sidebar store (source of the overview), if loaded. */
export const storeProject = (id: number): SidebarProject | undefined => workspace.findProject(id);

const orNull = <T>(p: Promise<T>): Promise<T | null> =>
  p.catch((e) => {
    toast(errorText(e));
    return null;
  });

/** PATCH /projects/{id}; returns the saved project or null on error (with toast). */
export const patchProject = (id: number, body: ProjectPatch, ownerName?: string): Promise<Project | null> => orNull(workspace.updateProject(id, body, ownerName));

/** PUT /projects/{id}/tags; returns the saved tag IDs or null on error (with toast). */
export const setProjectTags = (id: number, tagIds: number[]): Promise<number[] | null> => orNull(workspace.setProjectTags(id, tagIds));

/** Shared customer color (only the given fields; optimistic). */
export const setCustomerColors = (id: number, patch: CustomerColorPatch): Promise<boolean> =>
  workspace.setCustomerColors(id, patch).then(
    () => true,
    (e) => {
      toast(errorText(e));
      return false;
    }
  );

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
