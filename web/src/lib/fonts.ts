// Wählbare Schriften (Einstellungen). Alle selbst gehostet über @fontsource
// (Open Font License, keine externen Anfragen); geladen wird nur die gerade
// verwendete Schrift. Die Zuordnung id → font-family steht in app.css (html[data-font=…]).
import '@fontsource-variable/inter/opsz.css'; // mit optischer Größe (große Titel wie „Inter Display“)
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

/** `hint`: wo die Schrift (bzw. ihr Stil) bekannt ist – Orientierung in den Einstellungen. */
export const FONTS = [
  { id: 'inter', label: 'Inter', family: "'Inter Variable'", hint: 'Linear, Figma, Raycast' },
  { id: 'geist', label: 'Geist', family: "'Geist Variable'", hint: 'Vercel' },
  { id: 'mona', label: 'Mona Sans', family: "'Mona Sans Variable'", hint: 'GitHub' },
  { id: 'instrument', label: 'Instrument Sans', family: "'Instrument Sans Variable'", hint: 'Clean, kompakt' },
  { id: 'figtree', label: 'Figtree', family: "'Figtree Variable'", hint: 'Freundlich, geometrisch' },
  { id: 'onest', label: 'Onest', family: "'Onest Variable'", hint: 'Ruhig, gut lesbar' },
  { id: 'jakarta', label: 'Plus Jakarta Sans', family: "'Plus Jakarta Sans Variable'", hint: 'SaaS-Dashboards' },
  { id: 'manrope', label: 'Manrope', family: "'Manrope Variable'", hint: 'Modern, technisch' },
  { id: 'dmsans', label: 'DM Sans', family: "'DM Sans Variable'", hint: 'Geometrisch, leicht' },
  { id: 'schibsted', label: 'Schibsted Grotesk', family: "'Schibsted Grotesk Variable'", hint: 'Editorial' },
  { id: 'plex', label: 'IBM Plex Sans', family: "'IBM Plex Sans Variable'", hint: 'IBM Carbon' },
  { id: 'system', label: 'System', family: "system-ui, -apple-system, 'Segoe UI Variable Text', 'Segoe UI', Roboto, sans-serif", hint: 'Segoe UI / SF Pro' }
] as const;

export type FontId = (typeof FONTS)[number]['id'];
export type HeadingFont = 'serif' | 'sans';
export const FONT_IDS: readonly FontId[] = FONTS.map((f) => f.id);
export const DEFAULT_FONT: FontId = 'inter';

/** Schrift sofort auf <html> anwenden (theme.js macht das beim Laden). */
export function applyFonts(font: FontId, heading: HeadingFont) {
  const el = document.documentElement;
  if (font === DEFAULT_FONT) delete el.dataset.font;
  else el.dataset.font = font;
  if (heading === 'serif') delete el.dataset.heading;
  else el.dataset.heading = heading;
}
