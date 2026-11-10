// Reading an Excel file chosen by the user (project tables, table templates):
// errors are shown as a toast, the result is null then.
import { t } from './i18n.svelte';
import { toast } from './state.svelte';
import { MAX_STORED, storedSize, type StoredSheet } from './sheetData';

export async function readXlsx(file: File): Promise<StoredSheet[] | null> {
  const { fromXlsx, ImportTooLarge } = await import('./sheetXlsx'); // ExcelJS only when needed
  try {
    const sheets = await fromXlsx(await file.arrayBuffer());
    if (storedSize(sheets) > MAX_STORED) {
      toast(t('table.importTooLarge'));
      return null;
    }
    return sheets;
  } catch (e) {
    toast(t(e instanceof ImportTooLarge ? 'table.importTooLarge' : 'table.importFailed'));
    return null;
  }
}

/** "Kalkulation.xlsx" → "Kalkulation" (title of the new table or template). */
export const titleFromFile = (name: string) => name.replace(/\.xlsx$/i, '').trim().slice(0, 200) || name.slice(0, 200);
