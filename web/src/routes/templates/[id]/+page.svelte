<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { Ops, Template, TemplateItem } from '$lib/types';
  import DetailHeader from '$lib/components/DetailHeader.svelte';
  import BlockEditor from '$lib/components/BlockEditor.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';

  let template = $state<Template | null>(null);
  let items = $state<TemplateItem[]>([]);
  let notFound = $state(false);

  const id = $derived(Number(page.params.id));
  const editable = $derived(canWrite());

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    try {
      const r = await api.get<{ template: Template; items: TemplateItem[] }>(`/templates/${id}`);
      template = r.template;
      items = r.items;
    } catch {
      notFound = true;
    }
  }

  async function save(body: Partial<Template>) {
    template = await api.patch<Template>(`/templates/${id}`, body);
  }

  const ops: Ops<TemplateItem> = {
    create: (body) => api.post<TemplateItem>(`/templates/${id}/items`, body),
    patch: (itemId, body) => api.patch<TemplateItem>(`/template-items/${itemId}`, body),
    remove: (itemId) => api.del(`/template-items/${itemId}`),
    reorder: (ids) => api.put(`/templates/${id}/items/order`, { ids })
  };

  async function remove() {
    try {
      await api.del(`/templates/${id}`);
      goto('/templates');
    } catch (e) {
      toast(errorText(e));
    }
  }
</script>

<svelte:head><title>{template?.name ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if template}
  <DetailHeader crumbs={[{ label: t('nav.templates'), href: '/templates' }]}>
    {#snippet actions()}<SaveStatus />{/snippet}
    {#snippet title()}
      <Editable value={template!.name} required disabled={!editable} onsave={(name) => save({ name })} inputClass="font-serif text-2xl! font-bold" />
    {/snippet}
  </DetailHeader>

  <div class="mb-6 max-w-2xl text-sm text-zinc-600 dark:text-zinc-300">
    <Editable value={template.description} multiline placeholder={t('template.descPlaceholder')} disabled={!editable} onsave={(description) => save({ description })} />
  </div>

  {#key template.id}
    <BlockEditor bind:items {ops} {editable} collapseKey="template:{id}" onerror={() => load(id)} />
  {/key}

  <p class="mt-6 text-xs text-zinc-400">{t('template.hint')}</p>

  {#if editable}
    <div class="mt-16 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <ConfirmButton onconfirm={remove} label={t('template.delete')} confirmLabel={t('template.deleteConfirm')} />
    </div>
  {/if}
{/if}
