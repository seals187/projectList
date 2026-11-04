<script lang="ts">
  // Circular progress with the percentage in the center (board cards).
  let { done, total, size = 34 }: { done: number; total: number; size?: number } = $props();
  const pct = $derived(total ? Math.round((done / total) * 100) : 0);
  const r = 15.5;
  const c = 2 * Math.PI * r;
</script>

<span class="relative inline-flex shrink-0 items-center justify-center" style:width="{size}px" style:height="{size}px" title="{done}/{total}">
  <svg viewBox="0 0 36 36" class="absolute inset-0 -rotate-90" aria-hidden="true">
    <circle cx="18" cy="18" r={r} fill="none" stroke-width="3" class="stroke-zinc-200 dark:stroke-zinc-800" />
    <circle
      cx="18" cy="18" r={r} fill="none" stroke-width="3" stroke-linecap="round"
      class="transition-[stroke-dashoffset] duration-300 {pct === 100 ? 'stroke-emerald-500' : 'stroke-accent-500'}"
      stroke-dasharray={c}
      stroke-dashoffset={c * (1 - pct / 100)} />
  </svg>
  <span class="relative text-[9px] font-semibold tabular-nums">{pct}%</span>
</span>
