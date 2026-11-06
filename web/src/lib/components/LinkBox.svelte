<script lang="ts">
  // Result of an invitation / password reset: whether the e-mail was sent and
  // the link to pass on by hand (copy button).
  import { t } from '$lib/i18n.svelte';
  import Icon from './Icon.svelte';

  let { link, emailed, email }: { link: string; emailed: boolean; email: string } = $props();
  let copied = $state(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(link);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      /* clipboard blocked: the field stays selectable */
    }
  }
</script>

<div class="space-y-2" data-testid="link-box">
  {#if emailed}
    <p class="flex items-center gap-1.5 text-[13.5px] font-medium text-emerald-700 dark:text-emerald-400"><Icon name="check" size={15} />{t('set.users.mailSent', { email })}</p>
  {:else}
    <p class="text-[13px] text-zinc-500">{t('set.users.noMail')}</p>
  {/if}
  <div class="flex gap-2">
    <input class="input h-9! min-w-0 font-mono text-xs!" readonly value={link} aria-label={t('set.users.link')} onfocus={(e) => e.currentTarget.select()} />
    <button type="button" class="btn-outline h-9 shrink-0" onclick={copy}><Icon name={copied ? 'check' : 'copy'} size={14} />{copied ? t('set.users.copied') : t('set.users.copy')}</button>
  </div>
</div>
