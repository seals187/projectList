<script lang="ts">
  // Code block: highlighted display with line numbers; while editing, a
  // transparent <textarea> sits exactly over the code (same font/width).
  import { CODE_LANGS, guessLang, highlightLine } from '$lib/highlight';
  import { t } from '$lib/i18n.svelte';
  import Icon from './Icon.svelte';
  import { prefs } from '$lib/state.svelte'; // prefs.codeBg: Settings → Checklists → Code block

  let {
    code,
    lang,
    editable,
    oninput,
    onlang,
    onkeydown,
    onfocus,
    register
  }: {
    code: string;
    lang: string;
    editable: boolean;
    oninput: (v: string) => void;
    onlang: (lang: string) => void;
    onkeydown?: (e: KeyboardEvent, el: HTMLTextAreaElement) => void;
    onfocus?: () => void;
    register?: (el: HTMLTextAreaElement | null) => void;
  } = $props();

  let copied = $state(false);
  let ta = $state<HTMLTextAreaElement>();

  $effect(() => {
    register?.(ta ?? null);
    return () => register?.(null);
  });

  const effLang = $derived(lang || guessLang(code));
  const lines = $derived(code.split('\n'));
  const label = $derived(CODE_LANGS.find((l) => l.id === lang)?.label ?? CODE_LANGS.find((l) => l.id === effLang)?.label ?? 'Code');

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
    } catch {
      // Fallback for insecure contexts (HTTP via IP)
      const tmp = document.createElement('textarea');
      tmp.value = code;
      document.body.appendChild(tmp);
      tmp.select();
      document.execCommand('copy');
      tmp.remove();
    }
    copied = true;
    setTimeout(() => (copied = false), 1500);
  }

  function keydown(e: KeyboardEvent) {
    const el = e.currentTarget as HTMLTextAreaElement;
    if (e.key === 'Tab' && !e.shiftKey) {
      e.preventDefault();
      document.execCommand('insertText', false, '  ');
      return;
    }
    onkeydown?.(e, el);
  }
</script>

<div class="code-block my-1 overflow-hidden rounded-lg text-[13px] shadow-sm" data-bg={prefs.codeBg}>
  <div class="code-head flex items-center gap-2 border-b px-3 py-1.5 text-xs">
    {#if editable}
      <select
        class="code-hover cursor-pointer rounded bg-transparent pr-1 text-xs outline-none"
        value={lang}
        aria-label={t('code.language')}
        onchange={(e) => onlang(e.currentTarget.value)}>
        <option value="" class="text-zinc-900">Auto ({CODE_LANGS.find((l) => l.id === effLang)?.label})</option>
        {#each CODE_LANGS as l (l.id)}<option value={l.id} class="text-zinc-900">{l.label}</option>{/each}
      </select>
    {:else}
      <span>{label}</span>
    {/if}
    <!-- Background: Settings → Checklists → Code block -->
    <button type="button" class="code-hover code-btn ml-auto inline-flex items-center gap-1 rounded px-1.5 py-0.5" onclick={copy}>
      <Icon name={copied ? 'check' : 'copy'} size={12} />
      {copied ? t('code.copied') : t('code.copy')}
    </button>
  </div>
  <div class="relative py-2.5 font-mono leading-5">
    {#each lines as line, i (i)}
      <div class="flex">
        <span class="code-ln w-9 shrink-0 pr-3 text-right select-none">{i + 1}</span>
        <span class="min-w-0 flex-1 pr-3 break-all whitespace-pre-wrap">{@html highlightLine(line, effLang) || '​'}</span>
      </div>
    {/each}
    {#if editable}
      <textarea
        bind:this={ta}
        value={code}
        spellcheck="false"
        autocomplete="off"
        placeholder={t('code.placeholder')}
        oninput={(e) => oninput(e.currentTarget.value)}
        onkeydown={keydown}
        onfocus={() => onfocus?.()}
        class="absolute inset-y-2.5 right-0 left-9 resize-none overflow-hidden bg-transparent pr-3 font-mono leading-5 break-all whitespace-pre-wrap text-transparent outline-none selection:bg-accent-500/40"
      ></textarea>
    {/if}
  </div>
</div>
