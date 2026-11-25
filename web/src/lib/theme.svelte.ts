// Color scheme: light, dark or like the operating system ("system", also
// without a stored choice; the old value "dim" = dark). `dark` is the
// effective result. Stored per browser and synced via the pref "ui".
import { parseTheme, type ThemeChoice } from './appearance';
import { uiHooks } from './state.svelte';

export type Theme = ThemeChoice;

function readTheme(): Theme {
  try {
    return parseTheme(localStorage.getItem('theme'));
  } catch {
    return 'system';
  }
}
const osDark = () => typeof matchMedia === 'function' && matchMedia('(prefers-color-scheme: dark)').matches;
const initialTheme = readTheme();

export const theme = $state<{ value: Theme; dark: boolean }>({ value: initialTheme, dark: initialTheme === 'system' ? osDark() : initialTheme === 'dark' });

function applyTheme() {
  theme.dark = theme.value === 'system' ? osDark() : theme.value === 'dark';
  if (typeof document !== 'undefined') document.documentElement.classList.toggle('dark', theme.dark);
}
// follow OS changes while "system" is selected
if (typeof matchMedia === 'function') {
  matchMedia('(prefers-color-scheme: dark)').addEventListener?.('change', () => {
    if (theme.value === 'system') applyTheme();
  });
}

export function setTheme(value: Theme, sync = true) {
  theme.value = value;
  try {
    localStorage.setItem('theme', value);
  } catch {
    /* ignore */
  }
  applyTheme();
  if (sync) uiHooks.changed();
}
