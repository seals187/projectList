<script lang="ts">
  // Key figure of the reporting: label, large value, note next to it
  // (e.g. change vs. the previous period). Tone colors the note (good / bad /
  // muted); valueTone "bad" colors the value itself (e.g. overdue).
  type Tone = 'good' | 'bad' | 'muted';
  let {
    label,
    value,
    note = '',
    tone = 'muted',
    valueTone = null,
    testid
  }: { label: string; value: string | number; note?: string; tone?: Tone; valueTone?: 'bad' | null; testid?: string } = $props();
  const toneCls: Record<Tone, string> = {
    good: 'text-emerald-700 dark:text-emerald-400',
    bad: 'text-red-600 dark:text-red-400',
    muted: 'text-zinc-500 dark:text-zinc-400'
  };
</script>

<div class="surface min-w-0 rounded-xl border border-zinc-200 px-4 py-3 dark:border-zinc-800" data-testid={testid}>
  <p class="truncate text-[13px] text-zinc-500 dark:text-zinc-400">{label}</p>
  <p class="mt-0.5 flex min-w-0 items-baseline gap-2">
    <span class="truncate {typeof value === 'string' && value.length > 8 ? 'text-[22px]' : 'text-[28px]'} leading-tight font-bold tabular-nums {valueTone === 'bad' ? 'text-red-600 dark:text-red-400' : ''}">{value}</span>
    {#if note}<span class="max-w-[60%] shrink-0 truncate text-[13px] font-semibold {toneCls[tone]}">{note}</span>{/if}
  </p>
</div>
