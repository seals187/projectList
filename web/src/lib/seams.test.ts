// Guards the workspace seam: writes that change the sidebar data go through a
// named workspace operation (optimistic update, rollback, reload). A full
// refreshSidebar() is only for loading a page or a new session.
import { describe, expect, it } from 'vitest';

const sources = import.meta.glob('/src/**/*.{ts,svelte}', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

/** Files allowed to call refreshSidebar(): session start and pages that reload on open. */
const ALLOWED = new Set([
  '/src/lib/workspace.svelte.ts',
  '/src/routes/+layout.svelte',
  '/src/routes/+page.svelte',
  '/src/routes/customers/+page.svelte',
  '/src/routes/invite/+page.svelte',
  '/src/routes/login/+page.svelte',
  '/src/routes/tasks/+page.svelte'
]);

describe('workspace seam', () => {
  it('refreshSidebar() is called only on page open and session start', () => {
    const offenders = Object.entries(sources)
      .filter(([path, src]) => !path.endsWith('.test.ts') && !ALLOWED.has(path) && /\brefreshSidebar\(/.test(src))
      .map(([path]) => path);
    expect(offenders, 'use a workspace.* operation instead (web/src/lib/workspace.svelte.ts)').toEqual([]);
  });
});
