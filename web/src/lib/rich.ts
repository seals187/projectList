// Inline-Formatierung im Browser: Bereinigung (spiegelt die Server-Whitelist),
// Formatieren der Auswahl und Cursor-Hilfen für contenteditable-Blöcke.

import { HUES, SHADES } from './palette';

// Textfarben/Textmarker: gleiche Farbtöne und Stufen wie die Zeilenfarben;
// muss mit richtext.TextColors/HighlightColors übereinstimmen.
const PALETTE: readonly string[] = SHADES.flatMap((s) => HUES.map((h) => h + s));
export const TEXT_COLORS: readonly string[] = [...PALETTE, 'orange']; // "orange": ältere Einträge
export const HIGHLIGHTS: readonly string[] = PALETTE;

const TAGS: Record<string, string> = {
  B: 'b', STRONG: 'b', I: 'i', EM: 'i', U: 'u', INS: 'u', S: 's', STRIKE: 's', DEL: 's',
  CODE: 'code', A: 'a', SPAN: 'span', MARK: 'mark', BR: 'br'
};
const BLOCKS = new Set(['DIV', 'P', 'LI', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'BLOCKQUOTE', 'PRE']);
const DROP = new Set(['SCRIPT', 'STYLE', 'TEMPLATE', 'IFRAME', 'OBJECT']);

const esc = (s: string) =>
  s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!);

export function escapeHtml(s: string) {
  return esc(s);
}

function safeUrl(u: string) {
  return /^(https?:\/\/|mailto:)/i.test(u.trim());
}

function renderNode(n: Node, out: string[], depth: number) {
  if (depth > 20) return;
  if (n.nodeType === Node.TEXT_NODE) {
    out.push(esc(n.textContent ?? ''));
    return;
  }
  if (n.nodeType !== Node.ELEMENT_NODE) return;
  const el = n as Element;
  if (DROP.has(el.tagName)) return;
  const tag = TAGS[el.tagName];
  let open = '';
  if (tag === 'br') {
    out.push('<br>');
    return;
  }
  if (tag === 'a') {
    const href = el.getAttribute('href') ?? '';
    if (safeUrl(href)) open = `<a href="${esc(href.trim())}" target="_blank" rel="noopener noreferrer">`;
  } else if (tag === 'span') {
    const c = el.getAttribute('data-color') ?? '';
    if ((TEXT_COLORS as readonly string[]).includes(c)) open = `<span data-color="${c}">`;
  } else if (tag === 'mark') {
    let c = el.getAttribute('data-bg') ?? '';
    if (!(HIGHLIGHTS as readonly string[]).includes(c)) c = 'yellow';
    open = `<mark data-bg="${c}">`;
  } else if (tag) {
    open = `<${tag}>`;
  }
  if (!tag && BLOCKS.has(el.tagName) && el.previousSibling) out.push('<br>');
  out.push(open);
  el.childNodes.forEach((c) => renderNode(c, out, depth + 1));
  if (open) out.push(`</${tag}>`);
}

/** Bereinigt HTML auf die erlaubten Inline-Elemente (wie der Server). */
export function sanitizeHtml(html: string): string {
  if (!html) return '';
  if (!/[<&]/.test(html)) return esc(html);
  const doc = new DOMParser().parseFromString(`<body>${html}</body>`, 'text/html');
  const out: string[] = [];
  doc.body.childNodes.forEach((c) => renderNode(c, out, 0));
  // Ein einzelnes abschließendes <br> (contenteditable-Artefakt) entfernen.
  return out.join('').replace(/(<br>)+$/, '').trim();
}

/** Reiner Text aus (bereinigtem) HTML. */
export function htmlToText(html: string): string {
  if (!html) return '';
  if (!/[<&]/.test(html)) return html;
  const doc = new DOMParser().parseFromString(`<body>${html.replace(/<br\s*\/?>/gi, ' ')}</body>`, 'text/html');
  return doc.body.textContent ?? '';
}

// ---- Auswahl & Cursor ----------------------------------------------------------

export function selectionIn(root: HTMLElement): Range | null {
  const sel = getSelection();
  if (!sel || !sel.rangeCount) return null;
  const r = sel.getRangeAt(0);
  return root.contains(r.commonAncestorContainer) ? r : null;
}

/** Textlänge vom Anfang von root bis zum Cursor. */
export function caretOffset(root: HTMLElement): number {
  const r = selectionIn(root);
  if (!r) return 0;
  const pre = document.createRange();
  pre.selectNodeContents(root);
  pre.setEnd(r.startContainer, r.startOffset);
  return pre.toString().length;
}

export function caretAtStart(root: HTMLElement) {
  const r = selectionIn(root);
  return !!r && r.collapsed && caretOffset(root) === 0;
}

export function caretAtEnd(root: HTMLElement) {
  const r = selectionIn(root);
  if (!r || !r.collapsed) return false;
  const post = document.createRange();
  post.selectNodeContents(root);
  post.setStart(r.endContainer, r.endOffset);
  return post.toString().length === 0;
}

function caretRect(root: HTMLElement): DOMRect | null {
  const r = selectionIn(root);
  if (!r) return null;
  const rects = r.getClientRects();
  return rects.length ? rects[0] : null;
}

/** Steht der Cursor in der ersten bzw. letzten Zeile des Elements? */
export function caretOnFirstLine(root: HTMLElement) {
  if (caretAtStart(root)) return true;
  const c = caretRect(root);
  if (!c) return caretAtStart(root);
  return c.top - root.getBoundingClientRect().top < c.height * 0.8;
}

export function caretOnLastLine(root: HTMLElement) {
  if (caretAtEnd(root)) return true;
  const c = caretRect(root);
  if (!c) return caretAtEnd(root);
  return root.getBoundingClientRect().bottom - c.bottom < c.height * 0.8;
}

/** Setzt den Cursor an eine Text-Position (Ende bei -1). */
export function placeCaret(root: HTMLElement, offset: number) {
  root.focus({ preventScroll: false });
  const sel = getSelection();
  if (!sel) return;
  const range = document.createRange();
  if (offset < 0) {
    range.selectNodeContents(root);
    range.collapse(false);
  } else {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    let remaining = offset;
    let node: Node | null = walker.nextNode();
    let placed = false;
    while (node) {
      const len = node.textContent?.length ?? 0;
      if (remaining <= len) {
        range.setStart(node, remaining);
        placed = true;
        break;
      }
      remaining -= len;
      node = walker.nextNode();
    }
    if (!placed) {
      range.selectNodeContents(root);
      range.collapse(offset === 0);
    }
    range.collapse(true);
  }
  sel.removeAllRanges();
  sel.addRange(range);
}

/** Teilt den Inhalt am Cursor: gibt das HTML nach dem Cursor zurück und entfernt es aus root. */
export function extractAfterCaret(root: HTMLElement): string {
  const r = selectionIn(root);
  if (!r) return '';
  const after = document.createRange();
  after.selectNodeContents(root);
  after.setStart(r.endContainer, r.endOffset);
  const frag = after.extractContents();
  const div = document.createElement('div');
  div.appendChild(frag);
  return sanitizeHtml(div.innerHTML);
}

export function notifyInput(root: HTMLElement) {
  root.dispatchEvent(new Event('input', { bubbles: true }));
}

function unwrap(el: Element) {
  const parent = el.parentNode;
  if (!parent) return;
  while (el.firstChild) parent.insertBefore(el.firstChild, el);
  parent.removeChild(el);
}

/** Entfernt passende Elemente, die die Auswahl berühren (inkl. umschließender). */
export function unwrapInSelection(root: HTMLElement, selector: string) {
  const r = selectionIn(root);
  if (!r) return;
  const hits = new Set<Element>();
  root.querySelectorAll(selector).forEach((el) => {
    if (r.intersectsNode(el)) hits.add(el);
  });
  let n: Node | null = r.commonAncestorContainer;
  while (n && n !== root) {
    if (n instanceof Element && n.matches(selector)) hits.add(n);
    n = n.parentNode;
  }
  hits.forEach(unwrap);
}

/** Umschließt die Auswahl mit einem Element (vorher gleichartige entfernen). */
export function wrapSelection(root: HTMLElement, tag: string, attrs: Record<string, string> = {}, sameSelector?: string) {
  if (sameSelector) unwrapInSelection(root, sameSelector);
  const r = selectionIn(root);
  if (!r || r.collapsed) return;
  const frag = r.extractContents();
  if (sameSelector) frag.querySelectorAll(sameSelector).forEach(unwrap);
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
  el.appendChild(frag);
  r.insertNode(el);
  const sel = getSelection()!;
  const nr = document.createRange();
  nr.selectNodeContents(el);
  sel.removeAllRanges();
  sel.addRange(nr);
}

/** Ist die Auswahl (teilweise) in einem passenden Element? */
export function selectionWithin(root: HTMLElement, selector: string) {
  const r = selectionIn(root);
  if (!r) return false;
  let n: Node | null = r.commonAncestorContainer;
  while (n && n !== root) {
    if (n instanceof Element && n.matches(selector)) return true;
    n = n.parentNode;
  }
  return false;
}

/** Formatierung entfernen: Auswahl oder (ohne Auswahl) den ganzen Block. */
export function clearFormatting(root: HTMLElement) {
  const r = selectionIn(root);
  if (!r || r.collapsed) {
    root.innerHTML = esc(root.textContent ?? '');
    return;
  }
  const text = r.toString();
  r.deleteContents();
  r.insertNode(document.createTextNode(text));
}

export function insertPlainText(text: string) {
  document.execCommand('insertText', false, text);
}
