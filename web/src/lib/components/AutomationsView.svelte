<script lang="ts">
  // Templates page, view "Automations": rules (trigger → template, runs,
  // on/off) and the latest runs. Admins edit (click on a rule, switch).
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { locale, t } from '$lib/i18n.svelte';
  import { triggerDetail, triggerIcon, triggerTitle } from '$lib/automationText';
  import { categoryHex } from '$lib/templateLibrary';
  import type { AutomationRule, AutomationRun, TemplateCategory, TemplateSummary } from '$lib/types';
  import Icon from './Icon.svelte';
  import Switch from './Switch.svelte';

  let {
    rules,
    runs,
    templates,
    categories,
    admin,
    onedit,
    onchange
  }: {
    rules: AutomationRule[];
    runs: AutomationRun[];
    templates: TemplateSummary[];
    categories: TemplateCategory[];
    admin: boolean;
    onedit: (r: AutomationRule) => void;
    onchange: () => void;
  } = $props();

  function templateColor(id: number) {
    const cat = templates.find((x) => x.id === id)?.category_id;
    return categoryHex(categories.find((c) => c.id === cat));
  }

  async function toggle(r: AutomationRule, on: boolean) {
    r.enabled = on; // optimistic
    try {
      await api.patch(`/automations/${r.id}`, { enabled: on });
      onchange();
    } catch (e) {
      r.enabled = !on;
      toast(errorText(e));
    }
  }

  function when(iso: string): string {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    const time = d.toLocaleTimeString(locale(), { hour: '2-digit', minute: '2-digit' });
    const day = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const diff = Math.round((day(new Date()) - day(d)) / 86_400_000);
    if (diff === 0) return `${t('due.today')} ${time}`;
    if (diff === 1) return `${t('due.yesterday')} ${time}`;
    const p = (n: number) => String(n).padStart(2, '0');
    return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()}`;
  }
  function runText(r: AutomationRun): string {
    if (r.code === 'created') return t('auto.run.created', { name: r.template_name, target: r.target }) + (r.detail ? ` (${r.detail})` : '');
    if (r.code === 'no_project') return t('auto.run.noProject', { name: r.template_name }) + (r.detail ? `: ${r.detail}` : '');
    return t('auto.run.failed', { name: r.template_name, detail: r.detail });
  }
  const dotCls = { ok: 'bg-emerald-500', warning: 'bg-amber-500', error: 'bg-red-500' } as const;
</script>

<div class="surface overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800">
  <div class="flex items-center gap-1.5 border-b border-zinc-200 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase dark:border-zinc-800">
    {t('auto.col.trigger')} <Icon name="arrowRight" size={11} /> {t('auto.col.template')}
  </div>
  {#each rules as r (r.id)}
    <div
      class="grid grid-cols-[minmax(0,1fr)_1.5rem_minmax(0,1fr)_auto_auto] items-center gap-x-4 border-b border-zinc-100 px-4 py-3 last:border-b-0 dark:border-zinc-800/70 {admin ? 'cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/40' : ''}"
      role="button"
      aria-disabled={!admin}
      tabindex={admin ? 0 : -1}
      onclick={() => admin && onedit(r)}
      onkeydown={(e) => admin && e.key === 'Enter' && onedit(r)}
      data-testid="rule-row">
      <div class="flex min-w-0 items-center gap-3">
        <span class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"><Icon name={triggerIcon(r.trigger_kind)} size={17} /></span>
        <div class="min-w-0">
          <p class="truncate font-semibold">{triggerTitle(r)}</p>
          <p class="truncate text-[13px] text-zinc-500">{triggerDetail(r)}</p>
        </div>
      </div>
      <Icon name="arrowRight" size={16} class="text-zinc-400" />
      <div class="flex min-w-0 items-center gap-3">
        <span class="tpl-icon" style:--c={templateColor(r.template_id)}><Icon name="fileText" size={16} /></span>
        <div class="min-w-0">
          <p class="truncate font-semibold">{r.template_name}</p>
          <p class="truncate text-[13px] text-zinc-500">{r.target_name ? t('auto.action.in', { target: r.target_name }) : t('auto.action.create')}</p>
        </div>
      </div>
      <span class="hidden w-28 text-right text-[13px] text-zinc-500 sm:block">{r.enabled ? t('auto.runs', { n: r.run_count }) : t('auto.paused')}</span>
      <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
      <span onclick={(e) => e.stopPropagation()}>
        {#if admin}
          <Switch label={t('auto.enabled')} checked={r.enabled} onchange={(v) => toggle(r, v)} />
        {:else}
          <span class="badge {r.enabled ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400' : 'bg-zinc-100 text-zinc-500 dark:bg-zinc-800'}">{t(r.enabled ? 'auto.active' : 'auto.paused')}</span>
        {/if}
      </span>
    </div>
  {:else}
    <div class="px-4 py-8 text-center text-sm text-zinc-400">
      <p>{t('auto.empty')}</p>
      <p class="mt-1 text-xs">{t('auto.emptyHint')}</p>
    </div>
  {/each}
</div>

<section class="surface mt-4 rounded-xl border border-zinc-200 px-4 py-3 dark:border-zinc-800" data-testid="automation-runs">
  <h2 class="mb-1 font-semibold">{t('auto.recent')}</h2>
  <ul class="divide-y divide-zinc-100 dark:divide-zinc-800">
    {#each runs as run (run.id)}
      <li class="flex items-center gap-3 py-2 text-sm">
        <span class="size-2 shrink-0 rounded-full {dotCls[run.status] ?? 'bg-zinc-400'}"></span>
        <span class="min-w-0 flex-1 truncate">{runText(run)}</span>
        <span class="shrink-0 text-xs text-zinc-400 tabular-nums">{when(run.created_at)}</span>
      </li>
    {:else}
      <li class="py-2 text-sm text-zinc-400">{t('auto.noRuns')}</li>
    {/each}
  </ul>
</section>
