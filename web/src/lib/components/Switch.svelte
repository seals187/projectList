<script lang="ts">
  // On/off switch in the accent color.
  let { checked, onchange, label }: { checked: boolean; onchange: (v: boolean) => void; label: string } = $props();
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked}
  aria-label={label}
  class="relative inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors {checked ? 'bg-[var(--accent-primary)]' : 'bg-zinc-300 dark:bg-zinc-600'}"
  onclick={() => onchange(!checked)}>
  <span class="inline-block size-[22px] rounded-full bg-white shadow transition-transform {checked ? 'translate-x-[23px]' : 'translate-x-[3px]'}"></span>
</button>
