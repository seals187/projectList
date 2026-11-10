// Texts of the template library that need the UI language.
import { t } from './i18n.svelte';

/** "1 item" / "n items". */
export const itemsText = (n: number) => (n === 1 ? t('templates.item1') : t('templates.items', { n }));

/** Size of a template in lists: its items, or "Table" for a table template. */
export const templateSizeText = (tpl: { kind?: string; item_count: number }) => (tpl.kind === 'table' ? t('templates.kind.table') : itemsText(tpl.item_count));

/** "1 sheet" / "n sheets". */
export const sheetsText = (n: number) => (n === 1 ? t('table.oneSheet') : t('table.nSheets', { n }));
