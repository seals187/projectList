// Leichtgewichtiges Drag & Drop zum Sortieren mit Pointer-Events
// (funktioniert mit Maus, Touch und Stift). Das DOM wird während des Ziehens
// nur per transform verschoben; die echte Umsortierung macht Svelte, sobald
// onsort die neue Reihenfolge übernimmt.
//
// Verwendung:
//   <ul use:sortable={{ onsort: (ids) => …, disabled }}>
//     <li data-id={x.id}> <button data-handle>⋮⋮</button> … </li>
//   </ul>

export interface SortableOptions {
  onsort: (ids: number[]) => void;
  disabled?: boolean;
}

export function sortable(node: HTMLElement, options: SortableOptions) {
  let opts = options;

  function onPointerDown(e: PointerEvent) {
    if (opts.disabled || e.button !== 0) return;
    const handle = (e.target as HTMLElement).closest('[data-handle]');
    if (!handle || !node.contains(handle)) return;
    const el = handle.closest('[data-id]') as HTMLElement | null;
    if (!el || el.parentElement !== node) return;
    e.preventDefault();

    // Nur sichtbare Einträge (eingeklappte Abschnitte haben display:none);
    // onsort erhält dann die Reihenfolge der sichtbaren IDs.
    const items = Array.from(node.children).filter(
      (c): c is HTMLElement => c instanceof HTMLElement && !!c.dataset.id && c.getClientRects().length > 0
    );
    const from = items.indexOf(el);
    const rects = items.map((i) => i.getBoundingClientRect());
    const gap = rects.length > 1 ? Math.max(0, rects[1].top - rects[0].bottom) : 0;
    const shift = rects[from].height + gap;
    const startY = e.clientY;
    // Bei CSS zoom sind Bildschirm- und Elementkoordinaten unterschiedlich skaliert.
    const scale = el.offsetHeight ? rects[from].height / el.offsetHeight : 1;
    let to = from;

    el.setPointerCapture(e.pointerId);
    el.style.position = 'relative';
    el.style.zIndex = '20';
    el.classList.add('dragging');
    for (const i of items) if (i !== el) i.style.transition = 'transform 150ms ease';

    function onMove(ev: PointerEvent) {
      const dy = ev.clientY - startY;
      el!.style.transform = `translateY(${dy / scale}px)`;
      const center = rects[from].top + rects[from].height / 2 + dy;
      to = 0;
      items.forEach((it, idx) => {
        if (idx === from) return;
        if (rects[idx].top + rects[idx].height / 2 < center) to++;
      });
      items.forEach((it, idx) => {
        if (idx === from) return;
        let t = 0;
        if (from < to && idx > from && idx <= to) t = -shift;
        if (from > to && idx < from && idx >= to) t = shift;
        it.style.transform = t ? `translateY(${t / scale}px)` : '';
      });
    }

    function onUp() {
      el!.removeEventListener('pointermove', onMove);
      el!.removeEventListener('pointerup', onUp);
      el!.removeEventListener('pointercancel', onUp);
      el!.classList.remove('dragging');
      for (const i of items) {
        i.style.transition = '';
        i.style.transform = '';
        i.style.position = '';
        i.style.zIndex = '';
      }
      if (to !== from) {
        const ids = items.map((i) => Number(i.dataset.id));
        const [moved] = ids.splice(from, 1);
        ids.splice(to, 0, moved);
        opts.onsort(ids);
      }
    }

    el.addEventListener('pointermove', onMove);
    el.addEventListener('pointerup', onUp);
    el.addEventListener('pointercancel', onUp);
  }

  node.addEventListener('pointerdown', onPointerDown);
  return {
    update(o: SortableOptions) {
      opts = o;
    },
    destroy() {
      node.removeEventListener('pointerdown', onPointerDown);
    }
  };
}

/** Ordnet ein Array nach der übergebenen ID-Reihenfolge neu. */
export function reorder<T extends { id: number }>(list: T[], ids: number[]): T[] {
  const byId = new Map(list.map((x) => [x.id, x]));
  return ids.map((id) => byId.get(id)!).filter(Boolean);
}
