<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, errorText, setCsrf } from '$lib/api';
  import { refreshSidebar, session } from '$lib/state.svelte';
  import { i18n, LANGS, setLang, t } from '$lib/i18n.svelte';
  import type { User } from '$lib/types';

  let email = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  $effect(() => {
    if (session.ready && session.user) goto('/');
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    busy = true;
    try {
      const res = await api.post<{ user: User; csrf_token: string; app_subtitle?: string }>('/auth/login', { email, password });
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

<svelte:head><title>{t('login.submit')} · ProjectList</title></svelte:head>

<div class="flex min-h-dvh flex-col items-center justify-center bg-zinc-50 px-4 dark:bg-zinc-950">
  <div class="w-full max-w-sm rounded-xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
    <div class="mb-7 flex flex-col items-center gap-3 text-center">
      <img src="/favicon.svg" alt="" width="40" height="40" />
      <h1 class="font-serif text-[26px] font-bold tracking-tight">ProjectList</h1>
      <p class="text-sm text-zinc-500">{t('login.subtitle')}</p>
    </div>
    <form onsubmit={submit} class="space-y-4">
      <div>
        <label class="label" for="email">{t('common.email')}</label>
        <input id="email" type="email" autocomplete="username" required bind:value={email} class="input" />
      </div>
      <div>
        <label class="label" for="password">{t('common.password')}</label>
        <input id="password" type="password" autocomplete="current-password" required bind:value={password} class="input" />
      </div>
      {#if error}<p class="text-sm text-red-600 dark:text-red-400">{error}</p>{/if}
      <button type="submit" class="btn-primary w-full py-2!" disabled={busy}>{busy ? t('login.submitting') : t('login.submit')}</button>
    </form>
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
