import { describe, expect, it } from 'vitest';
import { parseTheme, parseUi, readAppearance, stepScale } from './appearance';
import { activeNav, filterNav, SETTINGS_NAV } from './settingsNav';
import { relTime } from './relTime';

describe('appearance', () => {
  it('theme: old "dim" = dark, unknown/missing = system', () => {
    expect(parseTheme('dim')).toBe('dark');
    expect(parseTheme('light')).toBe('light');
    expect(parseTheme(null)).toBe('system');
    expect(parseTheme('foo')).toBe('system');
  });

  it('defaults and validation of the appearance prefs', () => {
    expect(readAppearance({})).toEqual({ accent: 'violet', glass: true, dimBg: false, dimContent: 'off', density: 'normal', scale: 1 });
    expect(readAppearance({ accent: 'teal', glass: false, dimBg: true, dimContent: 'medium', density: 'airy', scale: 0.9 })).toEqual({ accent: 'teal', glass: false, dimBg: true, dimContent: 'medium', density: 'airy', scale: 0.9 });
    expect(readAppearance({ dimContent: 'extreme' }).dimContent).toBe('off');
    expect(readAppearance({ accent: 'gold', density: 'x', scale: 3 })).toMatchObject({ accent: 'violet', density: 'normal', scale: 1 });
  });

  it('font size steps stay within 80–120 %', () => {
    expect(stepScale(1, -1)).toBe(0.95);
    expect(stepScale(0.8, -1)).toBe(0.8);
    expect(stepScale(1.2, 1)).toBe(1.2);
    expect(stepScale(1.1, 0)).toBe(1);
  });

  it('synced "ui" pref: only known keys, malformed input ignored', () => {
    expect(parseUi(null)).toEqual({});
    expect(parseUi([1])).toEqual({});
    expect(parseUi({ theme: 'dim', lang: 'fr', prefs: { accent: 'red', dimContent: 'strong', zoom: 2, noteWidth: 300 } })).toEqual({ theme: 'dark', lang: 'fr', prefs: { accent: 'red', dimContent: 'strong' } });
  });
});

describe('settings navigation', () => {
  const tr = (k: string) => ({ 'set.nav.appearance': 'Darstellung', 'set.accent': 'Akzentfarbe', 'set.nav.users': 'Benutzer & Rechte', 'settings.password': 'Passwort ändern' })[k] ?? k;

  it('workspace entries only for admins', () => {
    expect(filterNav(SETTINGS_NAV, '', false, tr).some((i) => i.admin)).toBe(false);
    expect(filterNav(SETTINGS_NAV, '', true, tr)).toHaveLength(SETTINGS_NAV.length);
  });

  it('search matches labels and keywords, ignoring case and accents', () => {
    expect(filterNav(SETTINGS_NAV, 'akzent', true, tr).map((i) => i.id)).toEqual(['appearance']);
    expect(filterNav(SETTINGS_NAV, 'PASSWORT', true, tr).map((i) => i.id)).toEqual(['profile']);
    expect(filterNav(SETTINGS_NAV, 'andern', true, tr).map((i) => i.id)).toEqual(['profile']);
    expect(filterNav(SETTINGS_NAV, 'benutzer', false, tr)).toEqual([]);
  });

  it('active entry: an #anchor entry only wins on its anchor', () => {
    expect(activeNav(SETTINGS_NAV, '/settings/font', '')).toBe('font');
    expect(activeNav(SETTINGS_NAV, '/settings/font', '#checklists')).toBe('checklists');
    expect(activeNav(SETTINGS_NAV, '/settings/font', '#other')).toBe('font');
    expect(activeNav(SETTINGS_NAV, '/settings/tags', '#other')).toBe('tags');
  });
});

describe('relTime', () => {
  const now = new Date('2026-11-06T12:00:00Z');
  it('null below the "just now" limit, otherwise relative', () => {
    expect(relTime('2026-11-06T11:59:30Z', now, 'de-DE')).toBeNull();
    expect(relTime('2026-11-06T11:57:00Z', now, 'de-DE', 5)).toBeNull();
    expect(relTime('2026-11-06T11:58:00Z', now, 'de-DE')).toBe('vor 2 Min.');
    expect(relTime('2026-11-03T12:00:00Z', now, 'de-DE')).toBe('vor 3 Tagen');
    expect(relTime('2026-11-03T12:00:00Z', now, 'en-GB')).toBe('3 days ago');
  });
});
