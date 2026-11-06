// Relative time for "last active" / "last checked": "just now" below a minute
// (or below `justNowMin` minutes), otherwise "2 min. ago", "3 days ago" … in the
// UI language (Intl.RelativeTimeFormat, short style).

const UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ['year', 365 * 86400],
  ['month', 30 * 86400],
  ['week', 7 * 86400],
  ['day', 86400],
  ['hour', 3600],
  ['minute', 60]
];

/** null = shorter than `justNowMin` minutes (caller shows "just now"). */
export function relTime(iso: string, now: Date, locale: string, justNowMin = 1): string | null {
  const secs = Math.round((now.getTime() - Date.parse(iso)) / 1000);
  if (!Number.isFinite(secs) || secs < justNowMin * 60) return null;
  const fmt = new Intl.RelativeTimeFormat(locale, { numeric: 'auto', style: 'short' });
  for (const [unit, s] of UNITS) if (secs >= s) return fmt.format(-Math.floor(secs / s), unit);
  return null;
}
