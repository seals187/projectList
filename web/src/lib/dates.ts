// Calendar dates as day numbers (days since 1970-01-01, UTC): no time zone or
// daylight saving surprises when adding days or measuring ranges.

/** "YYYY-MM-DD" → day number. */
export const toDay = (iso: string): number => Math.floor(Date.UTC(+iso.slice(0, 4), +iso.slice(5, 7) - 1, +iso.slice(8, 10)) / 86_400_000);

/** Day number → "YYYY-MM-DD". */
export const fromDay = (day: number): string => new Date(day * 86_400_000).toISOString().slice(0, 10);

/** Weekday 0 = Monday … 6 = Sunday. */
export const weekday = (day: number): number => (((day + 3) % 7) + 7) % 7; // 1970-01-01 was a Thursday

/** Monday of the week containing the day. */
export const mondayOf = (day: number): number => day - weekday(day);

/** First day of the month containing the day. */
export function monthStart(day: number): number {
  const d = new Date(day * 86_400_000);
  return Math.floor(Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), 1) / 86_400_000);
}

/** Add months (to the first day of a month). */
export function addMonths(day: number, n: number): number {
  const d = new Date(day * 86_400_000);
  return Math.floor(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + n, 1) / 86_400_000);
}

/** Day of the month (1–31). */
export const dayOfMonth = (day: number): number => new Date(day * 86_400_000).getUTCDate();

/** ISO week number. */
export function isoWeek(day: number): number {
  const thursday = day - weekday(day) + 3;
  const yearStart = Math.floor(Date.UTC(new Date(thursday * 86_400_000).getUTCFullYear(), 0, 1) / 86_400_000);
  return Math.floor((thursday - yearStart) / 7) + 1;
}

/** Month name ("November 2026") in the UI language. */
export const monthLabel = (day: number, locale: string, withYear = true): string =>
  new Date(day * 86_400_000).toLocaleDateString(locale, { month: 'long', ...(withYear ? { year: 'numeric' } : {}), timeZone: 'UTC' });

/** Short weekday names Monday … Sunday in the UI language. */
export const weekdayNames = (locale: string): string[] =>
  Array.from({ length: 7 }, (_, i) => new Date(Date.UTC(2024, 0, 1 + i)).toLocaleDateString(locale, { weekday: 'short', timeZone: 'UTC' }));
