<script lang="ts">
  // Public page of invitation and password-reset links (/invite?token=…):
  // shows who the link is for, sets the password and signs the user in.
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { api, errorText, setCsrf } from '$lib/api';
  import { refreshSidebar, session } from '$lib/state.svelte';
  import { i18n, LANGS, setLang, t } from '$lib/i18n.svelte';
  import type { User } from '$lib/types';

  const token = $derived(page.url.searchParams.get('token') ?? '');
  let info = $state<{ name: string; email: string; purpose: 'invite' | 'reset' } | null>(null);
  let invalid = $state(false);
  let password = $state('');
  let repeat = $state('');
  let error = $state('');
  let busy = $state(false);

  $effect(() => {
    if (!token) {
      invalid = true;
      return;
    }
    api.get<{ name: string; email: string; purpose: 'invite' | 'reset' }>(`/auth/token/${encodeURIComponent(token)}`).then(
      (r) => (info = r),
      () => (invalid = true)
    );
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    if (password !== repeat) {
      error = t('invite.mismatch');
      return;
    }
    busy = true;
    try {
      const res = await api.post<{ user: User; csrf_token: string; app_subtitle?: string }>('/auth/token', { token, password });
      setCsrf(res.csrf_token);
      session.user = res.user;
      session.subtitle = res.app_subtitle ?? '';
      refreshSidebar(0);
      goto('/');
    } catch (err) {
      error = errorText(err);
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>{t('invite.title')} · ProjectList</title></svelte:head>

<div class="flex min-h-dvh flex-col items-center justify-center bg-zinc-50 px-4 dark:bg-zinc-950">
  <div class="w-full max-w-sm rounded-xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
    <div class="mb-7 flex flex-col items-center gap-3 text-center">
      <img src="/favicon.svg" alt="" width="40" height="40" />
      <h1 class="font-serif text-[26px] font-bold tracking-tight">ProjectList</h1>
      {#if info}
        <p class="text-sm text-zinc-500">{info.purpose === 'invite' ? t('invite.welcome', { name: info.name }) : t('invite.reset', { name: info.name })}</p>
      {/if}
    </div>
    {#if invalid}
      <p class="text-center text-sm text-red-600 dark:text-red-400" data-testid="invite-invalid">{t('invite.invalid')}</p>
      <a href="/login" class="btn-outline mt-5 w-full justify-center">{t('login.submit')}</a>
    {:else if info}
      <form onsubmit={submit} class="space-y-4">
        <input type="email" class="hidden" autocomplete="username" value={info.email} readonly aria-hidden="true" tabindex="-1" />
        <div>
          <label class="label" for="pw">{t('invite.password')}</label>
          <!-- svelte-ignore a11y_autofocus -->
          <input id="pw" type="password" autocomplete="new-password" minlength="8" required bind:value={password} class="input" autofocus />
        </div>
        <div>
          <label class="label" for="pw2">{t('invite.repeat')}</label>
          <input id="pw2" type="password" autocomplete="new-password" minlength="8" required bind:value={repeat} class="input" />
        </div>
        {#if error}<p class="text-sm text-red-600 dark:text-red-400">{error}</p>{/if}
        <button type="submit" class="btn-primary w-full py-2!" disabled={busy || password.length < 8}>{info.purpose === 'invite' ? t('invite.submitInvite') : t('invite.submitReset')}</button>
      </form>
    {/if}
  </div>
  <div class="mt-6 flex gap-1 text-xs">
    {#each LANGS as l (l.code)}
      <button
        type="button"
        onclick={() => setLang(l.code)}
        class="rounded px-2 py-1 {i18n.lang === l.code ? 'font-semibold text-accent-600' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}"
        >{l.code.toUpperCase()}</button>
    {/each}
  </div>
</div>
