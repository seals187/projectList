<script lang="ts">
  import { initials } from '$lib/i18n.svelte';

  let { name, id = 0, size = 24 }: { name: string; id?: number; size?: number } = $props();

  const colors = ['bg-orange-400', 'bg-sky-500', 'bg-emerald-500', 'bg-violet-500', 'bg-rose-500', 'bg-amber-500', 'bg-teal-500'];
  const color = $derived(colors[Math.abs(id) % colors.length]);
</script>

<span
  class="inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white {color}"
  style:width="{size}px"
  style:height="{size}px"
  style:font-size="{Math.round(size * 0.42)}px"
  aria-hidden="true">{initials(name)}</span>
