<script lang="ts">
  // Floating menu at the mouse position (right click) or below a button.
  // Stays inside the viewport, closes on outside click, Escape or scroll of the page.
  import type { Snippet } from 'svelte';

  let { x, y, onclose, label, children }: { x: number; y: number; onclose: () => void; label: string; children: Snippet } = $props();

  let el = $state<HTMLDivElement>();
  let pos = $state({ left: 0, top: 0 });

  // keep inside the viewport (also when the content grows, e.g. an expanded section,
  // or the window shrinks, e.g. the on-screen keyboard)
  function place() {
    if (!el) return;
    const r = el.getBoundingClientRect();
    pos = {
      left: Math.max(8, Math.min(x, innerWidth - r.width - 8)),
      top: Math.max(8, Math.min(y, innerHeight - r.height - 8))
    };
  }
  $effect(() => {
    if (!el) return;
    place();
    const ro = new ResizeObserver(place);
    ro.observe(el);
    return () => ro.disconnect();
  });

  // Render at the end of <body>: inside the header band (class "dark" over a
  // background image) the menu would otherwise inherit the dark variants even
  // in the light theme.
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }

  function outside(e: Event) {
    if (el && !el.contains(e.target as Node)) onclose();
  }
</script>

<svelte:window
  onmousedown={outside}
  onkeydown={(e) => e.key === 'Escape' && onclose()}
  onresize={place}
  onscroll={(e) => !el?.contains(e.target as Node) && onclose()} />

<div
  bind:this={el}
  use:portal
  role="menu"
  aria-label={label}
  tabindex="-1"
  class="fixed z-50 max-h-[calc(100vh-1rem)] w-72 overflow-y-auto rounded-xl border border-zinc-200/80 bg-zinc-50/75 p-1.5 text-sm text-zinc-900 shadow-xl backdrop-blur-xl dark:border-zinc-700 dark:bg-zinc-900/90 dark:text-zinc-100"
  style:left="{pos.left}px"
  style:top="{pos.top}px">
  {@render children()}
</div>
