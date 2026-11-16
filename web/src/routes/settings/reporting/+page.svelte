<script lang="ts">
  // Settings → Reporting (admin): which views are available and which one opens,
  // selectable periods and the default, how progress / due dates / "due soon" /
  // done projects / customer progress / overload are calculated, the dashboard
  // key figures (selection and order), and the scheduled report e-mails.
  // Every change is saved right away and applies to everyone – reporting page,
  // PDF and e-mails (internal/report/config.go).
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { t, type Key } from '$lib/i18n.svelte';
  import { isAdmin, toast } from '$lib/state.svelte';
  import { moveItem, type ReportConfig, type ReportConfigResp, type ReportKpi, type ReportSchedule, type ReportView } from '$lib/reporting';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Seg from '$lib/components/Seg.svelte';
  import Switch from '$lib/components/Switch.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Icon from '$lib/components/Icon.svelte';

  $effect(() => {
    if (!isAdmin()) goto('/settings');
  });

  let cfg = $state<ReportConfig | null>(null);
  let defaults = $state<ReportConfig | null>(null);
  let choices = $state<ReportConfigResp['choices'] | null>(null);
  let schedules = $state<ReportSchedule[]>([]);
  let mailEnabled = $state(true);

  async function load() {
    try {
      const r = await api.get<ReportConfigResp>('/reports/config');
      cfg = r.config;
      defaults = r.defaults;
      choices = r.choices;
    } catch (e) {
      toast(errorText(e));
    }
  }
  async function loadSchedules() {
    try {
      const r = await api.get<{ schedules: ReportSchedule[]; mail_enabled: boolean }>('/report-schedules');
      schedules = r.schedules;
      mailEnabled = r.mail_enabled;
    } catch (e) {
      toast(errorText(e));
    }
  }
  load();
  loadSchedules();

  // save the whole settings after every change; on an error the stored state comes back
  async function save(next: ReportConfig) {
    const prev = cfg;
    cfg = next;
    try {
      cfg = (await api.put<ReportConfigResp>('/reports/config', next)).config;
    } catch (e) {
      cfg = prev;
      toast(errorText(e));
    }
  }
  const patch = (p: Partial<ReportConfig>) => cfg && save({ ...cfg, ...p });

  const viewLabel = (v: ReportView) => t(`rep.view.${v}`);
  const kpiLabel = (k: ReportKpi) => t(({ completed: 'rep.kpi.completed', open: 'rep.kpi.open', overdue: 'rep.kpi.overdue', projects_done: 'rep.kpi.projectsDone' } as const)[k]);
  const periodLabel = (p: number) => t(`rep.period.${p}` as Key);

  // views: keep the order of the choices; at least one; the default view stays enabled
  function toggleView(v: ReportView) {
    if (!cfg || !choices) return;
    const on = cfg.views.includes(v);
    if (on && cfg.views.length === 1) return;
    const views = choices.views.filter((x) => (x === v ? !on : cfg!.views.includes(x)));
    patch({ views, default_view: views.includes(cfg.default_view) ? cfg.default_view : views[0] });
  }
  function togglePeriod(p: number) {
    if (!cfg) return;
    const on = cfg.periods.includes(p);
    if (on && cfg.periods.length === 1) return;
    const periods = on ? cfg.periods.filter((x) => x !== p) : [...cfg.periods, p].sort((a, b) => a - b);
    patch({ periods, default_period: periods.includes(cfg.default_period) ? cfg.default_period : periods[0] });
  }
  function toggleKpi(k: ReportKpi) {
    if (!cfg) return;
    patch({ kpis: cfg.kpis.includes(k) ? cfg.kpis.filter((x) => x !== k) : [...cfg.kpis, k] });
  }
  // KPI list: enabled ones in their order, then the disabled ones
  const kpiRows = $derived(cfg && choices ? [...cfg.kpis, ...choices.kpis.filter((k) => !cfg!.kpis.includes(k))] : []);

  // number fields: saved on leaving the field, kept within the limits
  function setNumber(field: 'due_soon_days' | 'overload_threshold', raw: string, min: number, max: number) {
    if (!cfg) return;
    const n = Math.round(Number(raw));
    if (!Number.isFinite(n)) return;
    const v = Math.min(max, Math.max(min, n));
    if (v !== cfg[field]) patch({ [field]: v });
  }

  async function sendNow(s: ReportSchedule) {
    try {
      await api.post(`/report-schedules/${s.id}/send`, {});
      toast(t('sched.sent', { name: s.recipient_name }), 'info');
    } catch (err) {
      toast(errorText(err));
    }
  }
  async function removeSchedule(s: ReportSchedule) {
    try {
      await api.del(`/report-schedules/${s.id}`);
      loadSchedules();
    } catch (err) {
      toast(errorText(err));
    }
  }
  const when = (s: ReportSchedule) => (s.freq === 'weekly' ? t('auto.every.weekly', { day: t(`weekday.${s.weekday}` as Key) }) : t('auto.every.monthly', { day: String(s.day) }));

  const chip = (on: boolean) =>
    `inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-[13px] font-medium transition-colors ${
      on
        ? 'border-accent-600 bg-accent-50 text-accent-800 dark:border-accent-400 dark:bg-accent-950/50 dark:text-accent-200'
        : 'border-zinc-200 text-zinc-500 hover:border-zinc-300 hover:text-zinc-800 dark:border-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-100'
    }`;
</script>

<SettingsShell title={t('set.nav.reporting')} subtitle={t('set.rep.subtitle')}>
  {#if cfg && choices}
    {@const c = cfg}
    <div class="max-w-[760px]" data-testid="settings-reporting">
      <!-- views -->
      <SetCard title={t('set.rep.views')}>
        <SetRow title={t('set.rep.views')} desc={t('set.rep.viewsDesc')}>
          {#each choices.views as v (v)}
            {@const on = c.views.includes(v)}
            <button type="button" class={chip(on)} aria-pressed={on} disabled={on && c.views.length === 1} onclick={() => toggleView(v)} data-testid="set-rep-view-{v}">
              {#if on}<Icon name="check" size={13} />{/if}{viewLabel(v)}
            </button>
          {/each}
        </SetRow>
        <SetRow title={t('set.rep.defaultView')} desc={t('set.rep.defaultViewDesc')}>
          <Seg label={t('set.rep.defaultView')} options={c.views.map((v) => ({ value: v, label: viewLabel(v) }))} value={c.default_view} onchange={(v) => patch({ default_view: v })} />
        </SetRow>
      </SetCard>

      <!-- period -->
      <SetCard title={t('set.rep.period')}>
        <SetRow title={t('set.rep.periods')} desc={t('set.rep.periodsDesc')}>
          <div class="flex max-w-md flex-wrap gap-1.5">
            {#each choices.periods as p (p)}
              {@const on = c.periods.includes(p)}
              <button type="button" class={chip(on)} aria-pressed={on} disabled={on && c.periods.length === 1} onclick={() => togglePeriod(p)}>
                {#if on}<Icon name="check" size={13} />{/if}{periodLabel(p)}
              </button>
            {/each}
          </div>
        </SetRow>
        <SetRow title={t('set.rep.defaultPeriod')} desc={t('set.rep.defaultPeriodDesc')} labelFor="rep-default-period">
          <select id="rep-default-period" class="input w-auto!" value={c.default_period} onchange={(e) => patch({ default_period: Number(e.currentTarget.value) })}>
            {#each c.periods as p (p)}<option value={p}>{periodLabel(p)}</option>{/each}
          </select>
        </SetRow>
      </SetCard>

      <!-- calculation -->
      <SetCard title={t('set.rep.calc')}>
        <SetRow title={t('set.rep.progress')} desc={t('set.rep.progressDesc')}>
          <Seg
            label={t('set.rep.progress')}
            options={[
              { value: 'checklists' as const, label: t('set.rep.progress.checklists') },
              { value: 'items' as const, label: t('set.rep.progress.items') }
            ]}
            value={c.progress_basis}
            onchange={(v) => patch({ progress_basis: v })} />
        </SetRow>
        <SetRow title={t('set.rep.due')} desc={t('set.rep.dueDesc')}>
          <Seg
            label={t('set.rep.due')}
            options={[
              { value: 'checklist' as const, label: t('set.rep.due.checklist') },
              { value: 'item' as const, label: t('set.rep.due.item') }
            ]}
            value={c.due_basis}
            onchange={(v) => patch({ due_basis: v })} />
        </SetRow>
        <SetRow title={t('set.rep.dueSoon')} desc={t('set.rep.dueSoonDesc')} labelFor="rep-due-soon">
          <input
            id="rep-due-soon"
            type="number"
            min="1"
            max="90"
            class="input w-20! tabular-nums"
            value={c.due_soon_days}
            onchange={(e) => setNumber('due_soon_days', e.currentTarget.value, 1, 90)} />
          <span class="text-sm text-zinc-500">{t('set.rep.days')}</span>
        </SetRow>
        <SetRow title={t('set.rep.includeDone')} desc={t('set.rep.includeDoneDesc')}>
          <Switch label={t('set.rep.includeDone')} checked={c.include_done} onchange={(v) => patch({ include_done: v })} />
        </SetRow>
        <SetRow title={t('set.rep.respect')} desc={t('set.rep.respectDesc')}>
          <Switch label={t('set.rep.respect')} checked={c.respect_customer_progress} onchange={(v) => patch({ respect_customer_progress: v })} />
        </SetRow>
        <SetRow title={t('set.rep.overload')} desc={t('set.rep.overloadDesc')} labelFor="rep-overload">
          <input
            id="rep-overload"
            type="number"
            min="0"
            max="999"
            class="input w-20! tabular-nums"
            value={c.overload_threshold}
            onchange={(e) => setNumber('overload_threshold', e.currentTarget.value, 0, 999)} />
          <span class="text-sm text-zinc-500">{t('set.rep.checklists')}</span>
        </SetRow>
      </SetCard>

      <!-- dashboard key figures: on/off and order -->
      <SetCard title={t('set.rep.kpis')}>
        <p class="px-[18px] pt-3 text-[13px] text-zinc-500 dark:text-zinc-400">{t('set.rep.kpisDesc')}</p>
        {#each kpiRows as k (k)}
          {@const on = c.kpis.includes(k)}
          {@const i = c.kpis.indexOf(k)}
          <div class="flex items-center gap-3 px-[18px] py-2.5" data-testid="set-rep-kpi">
            <Switch label={kpiLabel(k)} checked={on} onchange={() => toggleKpi(k)} />
            <span class="flex-1 text-[14.5px] {on ? 'font-semibold' : 'text-zinc-400'}">{kpiLabel(k)}</span>
            {#if on}
              <button type="button" class="btn-ghost px-1.5!" disabled={i === 0} onclick={() => patch({ kpis: moveItem(c.kpis, i, -1) })} title={t('set.rep.moveUp')} aria-label="{t('set.rep.moveUp')}: {kpiLabel(k)}">
                <Icon name="chevronDown" size={15} class="rotate-180" />
              </button>
              <button
                type="button"
                class="btn-ghost px-1.5!"
                disabled={i === c.kpis.length - 1}
                onclick={() => patch({ kpis: moveItem(c.kpis, i, 1) })}
                title={t('set.rep.moveDown')}
                aria-label="{t('set.rep.moveDown')}: {kpiLabel(k)}">
                <Icon name="chevronDown" size={15} />
              </button>
            {/if}
          </div>
        {/each}
      </SetCard>

      <!-- report e-mails -->
      <SetCard title={t('set.rep.mail')}>
        <p class="px-[18px] pt-3 pb-1 text-[13px] text-zinc-500 dark:text-zinc-400">{t('set.rep.mailDesc')}</p>
        {#if !mailEnabled}
          <p class="mx-[18px] my-2 flex gap-2 rounded-lg bg-amber-50 p-2.5 text-[13px] text-amber-800 dark:bg-amber-950/40 dark:text-amber-300"><Icon name="alert" size={15} class="mt-0.5 shrink-0" />{t('sched.noMail')}</p>
        {/if}
        {#each schedules as s (s.id)}
          <div class="flex items-center gap-2 px-[18px] py-2.5 text-sm" data-testid="set-rep-schedule">
            <Icon name="mail" size={15} class="shrink-0 text-zinc-400" />
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium">{s.recipient_name} · {when(s)}</span>
              <span class="block truncate text-xs text-zinc-500">{periodLabel(s.filters.period)}{s.last_sent_on ? ` · ${t('sched.lastSent', { date: s.last_sent_on })}` : ''}</span>
            </span>
            <button type="button" class="btn-ghost px-2! py-1! text-xs" onclick={() => sendNow(s)} disabled={!mailEnabled}>{t('sched.sendNow')}</button>
            <button type="button" class="btn-ghost px-1.5! text-zinc-400 hover:text-red-600!" onclick={() => removeSchedule(s)} aria-label={t('common.delete')}><Icon name="trash" size={14} /></button>
          </div>
        {:else}
          <p class="px-[18px] py-3 text-sm text-zinc-400">{t('set.rep.mailNone')}</p>
        {/each}
        <div class="px-[18px] py-3">
          <a href="/reports" class="btn-outline"><Icon name="chart" size={14} />{t('set.rep.mailOpen')}</a>
        </div>
      </SetCard>

      {#if defaults}
        {@const d = defaults}
        <ConfirmButton class="btn-outline" onconfirm={() => save(d)} label={t('set.rep.reset')} confirmLabel={t('set.rep.resetConfirm')}>
          <Icon name="undo" size={14} />{t('set.rep.reset')}
        </ConfirmButton>
      {/if}
    </div>
  {:else}
    <p class="text-sm text-zinc-400">{t('common.loading')}</p>
  {/if}
</SettingsShell>
