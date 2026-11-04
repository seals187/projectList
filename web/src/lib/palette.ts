// Color palette for row/section colors, modeled on Word/Excel:
// three shades per hue – light (subtle), medium, strong (highlighter/"neon").
// Values: "<hue>" | "<hue>-mid" | "<hue>-strong".
// Must match bgColors (internal/httpapi/dto.go).
//
// In dark mode the colors are laid transparently over the dark background
// so that light text stays readable.
// All classes are written out here so Tailwind finds them.

export const HUES = ['gray', 'yellow', 'green', 'blue', 'red', 'purple'] as const;
export const SHADES = ['', '-mid', '-strong'] as const;
export type Shade = (typeof SHADES)[number];

/** Background of a row or section. */
export const ROW_BG: Record<string, string> = {
  gray: 'bg-zinc-100/80 dark:bg-zinc-800/50',
  yellow: 'bg-amber-50 dark:bg-amber-400/[0.08]',
  green: 'bg-emerald-50 dark:bg-emerald-400/[0.08]',
  blue: 'bg-sky-50 dark:bg-sky-400/[0.08]',
  red: 'bg-rose-50 dark:bg-rose-400/[0.08]',
  purple: 'bg-violet-50 dark:bg-violet-400/[0.09]',

  'gray-mid': 'bg-zinc-200 dark:bg-zinc-700/60',
  'yellow-mid': 'bg-yellow-200 dark:bg-yellow-400/25',
  'green-mid': 'bg-green-200 dark:bg-green-400/25',
  'blue-mid': 'bg-sky-200 dark:bg-sky-400/25',
  'red-mid': 'bg-red-200 dark:bg-red-400/25',
  'purple-mid': 'bg-violet-200 dark:bg-violet-400/25',

  'gray-strong': 'bg-[#bfbfbf] dark:bg-zinc-500/50',
  'yellow-strong': 'bg-[#ffff00] dark:bg-[#ffff00]/30',
  'green-strong': 'bg-[#00ff00] dark:bg-[#00ff00]/25',
  'blue-strong': 'bg-[#00ffff] dark:bg-[#00ffff]/25',
  'red-strong': 'bg-[#ff4040] dark:bg-[#ff4040]/35',
  'purple-strong': 'bg-[#ff00ff] dark:bg-[#ff00ff]/30'
};

/** Swatches in the picker menu (as the color looks in light mode). */
export const SWATCH: Record<string, string> = {
  gray: 'bg-zinc-100',
  yellow: 'bg-amber-100',
  green: 'bg-emerald-100',
  blue: 'bg-sky-100',
  red: 'bg-rose-100',
  purple: 'bg-violet-100',

  'gray-mid': 'bg-zinc-200',
  'yellow-mid': 'bg-yellow-200',
  'green-mid': 'bg-green-200',
  'blue-mid': 'bg-sky-200',
  'red-mid': 'bg-red-200',
  'purple-mid': 'bg-violet-200',

  'gray-strong': 'bg-[#bfbfbf]',
  'yellow-strong': 'bg-[#ffff00]',
  'green-strong': 'bg-[#00ff00]',
  'blue-strong': 'bg-[#00ffff]',
  'red-strong': 'bg-[#ff4040]',
  'purple-strong': 'bg-[#ff00ff]'
};

/** Name for tooltip: hue + shade. */
export function colorLabel(value: string, tr: (k: string) => string): string {
  const [hue, shade] = value.split('-');
  return shade ? `${tr(`color.${hue}`)} (${tr(`color.shade.${shade}`)})` : tr(`color.${hue}`);
}

/** Text color in the menu (letter "A"), like [data-color] in app.css. */
export const TEXT_SWATCH: Record<string, string> = {
  gray: 'text-zinc-500 dark:text-zinc-400',
  yellow: 'text-amber-500 dark:text-amber-400',
  green: 'text-emerald-600 dark:text-emerald-400',
  blue: 'text-blue-600 dark:text-blue-400',
  red: 'text-red-600 dark:text-red-400',
  purple: 'text-violet-600 dark:text-violet-400',
  'gray-mid': 'text-zinc-700 dark:text-zinc-300',
  'yellow-mid': 'text-amber-700 dark:text-amber-300',
  'green-mid': 'text-emerald-800 dark:text-emerald-300',
  'blue-mid': 'text-blue-800 dark:text-blue-300',
  'red-mid': 'text-red-800 dark:text-red-300',
  'purple-mid': 'text-violet-800 dark:text-violet-300',
  'gray-strong': 'text-[#7f7f7f] dark:text-[#a6a6a6]',
  'yellow-strong': 'text-[#ffc000]',
  'green-strong': 'text-[#00b050]',
  'blue-strong': 'text-[#0070c0] dark:text-[#3b9cff]',
  'red-strong': 'text-[#ff0000] dark:text-[#ff5050]',
  'purple-strong': 'text-[#7030a0] dark:text-[#b07ce0]'
};

/** Highlighter in the menu, like mark[data-bg] in app.css (light mode). */
export const MARK_SWATCH: Record<string, string> = {
  gray: 'bg-zinc-200',
  yellow: 'bg-yellow-200',
  green: 'bg-emerald-200',
  blue: 'bg-sky-200',
  red: 'bg-rose-200',
  purple: 'bg-violet-200',
  'gray-mid': 'bg-zinc-300',
  'yellow-mid': 'bg-yellow-300',
  'green-mid': 'bg-green-300',
  'blue-mid': 'bg-sky-300',
  'red-mid': 'bg-red-300',
  'purple-mid': 'bg-violet-300',
  'gray-strong': 'bg-[#bfbfbf]',
  'yellow-strong': 'bg-[#ffff00]',
  'green-strong': 'bg-[#00ff00]',
  'blue-strong': 'bg-[#00ffff]',
  'red-strong': 'bg-[#ff4040]',
  'purple-strong': 'bg-[#ff00ff]'
};
