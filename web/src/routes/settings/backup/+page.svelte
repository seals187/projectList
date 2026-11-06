<script lang="ts">
  // Settings → Backup & restore (admin): download a consistent copy of the
  // database, or restore one. Restoring replaces all data; the server first
  // validates the file and keeps a safety copy of the current state
  // ("pre-restore-….db" in the data directory). Afterwards the app reloads.
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { isAdmin } from '$lib/state.svelte';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Icon from '$lib/components/Icon.svelte';

  $effect(() => {
    if (!isAdmin()) goto('/settings');
  });

  let input = $state<HTMLInputElement>();
  let file = $state<File | null>(null);
  let armed = $state(false); // second click confirms
  let busy = $state(false);
  let error = $state('');
  let done = $state<{ safety: string; relogin: boolean } | null>(null);

  const size = (n: number) => (n >= 1 << 20 ? `${(n / (1 << 20)).toFixed(1)} MB` : `${Math.max(1, Math.round(n / 1024))} KB`);

  function pick(e: Event) {
    file = (e.currentTarget as HTMLInputElement).files?.[0] ?? null;
    armed = false;
    error = '';
  }

  async function restore() {
    if (!file) return;
    if (!armed) {
      armed = true;
      return;
    }
    busy = true;
    error = '';
    try {
      const fd = new FormData();
      fd.append('file', file);
      const r = await api.post<{ safety_copy: string; relogin: boolean }>('/admin/restore', fd);
      done = { safety: r.safety_copy, relogin: r.relogin };
      // fresh start with the restored data (the session cookie was renewed by the server)
      setTimeout(() => location.assign(r.relogin ? '/login' : '/'), 2500);
    } catch (e) {
      error = errorText(e);
    } finally {
      busy = false;
      armed = false;
    }
  }

  function reset() {
    file = null;
    armed = false;
    error = '';
    if (input) input.value = '';
  }
</script>

<SettingsShell title={t('set.nav.backup')} subtitle={t('set.backup.subtitle')}>
  <div class="max-w-[680px]">
    <SetCard title={t('set.backup.download')}>
      <SetRow title={t('set.backup.db')} desc={t('settings.backupHint')}>
        <a href="/api/admin/backup" class="btn-outline" download><Icon name="download" size={14} /> {t('settings.backupDownload')}</a>
      </SetRow>
    </SetCard>

    <SetCard title={t('set.restore.title')}>
      <div class="px-[18px] py-3.5">
        <p class="text-[14.5px] font-semibold">{t('set.restore.row')}</p>
        <p class="mt-0.5 text-[13px] text-zinc-500 dark:text-zinc-400">{t('set.restore.desc')}</p>
        <p class="mt-3 flex items-start gap-2 rounded-lg bg-amber-50 px-3 py-2 text-[13px] text-amber-800 dark:bg-amber-950/40 dark:text-amber-300">
          <Icon name="alert" size={15} class="mt-px shrink-0" />{t('set.restore.warning')}
        </p>

        {#if done}
          <p class="mt-4 flex items-center gap-1.5 text-[13.5px] font-medium text-emerald-700 dark:text-emerald-400" data-testid="restore-done">
            <Icon name="check" size={15} />{t('set.restore.done', { file: done.safety })}
          </p>
          <p class="mt-1 text-[13px] text-zinc-500">{done.relogin ? t('set.restore.relogin') : t('set.restore.reloading')}</p>
        {:else}
          <div class="mt-4 flex flex-wrap items-center gap-2">
            <input bind:this={input} type="file" accept=".db,.sqlite,.sqlite3,application/vnd.sqlite3,application/octet-stream" class="hidden" onchange={pick} />
            <button type="button" class="btn-outline" onclick={() => input?.click()} disabled={busy}>
              <Icon name="database" size={14} />{file ? t('set.restore.other') : t('set.restore.choose')}
            </button>
            {#if file}
              <span class="min-w-0 truncate text-[13.5px]" title={file.name}><span class="font-medium">{file.name}</span> <span class="text-zinc-400">· {size(file.size)}</span></span>
              <span class="ml-auto flex gap-2">
                <button type="button" class="btn-ghost" onclick={reset} disabled={busy}>{t('common.cancel')}</button>
                <button type="button" class={armed ? 'btn bg-red-600 text-white hover:bg-red-700' : 'btn-danger'} onclick={restore} disabled={busy} data-testid="restore-btn">
                  {busy ? t('set.restore.busy') : armed ? t('set.restore.confirm') : t('set.restore.start')}
                </button>
              </span>
            {/if}
          </div>
          {#if error}<p class="mt-3 text-[13.5px] text-red-600 dark:text-red-400" data-testid="restore-error">{error}</p>{/if}
        {/if}
      </div>
    </SetCard>
  </div>
</SettingsShell>
