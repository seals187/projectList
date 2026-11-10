<script lang="ts">
  // Type icon (customer, project, checklist, table, template, saved view, favorites
  // group): icon in a rounded box in the type colors (CSS variables, see
  // $lib/entityTypes and app.css). Sizes: sm 22 px, md 26 px, lg 32 px.
  import { ENTITY_SIZES, ENTITY_TYPES, type EntitySize, type EntityType } from '$lib/entityTypes';
  import Icon from './Icon.svelte';

  let { type, size = 'md', class: cls = '', title }: { type: EntityType; size?: EntitySize; class?: string; title?: string } = $props();

  const def = $derived(ENTITY_TYPES[type]);
  const s = $derived(ENTITY_SIZES[size]);
</script>

<span
  class="inline-flex shrink-0 items-center justify-center rounded-[7px] {cls}"
  style:width="{s.box}px"
  style:height="{s.box}px"
  style:background-color="var(--type-{def.css}-bg)"
  style:color="var(--type-{def.css}-fg)"
  data-entity={type}
  {title}
  aria-hidden={title ? undefined : 'true'}>
  <Icon name={def.icon} size={Math.round(s.icon * (def.scale ?? 1))} />
</span>
