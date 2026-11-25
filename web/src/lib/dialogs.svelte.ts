// Global dialogs opened from anywhere (rendered once in the layout).
import type { ProjectStatus } from './types';

/** Command palette (Ctrl+K). */
export const palette = $state({ open: false });

/** "New project" dialog (sidebar, overview); null = closed. */
export const newProject = $state<{ value: { customerId: number | null; status: ProjectStatus } | null }>({ value: null });
export function openNewProject(customerId: number | null = null, status: ProjectStatus = 'planning') {
  newProject.value = { customerId, status };
}
