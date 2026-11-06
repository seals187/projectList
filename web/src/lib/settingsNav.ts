// Navigation of the settings (SettingsShell): two groups, "Personal" for every
// user and "Workspace" for admins. The search matches the label and the
// keywords of an entry (texts of the rows on its page), case- and accent-insensitive.

export interface SettingsNavItem {
  id: string;
  href: string; // path, optionally with #anchor
  icon: 'user' | 'palette' | 'type' | 'clipboard' | 'home' | 'tag' | 'users' | 'database' | 'info';
  label: string; // i18n key
  keys: string[]; // i18n keys of further search terms
  admin?: boolean;
  bottom?: boolean; // shown at the very bottom (About)
}

export const SETTINGS_NAV: readonly SettingsNavItem[] = [
  { id: 'profile', href: '/settings/profile', icon: 'user', label: 'set.nav.profile', keys: ['common.name', 'common.email', 'settings.password', 'nav.logout'] },
  { id: 'appearance', href: '/settings/appearance', icon: 'palette', label: 'set.nav.appearance', keys: ['settings.language', 'set.colorScheme', 'set.accent', 'bg.title', 'set.glass', 'set.dimBg', 'set.density', 'set.rowSpacing', 'set.fontSize'] },
  { id: 'font', href: '/settings/font', icon: 'type', label: 'set.nav.font', keys: ['set.fontFamily', 'settings.headings'] },
  { id: 'checklists', href: '/settings/font#checklists', icon: 'clipboard', label: 'set.nav.checklists', keys: ['set.checkbox', 'set.dividers', 'set.codeBlock'] },
  { id: 'overview', href: '/settings/overview', icon: 'home', label: 'set.nav.overview', keys: ['settings.defaultView'] },
  { id: 'tags', href: '/settings/tags', icon: 'tag', label: 'set.nav.tags', keys: ['tags.title', 'tags.color'], admin: true },
  { id: 'users', href: '/settings/users', icon: 'users', label: 'set.nav.users', keys: ['common.role', 'set.users.invite', 'set.users.access', 'set.users.resetPassword'], admin: true },
  { id: 'backup', href: '/settings/backup', icon: 'database', label: 'set.nav.backup', keys: ['settings.backupDownload', 'set.restore.title'], admin: true },
  { id: 'about', href: '/settings/about', icon: 'info', label: 'set.nav.about', keys: ['set.about.version', 'set.about.source'], bottom: true }
];

const norm = (s: string) =>
  s
    .toLocaleLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '');

/** Entries visible for the user that match the query (empty query = all). */
export function filterNav(items: readonly SettingsNavItem[], query: string, isAdmin: boolean, tr: (key: string) => string): SettingsNavItem[] {
  const q = norm(query.trim());
  return items.filter((it) => (isAdmin || !it.admin) && (!q || [it.label, ...it.keys].some((k) => norm(tr(k)).includes(q))));
}

/** Active entry for a path and hash (an entry with #anchor only wins on that anchor). */
export function activeNav(items: readonly SettingsNavItem[], path: string, hash: string): string {
  const withHash = items.find((it) => it.href === path + hash && hash);
  if (withHash) return withHash.id;
  return items.find((it) => it.href === path)?.id ?? '';
}
