<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { checkRadius, CODE_BGS, isAdmin, OVERVIEW_VIEWS, setOverviewDefault, prefs, session, setPref, setTheme, stepZoom, theme, toast, ZOOM_STEPS, type CheckPos, type CheckShape, type Theme } from '$lib/state.svelte';
  import { i18n, LANGS, setLang, t } from '$lib/i18n.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SettingsTabs from '$lib/components/SettingsTabs.svelte';
  import { FONTS } from '$lib/fonts';
  import Avatar from '$lib/components/Avatar.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import BackgroundPicker from '$lib/components/BackgroundPicker.svelte';
  import { backgroundCss, setUserBackground, userBackground } from '$lib/backgrounds';

  let current = $state('');
  let next = $state('');
  let busy = $state(false);

  const themes: Theme[] = ['system', 'light', 'dark', 'dim'];
  const positions: CheckPos[] = ['right', 'left'];
  const shapes: CheckShape[] = ['square', 'round'];
  // Swatches of the code block backgrounds (as in app.css)
  const CODE_SWATCH = { light: 'bg-[#f1f2f4]', dark: 'bg-[#2b2d35]', black: 'bg-black' } as const;

  async function changePassword(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    try {
      await api.put('/auth/password', { current_password: current, new_password: next });
      current = next = '';
      toast(t('settings.passwordChanged'), 'info');
    } catch (err) {
      toast(errorText(err));
    } finally {
      busy = false;
    }
  }

  async function logout() {
    try {
      await api.post('/auth/logout');
    } finally {
      session.user = null;
      goto('/login');
    }
  }

  const segCls = (on: boolean) =>
    `rounded-md px-3 py-1.5 text-sm ${on ? 'bg-white font-medium shadow-xs dark:bg-zinc-700' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'}`;

  // own background (whole app; synced across devices)
  let bgOpen = $state(false);
  async function pickBg(value: string) {
    try {
      await setUserBackground(value);
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{t('nav.settings')} · ProjectList</title></svelte:head>

<PageHeader>
  {#snippet title()}{t('nav.settings')}{/snippet}
</PageHeader>
<SettingsTabs />

<div class="max-w-xl space-y-10">
  <section>
    <h2 class="section-label mb-3">{t('settings.language')}</h2>
    <div class="inline-flex gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
      {#each LANGS as l (l.code)}
        <button type="button" class={segCls(i18n.lang === l.code)} onclick={() => setLang(l.code)}>{l.label}</button>
      {/each}
    </div>
  </section>

  <section>
    <h2 class="section-label mb-3">{t('settings.theme')}</h2>
    <div class="inline-flex gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
      {#each themes as th (th)}
        <button type="button" class={segCls(theme.value === th)} onclick={() => setTheme(th)}>{t(`theme.${th}`)}</button>
      {/each}
    </div>
  </section>

  <section>
    <h2 class="section-label mb-1">{t('bg.title')}</h2>
    <p class="mb-3 text-sm text-zinc-500">{t('bg.userHint')}</p>
    <div class="flex items-center gap-3">
      <span class="h-14 w-24 shrink-0 rounded-lg border border-zinc-200 bg-zinc-50 bg-cover bg-center dark:border-zinc-700 dark:bg-zinc-900" style:background-image={backgroundCss(userBackground()) || undefined}></span>
      <button type="button" class="btn-outline" onclick={() => (bgOpen = true)}>{t('bg.choose')}</button>
      {#if userBackground()}
        <button type="button" class="btn-ghost" onclick={() => pickBg('')}>{t('bg.remove')}</button>
      {/if}
    </div>
  </section>

  <section>
    <h2 class="section-label mb-3">{t('settings.font')}</h2>
    <div class="grid gap-2 sm:grid-cols-2" role="radiogroup" aria-label={t('settings.font')}>
      {#each FONTS as f (f.id)}
        {@const on = prefs.font === f.id}
        <button
          type="button"
          role="radio"
          aria-checked={on}
          class="rounded-lg border px-3 py-2 text-left transition-colors {on
            ? 'border-accent-500 bg-accent-50 ring-1 ring-accent-500 dark:bg-accent-950/40'
            : 'border-zinc-200 hover:border-zinc-300 dark:border-zinc-800 dark:hover:border-zinc-700'}"
          style:font-family={f.family}
          onclick={() => setPref('font', f.id)}>
          <span class="flex items-baseline justify-between gap-2">
            <span class="truncate text-[15px] font-semibold">{f.label}</span>
            <span class="shrink-0 text-[10px] text-zinc-400" style:font-family="var(--font-sans)">{f.hint ?? t(`font.hint.${f.id}`)}</span>
          </span>
          <span class="mt-1 block truncate text-sm text-zinc-600 dark:text-zinc-300">{t('settings.fontSample')}</span>
        </button>
      {/each}
    </div>

    <div class="mt-4 flex items-center gap-4 text-sm">
      <span class="muted">{t('settings.headings')}</span>
      <div class="inline-flex w-fit gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
        {#each ['serif', 'sans'] as const as h (h)}
          <button type="button" class={segCls(prefs.headingFont === h)} onclick={() => setPref('headingFont', h)}>{t(`settings.heading.${h}`)}</button>
        {/each}
      </div>
    </div>

    <!-- Preview with the selected font -->
    <div class="card mt-4 p-4">
      <p class="mb-2 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('settings.fontPreview')}</p>
      <p class="font-serif text-xl font-bold tracking-tight">{t('settings.fontPreviewTitle')}</p>
      <p class="mt-1 text-sm text-zinc-600 dark:text-zinc-300">{t('settings.fontPreviewText')}</p>
      <p class="mt-2 font-mono text-xs text-zinc-500">systemctl restart nginx  # 0123456789</p>
    </div>
  </section>

  <section>
    <h2 class="section-label mb-3">{t('nav.overview')}</h2>
    <div class="flex flex-wrap items-center gap-x-4 gap-y-2 text-sm">
      <span class="muted">{t('settings.defaultView')}</span>
      <div class="inline-flex w-fit flex-wrap gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
        {#each ['last', ...OVERVIEW_VIEWS] as const as v (v)}
          <button type="button" class={segCls(prefs.overviewDefault === v)} onclick={() => setOverviewDefault(v)}>
            {v === 'last' ? t('settings.lastUsedView') : t(`overview.view.${v}`)}
          </button>
        {/each}
      </div>
    </div>
    <p class="mt-2 text-xs text-zinc-400">{t('settings.defaultViewHint')}</p>
  </section>

  <section>
    <h2 class="section-label mb-3">{t('settings.checklists')}</h2>
    <div class="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-3 text-sm">
      <span class="muted">{t('settings.checkPos')}</span>
      <div class="inline-flex w-fit gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
        {#each positions as p (p)}
          <button type="button" class={segCls(prefs.checkPos === p)} onclick={() => setPref('checkPos', p)}>{t(`settings.pos.${p}`)}</button>
        {/each}
      </div>
      <span class="muted">{t('settings.checkShape')}</span>
      <div class="inline-flex w-fit gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
        {#each shapes as s (s)}
          <button type="button" class={segCls(prefs.checkShape === s)} onclick={() => setPref('checkShape', s)}>{t(`settings.shape.${s}`)}</button>
        {/each}
      </div>
      <span class="muted">{t('settings.gridLines')}</span>
      <div class="inline-flex w-fit gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
        {#each [true, false] as on (on)}
          <button type="button" class={segCls(prefs.gridLines === on)} onclick={() => setPref('gridLines', on)}>{t(on ? 'settings.on' : 'settings.off')}</button>
        {/each}
      </div>
      <span class="muted">{t('blocks.code')}</span>
      <div class="inline-flex w-fit gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70">
        {#each CODE_BGS as bg (bg)}
          <button type="button" class="{segCls(prefs.codeBg === bg)} inline-flex items-center gap-1.5" onclick={() => setPref('codeBg', bg)}>
            <span class="size-3 rounded-full border border-zinc-400/70 {CODE_SWATCH[bg]}"></span>{t(`code.bg.${bg}`)}
          </button>
        {/each}
      </div>
      <span class="muted">{t('editor.size')}</span>
      <div class="flex items-center gap-2">
        <button type="button" class="btn-outline size-8 p-0!" onclick={() => stepZoom(-1)} disabled={prefs.zoom <= ZOOM_STEPS[0]} aria-label={t('editor.smaller')}><Icon name="minus" size={14} /></button>
        <span class="w-12 text-center tabular-nums">{Math.round(prefs.zoom * 100)}%</span>
        <button type="button" class="btn-outline size-8 p-0!" onclick={() => stepZoom(1)} disabled={prefs.zoom >= ZOOM_STEPS[ZOOM_STEPS.length - 1]} aria-label={t('editor.larger')}><Icon name="plus" size={14} /></button>
      </div>
    </div>

    <!-- Preview -->
    <div class="card mt-4 space-y-1.5 p-3" style:zoom={prefs.zoom === 1 ? undefined : prefs.zoom}>
      {#each [true, false] as done (done)}
        <div class="flex items-center gap-2 text-sm {prefs.checkPos === 'right' ? 'flex-row-reverse justify-end' : ''}">
          <span class="flex size-4 shrink-0 items-center justify-center border {checkRadius()} {done ? 'border-accent-600 bg-accent-600 text-white' : 'border-zinc-500 dark:border-zinc-400'}">
            {#if done}<Icon name="check" size={10} class="stroke-[3]" />{/if}
          </span>
          <span class="{prefs.checkPos === 'right' ? 'flex-1' : ''} {done ? 'text-zinc-600 line-through dark:text-zinc-400' : ''}">{t('settings.preview')}</span>
        </div>
      {/each}
    </div>
  </section>

  <section>
    <h2 class="section-label mb-3">{t('settings.account')}</h2>
    <div class="flex items-center gap-3">
      <Avatar name={session.user?.name ?? ''} id={session.user?.id ?? 0} size={36} />
      <div class="min-w-0 flex-1">
        <p class="font-medium">{session.user?.name}</p>
        <p class="text-sm text-zinc-500">{session.user?.email} · {t(`role.${session.user?.role ?? 'viewer'}`)}</p>
      </div>
      <button type="button" class="btn-outline" onclick={logout}><Icon name="logout" size={14} /> {t('nav.logout')}</button>
    </div>

    <form onsubmit={changePassword} class="mt-6 space-y-3">
      <p class="text-sm font-medium">{t('settings.password')}</p>
      <div>
        <label class="label" for="cur">{t('settings.currentPassword')}</label>
        <input id="cur" type="password" autocomplete="current-password" class="input" required bind:value={current} />
      </div>
      <div>
        <label class="label" for="new">{t('settings.newPassword')}</label>
        <input id="new" type="password" autocomplete="new-password" minlength="8" class="input" required bind:value={next} />
      </div>
      <button class="btn-primary" type="submit" disabled={busy}>{t('common.save')}</button>
    </form>
  </section>

  {#if isAdmin()}
    <section>
      <h2 class="section-label mb-3">{t('settings.backup')}</h2>
      <p class="mb-3 text-sm text-zinc-500">{t('settings.backupHint')}</p>
      <a href="/api/admin/backup" class="btn-outline" download><Icon name="download" size={14} /> {t('settings.backupDownload')}</a>
    </section>
  {/if}
</div>

{#if bgOpen}
  <BackgroundPicker title={t('bg.title')} value={userBackground()} noneLabel={t('bg.none')} onpick={pickBg} onclose={() => (bgOpen = false)} />
{/if}
