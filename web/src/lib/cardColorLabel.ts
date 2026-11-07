// Display name of a card color ("Blau", "Blau · kräftig").
import { t } from './i18n.svelte';
import { parseCardColor } from './cardColors';

export function cardColorLabel(value: string): string {
  const c = parseCardColor(value);
  if (!c) return t('color.none');
  return t(`color.${c.base}`) + (c.strong ? ` · ${t('color.strong')}` : '');
}
