// Toast messages (bottom of the window): errors and short confirmations.

export interface Toast {
  id: number;
  text: string;
  kind: 'error' | 'info';
}

export const toasts = $state<{ list: Toast[] }>({ list: [] });
let toastId = 0;

export function toast(text: string, kind: Toast['kind'] = 'error') {
  const id = ++toastId;
  toasts.list.push({ id, text, kind });
  setTimeout(() => {
    toasts.list = toasts.list.filter((t) => t.id !== id);
  }, 4500);
}
