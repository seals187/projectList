<script lang="ts">
  // Star next to a page title: adds the customer, project or checklist to the
  // favorites in the sidebar or removes it.
  import { t } from '$lib/i18n.svelte';
  import { isFavorite, toggleFavorite } from '$lib/navMemory.svelte';
  import type { FavoriteKind } from '$lib/types';
  import Icon from './Icon.svelte';

  let { kind, id, size = 18 }: { kind: FavoriteKind; id: number; size?: number } = $props();
  const fav = $derived(isFavorite(kind, id));
</script>

<button
  type="button"
  class="rounded-md p-1 {fav ? 'text-amber-500' : 'text-zinc-400 hover:text-amber-500'}"
  onclick={() => toggleFavorite(kind, id)}
  title={t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
  aria-label={t(fav ? 'nav.removeFavorite' : 'nav.addFavorite')}
  aria-pressed={fav}>
  <Icon name="star" {size} class={fav ? 'fill-amber-400' : ''} />
</button>
