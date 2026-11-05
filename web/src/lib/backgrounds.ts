// Backgrounds: gradient presets and uploaded images. A value is '' (none),
// 'g<n>' (gradient preset) or 'img:<id>' (uploaded image). The most specific
// one applies: project, then customer, then the user's own background.
import { api } from './api';
import { sidebar } from './state.svelte';

export const GRADIENTS: { id: string; css: string }[] = [
  ['#b5d34f', '#6d8f1f'],
  ['#3cc6dc', '#1b5f8f'],
  ['#2b0b0e', '#a4321e'],
  ['#717720', '#3a3f0e'],
  ['#f8b0a4', '#e2709b'],
  ['#8e2f48', '#d9606e'],
  ['#f7d84f', '#f59a1c'],
  ['#e8642a', '#9c220c'],
  ['#1b3a78', '#3d94d6'],
  ['#0c7d43', '#22c16d'],
  ['#14294f', '#2e5ea8'],
  ['#e05c62', '#c43c4a'],
  ['#ff9563', '#ff6a3d'],
  ['#18403f', '#2f7a73'],
  ['#157f92', '#3cbccb'],
  ['#f4b07f', '#4b2c5e'],
  ['#272a38', '#4d556e'],
  ['#3fa9f5', '#c38fff', '#f28fb8'],
  ['#6e1159', '#c71f86'],
  ['#3f235f', '#9168c4'],
  ['#25345d', '#7383b3'],
  ['#0e4a63', '#1f8db3'],
  ['#ffb36b', '#ff5e8a', '#4cc9f0'],
  ['#2d7f6d', '#7ad3a3'],
  ['#e8174f', '#ff7092']
].map((c, i) => ({ id: `g${i + 1}`, css: `linear-gradient(135deg, ${c.join(', ')})` }));

/** CSS for a background value (gradient or image URL); '' = none. */
export function backgroundCss(value: string | undefined | null): string {
  if (!value) return '';
  if (value.startsWith('img:')) return `url(/api/backgrounds/${encodeURIComponent(value.slice(4))})`;
  return GRADIENTS.find((g) => g.id === value)?.css ?? '';
}

/** The user's own background (synced pref). */
export const userBackground = (): string => (typeof sidebar.prefs.background === 'string' ? sidebar.prefs.background : '');

export async function setUserBackground(value: string) {
  const prev = sidebar.prefs.background;
  sidebar.prefs.background = value;
  try {
    await api.put('/prefs/background', value);
  } catch (e) {
    sidebar.prefs.background = prev;
    throw e;
  }
}

export interface BackgroundImage {
  id: number;
  mime: string;
  size: number;
  uploaded_by: number | null;
  created_at: string;
}

export const MAX_BACKGROUND = 5 * 1024 * 1024;
const MAX_SIDE = 2560; // px; a background never needs more

/**
 * Prepares an image for upload: images up to 5 MB are kept as they are,
 * larger ones are scaled down (max. 2560 px) and re-encoded as JPEG until they
 * fit into 5 MB.
 */
export async function prepareBackground(file: File): Promise<Blob> {
  if (file.size <= MAX_BACKGROUND && /^image\/(jpeg|png|webp|gif)$/.test(file.type)) return file;
  const bitmap = await createImageBitmap(file);
  let scale = Math.min(1, MAX_SIDE / Math.max(bitmap.width, bitmap.height));
  for (let attempt = 0; attempt < 8; attempt++) {
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(bitmap.width * scale));
    canvas.height = Math.max(1, Math.round(bitmap.height * scale));
    canvas.getContext('2d')!.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    const quality = Math.max(0.5, 0.88 - attempt * 0.08);
    const blob = await new Promise<Blob | null>((res) => canvas.toBlob(res, 'image/jpeg', quality));
    if (blob && blob.size <= MAX_BACKGROUND) return blob;
    if (quality <= 0.5) scale *= 0.8; // still too big: smaller dimensions
  }
  throw new Error('image_too_large');
}
