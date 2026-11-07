// Texts of the template library that need the UI language.
import { t } from './i18n.svelte';

/** "1 item" / "n items". */
export const itemsText = (n: number) => (n === 1 ? t('templates.item1') : t('templates.items', { n }));
