// Template list (names, item counts) for places outside the templates page:
// favorites, breadcrumb and history of the back bar. Loaded on demand; loaded
// again when a template id is not in the list (e.g. created meanwhile).
import { api } from './api';
import type { TemplateSummary } from './types';

export const templatesIndex = $state<{ list: TemplateSummary[] | null }>({ list: null });
let loading = false;
let loadedAt = 0; // deleted templates stay missing: reload at most every 10 s

/** Makes sure the given templates are known (no-op if they already are). */
export function ensureTemplates(ids: number[]) {
  if (!ids.length || loading) return;
  if (templatesIndex.list && ids.every((id) => templatesIndex.list!.some((x) => x.id === id))) return;
  if (templatesIndex.list && Date.now() - loadedAt < 10_000) return;
  loading = true;
  api.get<TemplateSummary[]>('/templates').then(
    (list) => ((templatesIndex.list = list), (loadedAt = Date.now()), (loading = false)),
    () => (loading = false)
  );
}
