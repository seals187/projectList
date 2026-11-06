<script lang="ts">
  // Settings → Font & checklists: app font, headings, checkbox position/shape,
  // dividers and code block background. Right: live preview of a checklist.
  import { t } from '$lib/i18n.svelte';
  import { checkRadius, CODE_BGS, prefs, setPref, type CheckPos, type CheckShape } from '$lib/state.svelte';
  import { FONTS } from '$lib/fonts';
  import SettingsShell from '$lib/components/SettingsShell.svelte';
  import SetCard from '$lib/components/SetCard.svelte';
  import SetRow from '$lib/components/SetRow.svelte';
  import Seg from '$lib/components/Seg.svelte';
  import Switch from '$lib/components/Switch.svelte';
  import Icon from '$lib/components/Icon.svelte';

  type TKey = Parameters<typeof t>[0];
  const positions: CheckPos[] = ['left', 'right'];
  const shapes: CheckShape[] = ['square', 'round'];
  // code block backgrounds (as in app.css)
  const CODE_BG = { light: 'bg-[#f1f2f4] text-zinc-800', dark: 'bg-[#2b2d35] text-zinc-100', black: 'bg-black text-zinc-100' } as const;

  const font = $derived(FONTS.find((f) => f.id === prefs.font) ?? FONTS[0]);
  const items = $derived([
    { title: t('set.preview.item1'), done: true },
    { title: t('set.preview.item2'), done: false },
    { title: t('set.preview.item3'), done: false }
  ]);
</script>

<SettingsShell title={t('set.font.title')} subtitle={t('set.font.subtitle')} asideClass="bg-[#f9f9fb] dark:bg-zinc-950/30">
  <div class="max-w-[640px]">
    <SetCard title={t('set.fontFamily')}>
      <div class="p-2" role="radiogroup" aria-label={t('set.fontFamily')}>
        {#each FONTS as f (f.id)}
          {@const on = prefs.font === f.id}
          <button
            type="button"
            role="radio"
            aria-checked={on}
            class="flex w-full items-center gap-3 rounded-[10px] border px-3 py-2.5 text-left transition-colors {on
              ? 'border-accent-300 bg-accent-50 dark:border-accent-500/50 dark:bg-accent-400/10'
              : 'border-transparent hover:bg-zinc-50 dark:hover:bg-zinc-800/60'}"
            onclick={() => setPref('font', f.id)}>
            <span class="flex size-[18px] shrink-0 items-center justify-center rounded-full border-2 {on ? 'border-[var(--accent-primary)]' : 'border-zinc-300 dark:border-zinc-600'}">
              {#if on}<span class="size-2 rounded-full bg-[var(--accent-primary)]"></span>{/if}
            </span>
            <span class="text-[15px] font-semibold" style:font-family={f.family}>{f.label}</span>
            <span class="truncate text-[13px] text-zinc-400">{f.hint ?? t(`font.hint.${f.id}` as TKey)}</span>
            <span class="ml-auto shrink-0 text-[17px] tabular-nums" style:font-family={f.family}>Aa 0123</span>
          </button>
        {/each}
      </div>
      <SetRow title={t('settings.headings')} desc={t('set.headings.desc')}>
        <Seg label={t('settings.headings')} options={(['serif', 'sans'] as const).map((h) => ({ value: h, label: t(`settings.heading.${h}`) }))} value={prefs.headingFont} onchange={(v) => setPref('headingFont', v)} />
      </SetRow>
    </SetCard>

    <SetCard title={t('settings.checklists')} id="checklists">
      <SetRow title={t('set.checkbox')} desc={t('set.checkbox.desc')}>
        <Seg label={t('settings.checkPos')} options={positions.map((p) => ({ value: p, label: t(`settings.pos.${p}`) }))} value={prefs.checkPos} onchange={(v) => setPref('checkPos', v)} />
        <Seg label={t('settings.checkShape')} options={shapes.map((s) => ({ value: s, label: t(`settings.shape.${s}`) }))} value={prefs.checkShape} onchange={(v) => setPref('checkShape', v)} />
      </SetRow>
      <SetRow title={t('set.dividers')} desc={t('set.dividers.desc')}>
        <Switch label={t('set.dividers')} checked={prefs.gridLines} onchange={(v) => setPref('gridLines', v)} />
      </SetRow>
      <SetRow title={t('set.codeBlock')} desc={t('set.codeBlock.desc')}>
        <Seg label={t('set.codeBlock')} options={CODE_BGS.map((b) => ({ value: b, label: t(`set.code.${b}`) }))} value={prefs.codeBg} onchange={(v) => setPref('codeBg', v)} />
      </SetRow>
    </SetCard>
  </div>

  {#snippet aside()}
    <div class="p-5 md:pt-7">
      <p class="mb-3 text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t('set.preview.live')}</p>
      <div class="rounded-2xl bg-white p-5 shadow-[0_4px_18px_rgba(40,20,90,0.08)] dark:bg-zinc-900" data-testid="font-preview">
        <p class="text-[12.5px] text-zinc-400">{t('set.preview.crumb')}</p>
        <p class="mt-1 font-serif text-[22px] font-bold tracking-tight">{t('set.preview.title')}</p>
        <p class="mt-3 text-[11.5px] font-semibold tracking-wider text-zinc-500 uppercase">{t('set.preview.section')}</p>
        <ul class="mt-1.5 {prefs.gridLines ? 'divide-y divide-zinc-200 border-t border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800' : ''}">
          {#each items as it (it.title)}
            <li class="flex items-center gap-2.5 py-2.5 text-[14.5px] {prefs.checkPos === 'right' ? 'flex-row-reverse justify-end' : ''}">
              <span class="flex size-[18px] shrink-0 items-center justify-center border-2 {checkRadius()} {it.done ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)] text-white' : 'border-zinc-300 dark:border-zinc-600'}">
                {#if it.done}<Icon name="check" size={11} class="stroke-[3]" />{/if}
              </span>
              <span class="{prefs.checkPos === 'right' ? 'flex-1' : ''} {it.done ? 'text-zinc-400 line-through' : ''}">{it.title}</span>
            </li>
          {/each}
        </ul>
        <pre class="mt-3 rounded-lg px-3 py-2.5 font-mono text-[13px] {CODE_BG[prefs.codeBg]}">systemctl restart nginx</pre>
      </div>
      <p class="mt-3 text-[12.5px] text-zinc-400">
        {t('set.preview.summary', {
          font: font.label,
          pos: t(`settings.pos.${prefs.checkPos}`).toLowerCase(),
          shape: t(`settings.shape.${prefs.checkShape}`).toLowerCase(),
          code: t(`set.code.${prefs.codeBg}`).toLowerCase()
        })}
      </p>
    </div>
  {/snippet}
</SettingsShell>
