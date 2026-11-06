<script lang="ts">
  // Settings → Appearance: language, color scheme, accent color, background
  // (image, glass effect, darken), density and app font size. Everything is
  // applied immediately and synced to all devices of the user (pref "ui").
  import { errorText } from '$lib/api';
  import { i18n, LANGS, setLang, t } from '$lib/i18n.svelte';
  import { prefs, setPref, setTheme, theme, toast } from '$lib/state.svelte';
  import { ACCENT_SWATCH, ACCENTS, DENSITIES, DIM_CONTENT, SCALE_STEPS, stepScale, THEME_CHOICES } from '$lib/appearance';
  import { pushUi } from '$lib/uiSync.svelte';
  import { backgroundCss, setUserBackground, userBackground } from '$lib/backgrounds';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Seg from '$lib/components/Seg.svelte';
  import Switch from '$lib/components/Switch.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import BackgroundPicker from '$lib/components/BackgroundPicker.svelte';

  const THEME_ICON = { light: 'sun', dark: 'moon', system: 'monitor' } as const;

  let bgOpen = $state(false);
  async function pickBg(value: string) {
    try {
      await setUserBackground(value);
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<SettingsShell title={t('set.nav.appearance')} subtitle={t('set.appearance.subtitle')}>
  <div class="max-w-[680px]">
    <SetCard title={t('set.general')}>
      <SetRow title={t('settings.language')} desc={t('set.language.desc')}>
        <Seg
          label={t('settings.language')}
          options={LANGS.map((l) => ({ value: l.code, label: l.label }))}
          value={i18n.lang}
          onchange={(v) => {
            setLang(v);
            pushUi();
          }} />
      </SetRow>
      <SetRow title={t('set.colorScheme')} desc={t('set.colorScheme.desc')}>
        <Seg label={t('set.colorScheme')} options={THEME_CHOICES.map((v) => ({ value: v, label: t(`theme.${v}`), icon: THEME_ICON[v] }))} value={theme.value} onchange={(v) => setTheme(v)} />
      </SetRow>
      <SetRow title={t('set.accent')} desc={t('set.accent.desc')}>
        <div class="flex items-center gap-2" role="radiogroup" aria-label={t('set.accent')}>
          {#each ACCENTS as a (a)}
            {@const on = prefs.accent === a}
            <button
              type="button"
              role="radio"
              aria-checked={on}
              aria-label={t(`set.accent.${a}`)}
              title={t(`set.accent.${a}`)}
              class="size-8 rounded-full transition-transform hover:scale-110 {on ? 'ring-2 ring-offset-2 ring-offset-white dark:ring-offset-zinc-900' : ''}"
              style:background-color={ACCENT_SWATCH[a]}
              style:--tw-ring-color={ACCENT_SWATCH[a]}
              onclick={() => setPref('accent', a)}></button>
          {/each}
        </div>
      </SetRow>
    </SetCard>

    <SetCard title={t('bg.title')}>
      <SetRow title={t('set.bgImage')} desc={t('set.bgImage.desc')}>
        <span
          class="h-[46px] w-[78px] rounded-lg border border-zinc-200 bg-zinc-100 bg-cover bg-center dark:border-zinc-700 dark:bg-zinc-800"
          style:background-image={backgroundCss(userBackground()) || undefined}
          aria-hidden="true"></span>
        <button type="button" class="btn-outline" onclick={() => (bgOpen = true)}>{t('set.change')}</button>
        {#if userBackground()}
          <button type="button" class="btn-ghost text-zinc-500" onclick={() => pickBg('')}>{t('bg.remove')}</button>
        {/if}
      </SetRow>
      <SetRow title={t('set.glass')} desc={t('set.glass.desc')}>
        <Switch label={t('set.glass')} checked={prefs.glass} onchange={(v) => setPref('glass', v)} />
      </SetRow>
      <SetRow title={t('set.dimBg')} desc={t('set.dimBg.desc')}>
        <Switch label={t('set.dimBg')} checked={prefs.dimBg} onchange={(v) => setPref('dimBg', v)} />
      </SetRow>
      <SetRow title={t('set.dimContent')} desc={t('set.dimContent.desc')}>
        <Seg label={t('set.dimContent')} options={DIM_CONTENT.map((d) => ({ value: d, label: t(`set.dimContent.${d}`) }))} value={prefs.dimContent} onchange={(v) => setPref('dimContent', v)} />
      </SetRow>
    </SetCard>

    <SetCard title={t('set.density')}>
      <SetRow title={t('set.rowSpacing')} desc={t('set.rowSpacing.desc')}>
        <Seg label={t('set.rowSpacing')} options={DENSITIES.map((d) => ({ value: d, label: t(`set.density.${d}`) }))} value={prefs.density} onchange={(v) => setPref('density', v)} />
      </SetRow>
      <SetRow title={t('set.fontSize')} desc={t('set.fontSize.desc')}>
        <button type="button" class="btn-outline size-9 p-0!" onclick={() => setPref('scale', stepScale(prefs.scale, -1))} disabled={prefs.scale <= SCALE_STEPS[0]} aria-label={t('editor.smaller')}>
          <Icon name="minus" size={14} />
        </button>
        <button type="button" class="w-16 text-center text-[15px] font-semibold tabular-nums" onclick={() => setPref('scale', 1)} title={t('editor.resetSize')}>
          {Math.round(prefs.scale * 100)} %
        </button>
        <button type="button" class="btn-outline size-9 p-0!" onclick={() => setPref('scale', stepScale(prefs.scale, 1))} disabled={prefs.scale >= SCALE_STEPS[SCALE_STEPS.length - 1]} aria-label={t('editor.larger')}>
          <Icon name="plus" size={14} />
        </button>
      </SetRow>
    </SetCard>
  </div>

  {#snippet footer()}
    <p class="flex items-center gap-1.5 text-[13.5px] font-medium text-emerald-700 dark:text-emerald-400"><Icon name="check" size={15} />{t('set.autoSaved')}</p>
  {/snippet}
</SettingsShell>

{#if bgOpen}
  <BackgroundPicker title={t('bg.title')} value={userBackground()} noneLabel={t('bg.none')} onpick={pickBg} onclose={() => (bgOpen = false)} />
{/if}
