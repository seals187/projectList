<script lang="ts">
  // Manage the template categories (admins): name, color, add, delete.
  // Changes are saved immediately; onchange reloads the list of the page.
  import { api, errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import { CARD_COLORS, CARD_HEX } from '$lib/cardColors';
  import { categoryHex } from '$lib/templateLibrary';
  import type { TemplateCategory } from '$lib/types';
  import Icon from './Icon.svelte';

  let { categories, onchange, onclose }: { categories: TemplateCategory[]; onchange: () => void; onclose: () => void } = $props();

  let dialog = $state<HTMLDialogElement>();
  let newName = $state('');
  let newColor = $state<string>('blue');
  let colorFor = $state<number | null>(null); // open color row
  let confirmDel = $state<number | null>(null);

  $effect(() => {
    dialog?.showModal();
  });

  async function run(p: Promise<unknown>) {
    try {
      await p;
      onchange();
    } catch (e) {
      toast(errorText(e));
    }
  }
  const rename = (c: TemplateCategory, name: string) => name.trim() && name.trim() !== c.name && run(api.patch(`/template-categories/${c.id}`, { name: name.trim() }));
  const recolor = (c: TemplateCategory, color: string) => ((colorFor = null), run(api.patch(`/template-categories/${c.id}`, { color })));
  const remove = (c: TemplateCategory) => ((confirmDel = null), run(api.del(`/template-categories/${c.id}`)));
  async function add(e: SubmitEvent) {
    e.preventDefault();
    if (!newName.trim()) return;
    await run(api.post('/template-categories', { name: newName.trim(), color: newColor }));
    newName = '';
  }
</script>

{#snippet swatches(value: string, pick: (c: string) => void)}
  <div class="flex flex-wrap gap-1.5 py-1.5">
    {#each CARD_COLORS as c (c)}
      <button
        type="button"
        class="size-5 rounded-md ring-offset-2 ring-offset-white dark:ring-offset-zinc-900 {value === c ? 'ring-2 ring-zinc-500' : ''}"
        style:background-color={CARD_HEX[c]}
        onclick={() => pick(c)}
        aria-label={t(`color.${c}`)}
        aria-pressed={value === c}></button>
    {/each}
  </div>
{/snippet}

<dialog
  bind:this={dialog}
  {onclose}
  onclick={(e) => e.target === dialog && dialog?.close()}
  class="m-auto w-[min(30rem,calc(100vw-2rem))] rounded-xl border border-zinc-200 bg-white p-0 text-zinc-900 shadow-xl backdrop:bg-zinc-950/30 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100"
  data-testid="category-dialog">
  <div class="space-y-3 p-5">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-semibold">{t('cat.manage')}</h2>
      <button type="button" class="btn-ghost px-1.5!" onclick={() => dialog?.close()} aria-label={t('common.close')}><Icon name="x" size={16} /></button>
    </div>
    <p class="text-xs text-zinc-500">{t('cat.hint')}</p>

    <ul class="divide-y divide-zinc-100 dark:divide-zinc-800">
      {#each categories as c (c.id)}
        <li class="py-1.5">
          <div class="flex items-center gap-2">
            <button type="button" class="size-5 shrink-0 rounded-md" style:background-color={categoryHex(c)} onclick={() => (colorFor = colorFor === c.id ? null : c.id)} aria-label={t('cat.color')}></button>
            <input class="input py-1! text-[13px]!" value={c.name} maxlength="60" aria-label={t('cat.name')} onchange={(e) => rename(c, e.currentTarget.value)} />
            {#if confirmDel === c.id}
              <button type="button" class="btn bg-red-600 px-2! py-1! text-xs text-white hover:bg-red-700" onclick={() => remove(c)}>{t('common.delete')}</button>
              <button type="button" class="btn-ghost px-1.5!" onclick={() => (confirmDel = null)} aria-label={t('common.cancel')}><Icon name="x" size={14} /></button>
            {:else}
              <button type="button" class="btn-ghost px-1.5! text-zinc-400 hover:text-red-600!" onclick={() => (confirmDel = c.id)} title={t('cat.delete')} aria-label="{t('cat.delete')}: {c.name}">
                <Icon name="trash" size={14} />
              </button>
            {/if}
          </div>
          {#if colorFor === c.id}{@render swatches(c.color, (col) => recolor(c, col))}{/if}
          {#if confirmDel === c.id}<p class="pt-1 text-xs text-zinc-500">{t('cat.deleteHint')}</p>{/if}
        </li>
      {:else}
        <li class="py-2 text-sm text-zinc-400">{t('cat.empty')}</li>
      {/each}
    </ul>

    <form onsubmit={add} class="rounded-lg bg-zinc-50 p-2.5 dark:bg-zinc-800/50">
      <div class="flex items-center gap-2">
        <span class="size-5 shrink-0 rounded-md" style:background-color={CARD_HEX[newColor as keyof typeof CARD_HEX]}></span>
        <input class="input py-1! text-[13px]!" bind:value={newName} maxlength="60" placeholder={t('cat.new')} aria-label={t('cat.new')} />
        <button type="submit" class="btn-primary px-2.5! py-1!" disabled={!newName.trim()}><Icon name="plus" size={14} />{t('common.create')}</button>
      </div>
      {@render swatches(newColor, (col) => (newColor = col))}
    </form>
  </div>
</dialog>
