<script lang="ts">
  // Zweistufiges Löschen ohne Modal: 1. Klick fragt nach, 2. Klick führt aus.
  import type { Snippet } from 'svelte';
  import { t } from '$lib/i18n.svelte';

  let {
    onconfirm,
    label,
    confirmLabel,
    class: cls = 'btn-danger',
    children
  }: {
    onconfirm: () => void;
    label?: string;
    confirmLabel?: string;
    class?: string;
    children?: Snippet;
  } = $props();

  let armed = $state(false);
  let timer: ReturnType<typeof setTimeout>;

  function click() {
    if (armed) {
      clearTimeout(timer);
      armed = false;
      onconfirm();
      return;
    }
    armed = true;
    timer = setTimeout(() => (armed = false), 3000);
  }
</script>

<button type="button" class="{cls} {armed ? 'bg-red-600! text-white!' : ''}" onclick={click} title={label ?? t('common.delete')}>
  {#if armed}{confirmLabel ?? t('common.confirmDelete')}{:else if children}{@render children()}{:else}{label ?? t('common.delete')}{/if}
</button>
