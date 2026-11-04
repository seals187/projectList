// Selectable fonts (settings). All self-hosted via @fontsource
// (Open Font License, no external requests); only the font currently in use
// is loaded. The mapping id → font-family is in app.css (html[data-font=…]).
import '@fontsource-variable/inter/opsz.css'; // with optical size (large titles like "Inter Display")
import '@fontsource-variable/geist';
import '@fontsource-variable/mona-sans';
import '@fontsource-variable/instrument-sans';
import '@fontsource-variable/figtree';
import '@fontsource-variable/onest';
import '@fontsource-variable/plus-jakarta-sans';
import '@fontsource-variable/manrope';
import '@fontsource-variable/dm-sans';
import '@fontsource-variable/schibsted-grotesk';
import '@fontsource-variable/ibm-plex-sans';
import '@fontsource-variable/source-serif-4';
import '@fontsource-variable/jetbrains-mono';

/** `hint`: products known for the font (proper names, not translated); null = short style description from i18n (`font.hint.<id>`). */
export const FONTS = [
  { id: 'inter', label: 'Inter', family: "'Inter Variable'", hint: 'Linear, Figma, Raycast' },
  { id: 'geist', label: 'Geist', family: "'Geist Variable'", hint: 'Vercel' },
  { id: 'mona', label: 'Mona Sans', family: "'Mona Sans Variable'", hint: 'GitHub' },
  { id: 'instrument', label: 'Instrument Sans', family: "'Instrument Sans Variable'", hint: null },
  { id: 'figtree', label: 'Figtree', family: "'Figtree Variable'", hint: null },
  { id: 'onest', label: 'Onest', family: "'Onest Variable'", hint: null },
  { id: 'jakarta', label: 'Plus Jakarta Sans', family: "'Plus Jakarta Sans Variable'", hint: null },
  { id: 'manrope', label: 'Manrope', family: "'Manrope Variable'", hint: null },
  { id: 'dmsans', label: 'DM Sans', family: "'DM Sans Variable'", hint: null },
  { id: 'schibsted', label: 'Schibsted Grotesk', family: "'Schibsted Grotesk Variable'", hint: null },
  { id: 'plex', label: 'IBM Plex Sans', family: "'IBM Plex Sans Variable'", hint: 'IBM Carbon' },
  { id: 'system', label: 'System', family: "system-ui, -apple-system, 'Segoe UI Variable Text', 'Segoe UI', Roboto, sans-serif", hint: 'Segoe UI / SF Pro' }
] as const;

export type FontId = (typeof FONTS)[number]['id'];
export type HeadingFont = 'serif' | 'sans';
export const FONT_IDS: readonly FontId[] = FONTS.map((f) => f.id);
export const DEFAULT_FONT: FontId = 'inter';

/** Apply the font to <html> immediately (theme.js does this on load). */
export function applyFonts(font: FontId, heading: HeadingFont) {
  const el = document.documentElement;
  if (font === DEFAULT_FONT) delete el.dataset.font;
  else el.dataset.font = font;
  if (heading === 'serif') delete el.dataset.heading;
  else el.dataset.heading = heading;
}
