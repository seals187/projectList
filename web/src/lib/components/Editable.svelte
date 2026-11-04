<script lang="ts">
  // Inline editing: click text → input field. Enter/blur saves,
  // Escape cancels. In multiline fields, Ctrl+Enter saves.
  import { tick } from 'svelte';
  import { errorText } from '$lib/api';
  import { toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';

  let {
    value,
    onsave,
    multiline = false,
    placeholder = '',
    disabled = false,
    required = false,
    class: cls = '',
    placeholderClass = '',
    inputClass = '',
    title = undefined
  }: {
    value: string;
    onsave: (v: string) => Promise<unknown> | void;
    multiline?: boolean;
    placeholder?: string;
    disabled?: boolean;
    required?: boolean;
    class?: string;
    placeholderClass?: string;
    inputClass?: string;
    title?: string;
  } = $props();

  let editing = $state(false);
  let draft = $state('');
  let saving = $state(false);
  let el = $state<HTMLInputElement | HTMLTextAreaElement>();

  async function start(e?: Event) {
    if (disabled) return;
    // Clicks on links in the formatted text should open the link.
    if (e && (e.target as HTMLElement).closest('a')) return;
    draft = value;
    editing = true;
    await tick();
    el?.focus();
    if (el && !multiline) el.select();
  }

  async function commit() {
    if (!editing || saving) return;
    const v = draft.trim();
    if (v === value.trim() || (required && !v)) {
      editing = false;
      return;
    }
    saving = true;
    try {
      await onsave(v);
      editing = false;
    } catch (e) {
      toast(errorText(e));
      el?.focus();
    } finally {
      saving = false;
    }
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      editing = false;
    } else if (e.key === 'Enter' && (!multiline || e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      commit();
    }
  }

  function onDisplayKey(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      start();
    }
  }
</script>

{#if editing}
  {#if multiline}
    <textarea
      bind:this={el}
      bind:value={draft}
      {onkeydown}
      onblur={commit}
      rows="3"
      disabled={saving}
      class="input {inputClass}"></textarea>
  {:else}
    <input
      bind:this={el}
      bind:value={draft}
      {onkeydown}
      onblur={commit}
      disabled={saving}
      class="input {inputClass}" />
  {/if}
{:else if disabled}
  <div class={cls}>
    {#if value}
      <span class={multiline ? 'whitespace-pre-wrap' : ''}>{value}</span>
    {/if}
  </div>
{:else}
  <div
    role="button"
    tabindex="0"
    onclick={start}
    onkeydown={onDisplayKey}
    title={title ?? t('common.clickToEdit')}
    class="-mx-1 min-h-[1.25em] cursor-text rounded px-1 outline-none focus-visible:ring-2 focus-visible:ring-accent-500/40 {cls}">
    {#if value}
      <span class={multiline ? 'whitespace-pre-wrap' : ''}>{value}</span>
    {:else}
      <span class="text-zinc-400 dark:text-zinc-500 {placeholderClass}">{placeholder || t('common.clickToEdit')}</span>
    {/if}
  </div>
{/if}
