<script lang="ts">
  // Templates page, view "Variables": system and custom variables ({{name}})
  // with source, description, example / value and usage. Click on a variable
  // copies "{{name}}"; admins edit custom variables (click on the row).
  import { toast } from '$lib/state.svelte';
  import { t, type Key } from '$lib/i18n.svelte';
  import type { TemplateVariable } from '$lib/types';

  let { variables, admin, onedit }: { variables: TemplateVariable[]; admin: boolean; onedit: (v: TemplateVariable) => void } = $props();

  const editable = (v: TemplateVariable) => admin && !!v.id;
  const badgeCls = (s: string) =>
    s === 'system'
      ? 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300'
      : s === 'custom'
        ? 'bg-accent-50 text-accent-700 dark:bg-accent-950/60 dark:text-accent-300'
        : 'bg-sky-50 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300';

  async function copy(e: MouseEvent, name: string) {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(`{{${name}}}`);
      toast(t('var.copied', { name: `{{${name}}}` }), 'info');
    } catch {
      /* clipboard not available (http) */
    }
  }
  const cols = 'grid grid-cols-[minmax(12rem,1.4fr)_7rem_minmax(10rem,1.4fr)_minmax(8rem,1fr)_7rem] items-center gap-x-4';
</script>

<div class="surface overflow-x-auto rounded-xl border border-zinc-200 dark:border-zinc-800">
  <div class="min-w-[52rem]">
    <div class="{cols} border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
      <span>{t('var.col.variable')}</span>
      <span>{t('var.col.source')}</span>
      <span>{t('var.col.description')}</span>
      <span>{t('var.col.example')}</span>
      <span>{t('var.col.used')}</span>
    </div>
    {#each variables as v (v.source + v.name)}
      <div
        class="{cols} border-b border-zinc-100 px-4 py-2.5 text-sm last:border-b-0 dark:border-zinc-800/70 {editable(v) ? 'cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/40' : ''}"
        role="button"
        aria-disabled={!editable(v)}
        tabindex={editable(v) ? 0 : -1}
        onclick={() => editable(v) && onedit(v)}
        onkeydown={(e) => editable(v) && e.key === 'Enter' && onedit(v)}
        data-testid="variable-row">
        <span class="min-w-0">
          <button type="button" class="var-chip max-w-full truncate hover:ring-1 hover:ring-accent-400" onclick={(e) => copy(e, v.name)} title={t('var.copy')}>{`{{${v.name}}}`}</button>
        </span>
        <span><span class="badge {badgeCls(v.source)}">{t(`var.source.${v.source}` as Key)}</span></span>
        <span class="truncate">{v.desc_key ? t(v.desc_key as Key) : v.description || '–'}</span>
        <span class="truncate">
          {#if v.source === 'system'}
            {v.example || '–'}
          {:else if v.default_value}
            {v.default_value}
          {:else if v.source === 'custom'}
            <i class="text-zinc-400">{t('var.asked')}</i>
          {:else}
            <i class="text-zinc-400">{t('var.fromIntegration', { name: t(`var.source.${v.source}` as Key) })}</i>
          {/if}
        </span>
        <span class="text-zinc-500">{v.used_in ? t(v.used_in === 1 ? 'var.usedIn1' : 'var.usedIn', { n: v.used_in }) : '–'}</span>
      </div>
    {:else}
      <p class="px-4 py-4 text-sm text-zinc-400">{t('var.none')}</p>
    {/each}
  </div>
</div>
<p class="mt-3 text-xs text-zinc-500 in-[.has-bg]:text-white/80">{t('var.hint')}</p>
