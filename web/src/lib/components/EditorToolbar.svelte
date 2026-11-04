<script lang="ts">
  // Formatierungsleiste des Block-Editors. Alle Buttons verhindern per
  // mousedown.preventDefault, dass der Fokus/die Auswahl im Block verloren geht.
  import { KINDS } from '$lib/blocks';
  import { t, tMaybe, type Key } from '$lib/i18n.svelte';
  import { colorLabel, HUES, MARK_SWATCH, SHADES, TEXT_SWATCH } from '$lib/palette';
  import type { EditorApi, Kind } from '$lib/types';
  import { prefs, setPref, stepZoom, ZOOM_STEPS } from '$lib/state.svelte';
  import Icon from './Icon.svelte';
  import BlockIcon from './BlockIcon.svelte';

  let { api }: { api: EditorApi } = $props();

  let menu = $state<'kind' | 'color' | 'highlight' | 'link' | null>(null);
  let url = $state('');

  const keep = (e: MouseEvent) => e.preventDefault();

  function toggle(m: typeof menu) {
    if (m === 'link') api.saveSelection();
    menu = menu === m ? null : m;
    url = '';
  }

  function pick(fn: () => void) {
    fn();
    menu = null;
  }

  const kindLabel = $derived(KINDS.find((k) => k.kind === api.kind)?.label ?? 'blocks.text');

  const btn = (on = false) =>
    `inline-flex h-7 min-w-7 items-center justify-center rounded-md px-1.5 text-[13px] transition-colors disabled:opacity-35 ${
      on
        ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/70 dark:text-accent-400'
        : 'text-zinc-600 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800'
    }`;
  const sep = 'mx-1 h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700';
  const pop =
    'absolute top-full left-0 z-40 mt-1 rounded-lg border border-zinc-200 bg-white p-1 shadow-lg dark:border-zinc-700 dark:bg-zinc-900';

  const headings: Kind[] = ['h1', 'h2', 'h3', 'h4'];
  const lists: { kind: Kind; icon: 'ul' | 'ol' | Kind; label: Key }[] = [
    { kind: 'bullet', icon: 'ul', label: 'blocks.bullet' },
    { kind: 'numbered', icon: 'ol', label: 'blocks.numbered' },
    { kind: 'item', icon: 'item', label: 'blocks.item' },
    { kind: 'quote', icon: 'quote', label: 'blocks.quote' },
    { kind: 'code', icon: 'code', label: 'blocks.code' }
  ];
</script>

<svelte:window onmousedown={(e) => menu && !(e.target as HTMLElement).closest('[data-toolbar]') && (menu = null)} />

<div
  data-toolbar
  class="sticky top-0 z-20 -mx-1 mb-5 flex items-center gap-0.5 gap-y-1 overflow-x-auto rounded-xl md:flex-wrap border border-zinc-200 bg-white/95 px-2 py-1.5 shadow-xs backdrop-blur md:overflow-visible dark:border-zinc-800 dark:bg-zinc-950/95"
  role="toolbar"
  tabindex="-1">
  <button type="button" class={btn()} onmousedown={keep} onclick={() => api.undo()} disabled={!api.canUndo} title={t('editor.undo')}>
    <Icon name="undo" size={15} />
  </button>
  <button type="button" class={btn()} onmousedown={keep} onclick={() => api.redo()} disabled={!api.canRedo} title={t('editor.redo')}>
    <Icon name="redo" size={15} />
  </button>
  <span class={sep}></span>

  <!-- Blocktyp -->
  <div class="relative">
    <button type="button" class="{btn(menu === 'kind')} w-32 justify-between! gap-1" onmousedown={keep} onclick={() => toggle('kind')} title={t('editor.blockType')}>
      <span class="truncate">{t(kindLabel)}</span>
      <Icon name="chevronDown" size={13} />
    </button>
    {#if menu === 'kind'}
      <div class="{pop} w-52">
        {#each KINDS.filter((k) => k.kind !== 'divider') as k (k.kind)}
          <button
            type="button"
            class="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800 {api.kind === k.kind ? 'text-accent-700 dark:text-accent-400' : ''}"
            onmousedown={keep}
            onclick={() => pick(() => api.setKind(k.kind))}>
            <span class="flex w-5 justify-center text-zinc-500"><BlockIcon kind={k.kind} size={15} /></span>
            {t(k.label)}
          </button>
        {/each}
      </div>
    {/if}
  </div>
  {#each headings as h (h)}
    <button type="button" class="{btn(api.kind === h)} font-semibold" onmousedown={keep} onclick={() => api.setKind(h)} title={t(`blocks.${h}`)}>
      {h.toUpperCase()}
    </button>
  {/each}
  <span class={sep}></span>

  <!-- Inline -->
  <button type="button" class="{btn(api.marks.bold)} font-bold" onmousedown={keep} onclick={() => api.format('bold')} title={t('editor.bold')}>B</button>
  <button type="button" class="{btn(api.marks.italic)} font-serif italic" onmousedown={keep} onclick={() => api.format('italic')} title={t('editor.italic')}>I</button>
  <button type="button" class="{btn(api.marks.underline)} underline" onmousedown={keep} onclick={() => api.format('underline')} title={t('editor.underline')}>U</button>
  <button type="button" class="{btn(api.marks.strike)} line-through" onmousedown={keep} onclick={() => api.format('strikeThrough')} title={t('editor.strike')}>S</button>
  <button type="button" class="{btn(api.marks.code)} font-mono text-xs" onmousedown={keep} onclick={() => api.format('code')} title={t('editor.inlineCode')}>&lt;/&gt;</button>
  <span class={sep}></span>

  <!-- Farben -->
  <div class="relative">
    <button type="button" class="{btn(menu === 'color')} gap-0.5" onmousedown={keep} onclick={() => toggle('color')} title={t('editor.textColor')}>
      <span class="flex flex-col items-center leading-none"><span class="font-semibold">A</span><span class="mt-0.5 h-[3px] w-3.5 rounded-full bg-red-500"></span></span>
      <Icon name="chevronDown" size={11} />
    </button>
    {#if menu === 'color'}
      <div class="{pop} w-max p-2">
        <button type="button" class="mb-1.5 flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" onmousedown={keep} onclick={() => pick(() => api.color(''))}>
          <span class="w-4 text-center font-semibold">A</span>{t('editor.colorDefault')}
        </button>
        <!-- gleiches Raster wie der Zeilen-Farbwähler: Zeilen = hell, mittel, kräftig -->
        <div class="grid grid-cols-6 gap-1">
          {#each SHADES as shade (shade)}
            {#each HUES as hue (hue)}
              {@const c = hue + shade}
              <button
                type="button"
                class="flex size-6 items-center justify-center rounded-md border border-zinc-200 text-[15px] font-bold hover:scale-110 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800 {TEXT_SWATCH[c]}"
                onmousedown={keep}
                onclick={() => pick(() => api.color(c))}
                title={colorLabel(c, (k) => tMaybe(k) ?? k)}
                aria-label={colorLabel(c, (k) => tMaybe(k) ?? k)}>A</button>
            {/each}
          {/each}
        </div>
      </div>
    {/if}
  </div>
  <div class="relative">
    <button type="button" class="{btn(menu === 'highlight')} gap-0.5" onmousedown={keep} onclick={() => toggle('highlight')} title={t('editor.highlight')}>
      <span class="flex flex-col items-center leading-none">
        <svg width="14" height="12" viewBox="0 0 24 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 3l6 6-9 9H5v-6z" /></svg>
        <span class="mt-0.5 h-[3px] w-3.5 rounded-full bg-yellow-300"></span>
      </span>
      <Icon name="chevronDown" size={11} />
    </button>
    {#if menu === 'highlight'}
      <div class="{pop} w-max p-2">
        <button type="button" class="mb-1.5 flex w-full items-center gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800" onmousedown={keep} onclick={() => pick(() => api.highlight(''))}>
          <span class="size-4 rounded border border-zinc-300 dark:border-zinc-600"></span>{t('editor.none')}
        </button>
        <!-- gleiches Raster wie der Zeilen-Farbwähler: Zeilen = hell, mittel, kräftig (Word) -->
        <div class="grid grid-cols-6 gap-1">
          {#each SHADES as shade (shade)}
            {#each HUES as hue (hue)}
              {@const c = hue + shade}
              <button
                type="button"
                class="size-6 rounded-md border border-black/10 hover:scale-110 {MARK_SWATCH[c]}"
                onmousedown={keep}
                onclick={() => pick(() => api.highlight(c))}
                title={colorLabel(c, (k) => tMaybe(k) ?? k)}
                aria-label={colorLabel(c, (k) => tMaybe(k) ?? k)}></button>
            {/each}
          {/each}
        </div>
      </div>
    {/if}
  </div>
  <span class={sep}></span>

  <!-- Blöcke -->
  {#each lists as l (l.kind)}
    <button type="button" class={btn(api.kind === l.kind)} onmousedown={keep} onclick={() => api.setKind(l.kind)} title={t(l.label)}>
      <BlockIcon kind={l.icon} size={15} />
    </button>
  {/each}
  <button type="button" class={btn()} onmousedown={keep} onclick={() => api.openLink()} title={t('editor.link')}>
    <Icon name="link" size={15} />
  </button>
  <button type="button" class={btn()} onmousedown={keep} onclick={() => api.add('divider')} title={t('blocks.divider')}>
    <BlockIcon kind="divider" size={15} />
  </button>
  <span class={sep}></span>
  <button type="button" class={btn()} onmousedown={keep} onclick={() => api.clear()} title={t('editor.clear')}>
    <Icon name="eraser" size={15} />
  </button>

  <!-- rechte Gruppe: bleibt beim Umbrechen zusammen und rechtsbündig -->
  <div class="ml-auto flex shrink-0 items-center pl-2">
  <!-- Rahmenlinien ein/aus (persönlich, pro Browser) -->
  <button type="button" class="{btn(prefs.gridLines)} mr-1.5" onmousedown={keep} onclick={() => setPref('gridLines', !prefs.gridLines)} aria-pressed={prefs.gridLines} title={t('settings.gridLines')} aria-label={t('settings.gridLines')}>
    <Icon name="rows" size={15} />
  </button>
  <!-- Größe der Checkliste (persönlich, pro Browser) -->
  <div class="mr-1.5 flex shrink-0 items-center rounded-md border border-zinc-200 dark:border-zinc-700" role="group" aria-label={t('editor.size')}>
    <button type="button" class="{btn()} h-6! min-w-6! rounded-r-none!" onmousedown={keep} onclick={() => stepZoom(-1)} disabled={prefs.zoom <= ZOOM_STEPS[0]} title={t('editor.smaller')} aria-label={t('editor.smaller')}>
      <Icon name="minus" size={13} />
    </button>
    <button type="button" class="{btn()} h-6! w-11 rounded-none! px-0! text-[11px]! tabular-nums" onmousedown={keep} onclick={() => stepZoom(0)} title={t('editor.resetSize')}>
      {Math.round(prefs.zoom * 100)}%
    </button>
    <button type="button" class="{btn()} h-6! min-w-6! rounded-l-none!" onmousedown={keep} onclick={() => stepZoom(1)} disabled={prefs.zoom >= ZOOM_STEPS[ZOOM_STEPS.length - 1]} title={t('editor.larger')} aria-label={t('editor.larger')}>
      <Icon name="plus" size={13} />
    </button>
  </div>
  <button type="button" class="btn-outline shrink-0 px-2.5! py-1! text-[13px]! text-accent-700! dark:text-accent-400!" onmousedown={keep} onclick={() => api.add('h1')}>
    <Icon name="plus" size={13} />
    {t('editor.addHeading')}
  </button>
  <button type="button" class="btn-primary ml-1 shrink-0 px-2.5! py-1! text-[13px]!" onmousedown={keep} onclick={() => api.add('item')}>
    <Icon name="plus" size={13} />
    {t('editor.addItem')}
  </button>
  </div>
</div>
