<script lang="ts">
  // Dialog to choose a background: gradient presets or uploaded images (library
  // shared by all users; larger images are scaled down to max. 5 MB before upload).
  import { api, errorText } from '$lib/api';
  import { t } from '$lib/i18n.svelte';
  import { isAdmin, session, toast } from '$lib/state.svelte';
  import { backgroundCss, GRADIENTS, prepareBackground, type BackgroundImage } from '$lib/backgrounds';
  import Icon from './Icon.svelte';

  let {
    title,
    value,
    noneLabel,
    onpick,
    onclose
  }: { title: string; value: string; noneLabel: string; onpick: (value: string) => void; onclose: () => void } = $props();

  // svelte-ignore state_referenced_locally
  let tab = $state<'gradients' | 'images'>(value.startsWith('img:') ? 'images' : 'gradients');
  let images = $state<BackgroundImage[] | null>(null);
  let uploading = $state(false);
  let input = $state<HTMLInputElement>();

  $effect(() => {
    if (tab === 'images' && images === null) load();
  });
  async function load() {
    try {
      images = await api.get<BackgroundImage[]>('/backgrounds');
    } catch (e) {
      toast(errorText(e));
      images = [];
    }
  }

  async function upload(file: File) {
    uploading = true;
    try {
      const blob = await prepareBackground(file);
      const body = new FormData();
      body.append('file', blob, file.name);
      const r = await api.post<{ id: number }>('/backgrounds', body);
      await load();
      onpick(`img:${r.id}`);
    } catch (e) {
      toast(e instanceof Error && e.message === 'image_too_large' ? t('errors.image_too_large') : errorText(e));
    } finally {
      uploading = false;
    }
  }

  async function remove(img: BackgroundImage) {
    try {
      await api.del(`/backgrounds/${img.id}`);
      images = images?.filter((x) => x.id !== img.id) ?? null;
      if (value === `img:${img.id}`) onpick('');
    } catch (e) {
      toast(errorText(e));
    }
  }
  const canDelete = (img: BackgroundImage) => isAdmin() || img.uploaded_by === session.user?.id;

  const tabCls = (on: boolean) =>
    `flex-1 rounded-md py-1.5 text-sm ${on ? 'seg-on' : 'sel-off font-medium'}`;
  const tileCls = (on: boolean) =>
    `relative aspect-[16/10] w-full overflow-hidden rounded-lg bg-cover bg-center transition-transform hover:scale-[1.03] ${on ? 'ring-3 ring-accent-500 ring-offset-2 dark:ring-offset-zinc-900' : ''}`;
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/40 p-4" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="flex max-h-[90vh] w-full max-w-2xl flex-col rounded-2xl bg-white shadow-2xl dark:bg-zinc-900" role="dialog" aria-modal="true" aria-label={title}>
    <header class="flex items-center justify-between border-b border-zinc-100 px-5 py-3.5 dark:border-zinc-800">
      <h2 class="font-semibold">{title}</h2>
      <button type="button" class="rounded-md p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800" onclick={onclose} aria-label={t('common.close')}>
        <Icon name="x" size={16} />
      </button>
    </header>

    <div class="overflow-y-auto p-5">
      <div class="mb-4 flex gap-1 rounded-lg bg-zinc-100 p-1 dark:bg-zinc-800/70" role="tablist">
        <button type="button" role="tab" aria-selected={tab === 'gradients'} class={tabCls(tab === 'gradients')} onclick={() => (tab = 'gradients')}>{t('bg.gradients')}</button>
        <button type="button" role="tab" aria-selected={tab === 'images'} class={tabCls(tab === 'images')} onclick={() => (tab = 'images')}>{t('bg.images')}</button>
      </div>

      <!-- none (= inherit) -->
      <button
        type="button"
        class="mb-4 flex w-full items-center gap-2 rounded-lg border px-3 py-2 text-left text-sm {value === '' ? 'border-accent-500 bg-accent-50 text-accent-800 dark:bg-accent-950/40 dark:text-accent-300' : 'border-zinc-200 hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800'}"
        onclick={() => onpick('')}>
        <Icon name="x" size={14} class="text-zinc-400" />{noneLabel}
        {#if value === ''}<Icon name="check" size={14} class="ml-auto" />{/if}
      </button>

      {#if tab === 'gradients'}
        <div class="grid grid-cols-3 gap-3 sm:grid-cols-5">
          {#each GRADIENTS as g (g.id)}
            <button type="button" class={tileCls(value === g.id)} style:background-image={g.css} onclick={() => onpick(g.id)} aria-label="{t('bg.gradient')} {g.id.slice(1)}" aria-pressed={value === g.id}></button>
          {/each}
        </div>
      {:else}
        <div class="grid grid-cols-3 gap-3 sm:grid-cols-5">
          <button
            type="button"
            class="flex aspect-[16/10] w-full flex-col items-center justify-center gap-1 rounded-lg border-2 border-dashed border-zinc-300 text-xs text-zinc-500 hover:border-accent-400 hover:text-accent-600 dark:border-zinc-700"
            disabled={uploading}
            onclick={() => input?.click()}>
            <Icon name={uploading ? 'clock' : 'plus'} size={18} />
            {uploading ? t('bg.uploading') : t('bg.upload')}
          </button>
          {#each images ?? [] as img (img.id)}
            <div class="group relative">
              <button type="button" class={tileCls(value === `img:${img.id}`)} style:background-image={backgroundCss(`img:${img.id}`)} onclick={() => onpick(`img:${img.id}`)} aria-label={t('bg.image')} aria-pressed={value === `img:${img.id}`}></button>
              {#if canDelete(img)}
                <button
                  type="button"
                  class="absolute top-1 right-1 rounded-md bg-black/50 p-1 text-white opacity-0 group-hover:opacity-100 hover:bg-red-600 focus:opacity-100 [@media(hover:none)]:opacity-100"
                  onclick={() => remove(img)}
                  title={t('bg.delete')}
                  aria-label={t('bg.delete')}>
                  <Icon name="trash" size={12} />
                </button>
              {/if}
            </div>
          {/each}
        </div>
        {#if images !== null && !images.length}<p class="mt-3 text-sm text-zinc-400">{t('bg.noImages')}</p>{/if}
        <p class="mt-3 text-xs text-zinc-400">{t('bg.hint')}</p>
        <input
          bind:this={input}
          type="file"
          accept="image/*"
          class="hidden"
          onchange={(e) => {
            const f = e.currentTarget.files?.[0];
            if (f) upload(f);
            e.currentTarget.value = '';
          }} />
      {/if}
    </div>
  </div>
</div>
