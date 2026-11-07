// Card colors of the overview (Cards, Board, List): personal per user (pref
// "card_colors"), unless the shared customer color is set to "use in all views".
// Names must match cardColors in internal/httpapi/handlers_colors.go.
// A card is tinted with its color (light and dark theme, see .card-tint in app.css);
// "<name>-strong" is a stronger tint (darker in the light theme, lighter in the dark one).
import type { SidebarCustomer } from './types';

export const CARD_COLORS = ['red', 'orange', 'amber', 'yellow', 'lime', 'green', 'teal', 'cyan', 'blue', 'indigo', 'violet', 'pink', 'gray'] as const;
export type CardColor = (typeof CARD_COLORS)[number];

export const CARD_HEX: Record<CardColor, string> = {
  red: '#ef4444',
  orange: '#f97316',
  amber: '#f59e0b',
  yellow: '#eab308',
  lime: '#84cc16',
  green: '#22c55e',
  teal: '#14b8a6',
  cyan: '#06b6d4',
  blue: '#3b82f6',
  indigo: '#6366f1',
  violet: '#8b5cf6',
  pink: '#ec4899',
  gray: '#71717a'
};

/** Where a personal card color applies: customer cards ("cards", "list") or project cards ("board"). */
export type CardView = 'cards' | 'list' | 'board';
type ColorFields = Pick<SidebarCustomer, 'color' | 'color_everywhere'>;

/** Personal card colors of the user (pref "card_colors"): view → id → color. */
export type MyColors = Record<CardView, Record<string, string>>;

export const emptyMyColors = (): MyColors => ({ cards: {}, list: {}, board: {} });

/** Validated personal colors from the stored JSON (unknown colors are dropped). */
export function parseMyColors(raw: unknown): MyColors {
  const out = emptyMyColors();
  if (!raw || typeof raw !== 'object') return out;
  for (const view of ['cards', 'list', 'board'] as const) {
    const m = (raw as Record<string, unknown>)[view];
    if (!m || typeof m !== 'object') continue;
    for (const [id, v] of Object.entries(m)) if (/^\d+$/.test(id) && typeof v === 'string' && parseCardColor(v)) out[view][id] = v;
  }
  return out;
}

/** Copy with one personal color set ("" removes it). */
export function withMyColor(all: MyColors, view: CardView, id: number, color: string): MyColors {
  const next = { ...all, [view]: { ...all[view] } };
  if (color) next[view][String(id)] = color;
  else delete next[view][String(id)];
  return next;
}

/** The shared customer color wins everywhere when it is set and "use everywhere" is on
 *  (for all users; also on Customers and My tasks, which have no colors of their own). */
export function customerWide(c: ColorFields): string {
  return c.color_everywhere && c.color ? c.color : '';
}

/** Color of a customer card in the "Cards" or "List" view ("" = none). */
export function customerCardColor(c: ColorFields & { id: number }, view: 'cards' | 'list', mine: MyColors): string {
  return customerWide(c) || mine[view][String(c.id)] || '';
}

/** Color of a project card in the board ("" = none). */
export function projectCardColor(p: { id: number }, c: ColorFields, mine: MyColors): string {
  return customerWide(c) || mine.board[String(p.id)] || '';
}

export const STRONG = '-strong';

/** Splits a stored value ("blue", "blue-strong") into base color and shade; null if unknown. */
export function parseCardColor(value: string): { base: CardColor; strong: boolean } | null {
  const strong = value.endsWith(STRONG);
  const base = (strong ? value.slice(0, -STRONG.length) : value) as CardColor;
  return CARD_COLORS.includes(base) ? { base, strong } : null;
}

/** Solid color of a value (swatch dot), "" if unknown. */
export function cardHex(value: string): string {
  const c = parseCardColor(value);
  return c ? CARD_HEX[c.base] : '';
}

/** Inline style for a tinted card (class "card-tint"); undefined without a color.
 *  --tint-l / --tint-d: share of the color in the light / dark theme. */
export function tintStyle(color: string): string | undefined {
  const c = parseCardColor(color);
  if (!c) return undefined;
  return `--tint:${CARD_HEX[c.base]}` + (c.strong ? ';--tint-l:30%;--tint-d:40%' : '');
}

/** Personal color of a project (set in the project menu; "" = none): tints the board card and the
 *  project rows elsewhere (cards, list, My tasks, customers). */
export function projectColor(id: number, mine: MyColors): string {
  return mine.board[String(id)] ?? '';
}

/** Inline style for a color stripe at the left edge (board card tinted by the customer color). */
export function stripeStyle(color: string): string | undefined {
  const hex = cardHex(color);
  return hex ? `box-shadow: inset 3px 0 0 ${hex}` : undefined;
}
