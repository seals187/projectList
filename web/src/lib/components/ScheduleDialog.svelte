<script lang="ts">
  // Report e-mails: list of the schedules (send now, delete) and a new schedule
  // with the current filters (weekly on a weekday or monthly on a day).
  // Admins choose the recipient, members send to themselves.
  import { untrack } from 'svelte';
  import { api, errorText } from '$lib/api';
  import { session, toast } from '$lib/state.svelte';
  import { i18n, t, type Key } from '$lib/i18n.svelte';
  import type { ReportFilter, ReportSchedule } from '$lib/reporting';
  import Icon from './Icon.svelte';

  let { filter, onclose }: { filter: ReportFilter; onclose: () => void } = $props();

  let dialog = $state<HTMLDialogElement>();
  let schedules = $state<ReportSchedule[]>([]);
  let recipients = $state<{ id: number; name: string }[]>([]);
  let mailEnabled = $state(true);
  let recipient = $state(session.user?.id ?? 0);
  let freq = $state<'weekly' | 'monthly'>('weekly');
  let weekday = $state(1);
  let day = $state(1);
  let busy = $state(false);
  const current = untrack(() => ({ ...filter }));

  async function load() {
    try {
      const r = await api.get<{ schedules: ReportSchedule[]; recipients: { id: number; name: string }[]; mail_enabled: boolean }>('/report-schedules');
      schedules = r.schedules;
      recipients = r.recipients;
      mailEnabled = r.mail_enabled;
    } catch (e) {
      toast(errorText(e));
    }
  }
  $effect(() => {
    dialog?.showModal();
    load();
  });

  async function add(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    try {
      await api.post('/report-schedules', { recipient_id: recipient, freq, weekday, day, filters: current, lang: i18n.lang });
      toast(t('sched.saved'), 'info');
      load();
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }
  async function sendNow(s: ReportSchedule) {
    try {
      await api.post(`/report-schedules/${s.id}/send`, {});
      toast(t('sched.sent', { name: s.recipient_name }), 'info');
    } catch (err) {
      toast(errorText(err));
    }
  }
  async function remove(s: ReportSchedule) {
    try {
      await api.del(`/report-schedules/${s.id}`);
      load();
    } catch (err) {
      toast(errorText(err));
    }
  }
  const when = (s: Pick<ReportSchedule, 'freq' | 'weekday' | 'day'>) =>
    s.freq === 'weekly' ? t('auto.every.weekly', { day: t(`weekday.${s.weekday}` as Key) }) : t('auto.every.monthly', { day: String(s.day) });
  const period = (n: number) => t(`rep.period.${n}` as Key);
</script>

<dialog
  bind:this={dialog}
  {onclose}
  onclick={(e) => e.target === dialog && dialog?.close()}
  class="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
  data-testid="schedule-dialog">
  <div class="space-y-4 p-5">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold">{t('sched.title')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.close')}><Icon name="x" size={16} /></button>
    </div>
    {#if !mailEnabled}
      <p class="flex gap-2 rounded-lg bg-amber-50 p-2.5 text-[13px] text-amber-800 dark:bg-amber-950/40 dark:text-amber-300"><Icon name="alert" size={15} class="mt-0.5 shrink-0" />{t('sched.noMail')}</p>
    {/if}

    {#if schedules.length}
      <ul class="divide-y divide-zinc-100 rounded-lg border border-zinc-200 text-sm dark:divide-zinc-800 dark:border-zinc-700">
        {#each schedules as s (s.id)}
          <li class="flex items-center gap-2 px-3 py-2">
            <Icon name="mail" size={15} class="shrink-0 text-zinc-400" />
            <span class="min-w-0 flex-1">
              <span class="block truncate font-medium">{s.recipient_name} · {when(s)}</span>
              <span class="block truncate text-xs text-zinc-500">{period(s.filters.period)}{s.last_sent_on ? ` · ${t('sched.lastSent', { date: s.last_sent_on })}` : ''}</span>
            </span>
            <button type="button" class="btn-ghost px-2! py-1! text-xs" onclick={() => sendNow(s)} disabled={!mailEnabled}>{t('sched.sendNow')}</button>
            <button type="button" class="btn-ghost px-1.5! text-zinc-400 hover:text-red-600!" onclick={() => remove(s)} aria-label={t('common.delete')}><Icon name="trash" size={14} /></button>
          </li>
        {/each}
      </ul>
    {/if}

    <form onsubmit={add} class="space-y-3 rounded-lg bg-zinc-50 p-3 dark:bg-zinc-800/50">
      <p class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('sched.new')}</p>
      {#if recipients.length > 1}
        <label class="block">
          <span class="label">{t('sched.recipient')}</span>
          <select class="input" bind:value={recipient}>
            {#each recipients as r (r.id)}<option value={r.id}>{r.name}</option>{/each}
          </select>
        </label>
      {/if}
      <div class="flex flex-wrap gap-2">
        <select class="input w-auto!" bind:value={freq} aria-label={t('auto.freq')}>
          <option value="weekly">{t('auto.freq.weekly')}</option>
          <option value="monthly">{t('auto.freq.monthly')}</option>
        </select>
        {#if freq === 'weekly'}
          <select class="input w-auto!" bind:value={weekday} aria-label={t('auto.weekday')}>
            {#each [1, 2, 3, 4, 5, 6, 7] as d (d)}<option value={d}>{t(`weekday.${d}` as Key)}</option>{/each}
          </select>
        {:else}
          <select class="input w-auto!" bind:value={day} aria-label={t('auto.day')}>
            {#each Array.from({ length: 31 }, (_, i) => i + 1) as d (d)}<option value={d}>{d}.</option>{/each}
          </select>
        {/if}
      </div>
      <p class="text-xs text-zinc-500">{t('sched.filters', { period: period(current.period) })}</p>
      <div class="flex justify-end">
        <button type="submit" class="btn-primary" disabled={busy}><Icon name="plus" size={14} />{t('sched.add')}</button>
      </div>
    </form>
  </div>
</dialog>
