// Appearance settings (Settings → Appearance): accent color, glass effect,
// darkened background image, density and app font size. Pure helpers and
// constants (testable); the values live in `prefs` (state.svelte.ts) and are
// synced across devices via the pref "ui" (uiSync.svelte.ts).

export type Accent = 'violet' | 'blue' | 'teal' | 'green' | 'orange' | 'red' | 'pink';
export const ACCENTS: readonly Accent[] = ['violet', 'blue', 'teal', 'green', 'orange', 'red', 'pink'];
/** Swatch per accent (= primary button color, see app.css). */
export const ACCENT_SWATCH: Record<Accent, string> = {
  violet: '#5b3df5',
  blue: '#2563eb',
  teal: '#0d9488',
  green: '#16a34a',
  orange: '#d97706',
  red: '#dc2626',
  pink: '#db2777'
};

export type Density = 'compact' | 'normal' | 'airy';
export const DENSITIES: readonly Density[] = ['compact', 'normal', 'airy'];

// darken the main window (all pages next to the sidebar): strength of a dark veil
export type DimContent = 'off' | 'light' | 'medium' | 'strong';
export const DIM_CONTENT: readonly DimContent[] = ['off', 'light', 'medium', 'strong'];
export const DIM_ALPHA: Record<DimContent, number> = { off: 0, light: 0.07, medium: 0.14, strong: 0.24 };

// app font size: 80 % to 120 % in 5 % steps (rounded so values compare exactly)
export const SCALE_STEPS: readonly number[] = Array.from({ length: 9 }, (_, i) => Math.round(80 + i * 5) / 100);

export type ThemeChoice = 'light' | 'dark' | 'system';
export const THEME_CHOICES: readonly ThemeChoice[] = ['light', 'dark', 'system'];

/** Theme from storage: "dim" (old dark gray) = dark; unknown or missing = system. */
export function parseTheme(v: unknown): ThemeChoice {
  if (v === 'light') return 'light';
  if (v === 'dark' || v === 'dim') return 'dark';
  return 'system';
}

/** One step bigger/smaller (dir ±1), 0 = 100 %. */
export function stepScale(current: number, dir: -1 | 0 | 1): number {
  if (dir === 0) return 1;
  const i = SCALE_STEPS.indexOf(current);
  const base = i < 0 ? SCALE_STEPS.indexOf(1) : i;
  return SCALE_STEPS[Math.min(SCALE_STEPS.length - 1, Math.max(0, base + dir))];
}

/** Settings that follow the user to all devices (pref "ui"). */
export const UI_PREF_KEYS = [
  'accent',
  'glass',
  'dimBg',
  'dimContent',
  'density',
  'scale',
  'font',
  'headingFont',
  'checkPos',
  'checkShape',
  'gridLines',
  'codeBg',
  'overviewDefault'
] as const;
export type UiPrefKey = (typeof UI_PREF_KEYS)[number];

export interface UiPayload {
  theme?: ThemeChoice;
  lang?: string;
  prefs?: Record<string, unknown>;
}

/** Reads the stored "ui" pref; anything malformed yields an empty payload. */
export function parseUi(v: unknown): UiPayload {
  if (!v || typeof v !== 'object' || Array.isArray(v)) return {};
  const o = v as Record<string, unknown>;
  const out: UiPayload = {};
  if (typeof o.theme === 'string') out.theme = parseTheme(o.theme);
  if (typeof o.lang === 'string') out.lang = o.lang;
  if (o.prefs && typeof o.prefs === 'object' && !Array.isArray(o.prefs)) {
    const p = o.prefs as Record<string, unknown>;
    out.prefs = Object.fromEntries(UI_PREF_KEYS.filter((k) => k in p).map((k) => [k, p[k]]));
  }
  return out;
}

/** Validates the appearance values of a prefs object (missing/invalid → default). */
export function readAppearance(p: Record<string, unknown>) {
  return {
    accent: ACCENTS.includes(p.accent as Accent) ? (p.accent as Accent) : 'violet',
    glass: p.glass !== false,
    dimBg: p.dimBg === true,
    dimContent: DIM_CONTENT.includes(p.dimContent as DimContent) ? (p.dimContent as DimContent) : 'off',
    density: DENSITIES.includes(p.density as Density) ? (p.density as Density) : 'normal',
    scale: SCALE_STEPS.includes(p.scale as number) ? (p.scale as number) : 1
  };
}

/** Applies accent, density, font size and glass to <html> (also done by theme.js before the first render). */
export function applyAppearance(a: { accent: Accent; glass: boolean; density: Density; scale: number }) {
  if (typeof document === 'undefined') return;
  const el = document.documentElement;
  if (a.accent === 'violet') delete el.dataset.accent;
  else el.dataset.accent = a.accent;
  if (a.density === 'normal') delete el.dataset.density;
  else el.dataset.density = a.density;
  el.style.fontSize = a.scale === 1 ? '' : `${a.scale * 100}%`;
  el.classList.toggle('no-glass', !a.glass);
}
