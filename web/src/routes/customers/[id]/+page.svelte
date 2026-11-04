<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, errorText } from '$lib/api';
  import { canWrite, isAdmin, refreshSidebar, toast } from '$lib/state.svelte';
  import { t } from '$lib/i18n.svelte';
  import type { Customer, Project, ProjectWithProgress } from '$lib/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Editable from '$lib/components/Editable.svelte';
  import ConfirmButton from '$lib/components/ConfirmButton.svelte';
  import Progress from '$lib/components/Progress.svelte';
  import SaveStatus from '$lib/components/SaveStatus.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { STATUS_BADGE } from '$lib/projectStatus';

  let customer = $state<Customer | null>(null);
  let projects = $state<ProjectWithProgress[]>([]);
  let notFound = $state(false);
  let newProject = $state('');
  let showArchived = $state(false);

  const id = $derived(Number(page.params.id));

  $effect(() => {
    load(id);
  });

  async function load(id: number) {
    notFound = false;
    try {
      const r = await api.get<{ customer: Customer; projects: ProjectWithProgress[] }>(`/customers/${id}`);
      customer = r.customer;
      projects = r.projects;
    } catch {
      customer = null;
      notFound = true;
    }
  }

  async function save(field: 'name' | 'notes' | 'location' | 'contact_name' | 'contact_email', value: string) {
    customer = await api.patch<Customer>(`/customers/${id}`, { [field]: value });
    if (field === 'name') refreshSidebar(0);
  }

  async function addProject(e: SubmitEvent) {
    e.preventDefault();
    if (!newProject.trim()) return;
    try {
      const p = await api.post<Project>(`/customers/${id}/projects`, { name: newProject });
      newProject = '';
      refreshSidebar(0);
      goto(`/projects/${p.id}`);
    } catch (err) {
      toast(errorText(err));
    }
  }

  async function remove() {
    try {
      await api.del(`/customers/${id}`);
      refreshSidebar(0);
      goto('/customers');
    } catch (err) {
      toast(errorText(err));
    }
  }

  const visible = $derived(projects.filter((p) => showArchived || p.status !== 'archived'));
  const archivedCount = $derived(projects.filter((p) => p.status === 'archived').length);
</script>

<svelte:head><title>{customer?.name ?? ''} · ProjectList</title></svelte:head>

{#if notFound}
  <p class="text-zinc-500">{t('common.notFound')}</p>
{:else if customer}
  <PageHeader crumbs={[{ label: t('nav.customers'), href: '/customers' }]}>
    {#snippet actions()}<SaveStatus />{/snippet}
    {#snippet title()}
      <Editable value={customer!.name} required disabled={!canWrite()} onsave={(v) => save('name', v)} inputClass="font-serif text-2xl! font-bold" />
    {/snippet}
  </PageHeader>

  <!-- Location and contact -->
  <dl class="mb-5 grid max-w-2xl gap-x-6 gap-y-2 text-sm sm:grid-cols-3">
    {#each [['location', 'customers.location'], ['contact_name', 'customers.contact'], ['contact_email', 'customers.email']] as const as [field, label] (field)}
      <div class="min-w-0">
        <dt class="text-[11px] font-semibold tracking-wider text-zinc-400 uppercase">{t(label)}</dt>
        <dd class="mt-0.5 min-w-0">
          <Editable value={customer[field]} placeholder="–" disabled={!canWrite()} onsave={(v) => save(field, v)} />
        </dd>
      </div>
    {/each}
  </dl>

  <div class="max-w-2xl text-sm text-zinc-600 dark:text-zinc-300">
    <Editable value={customer.notes} multiline placeholder={t('customer.notesPlaceholder')} disabled={!canWrite()} onsave={(v) => save('notes', v)} />
  </div>

  <section class="mt-10">
    <div class="mb-2 flex items-center justify-between">
      <h2 class="section-label">{t('customer.projects')}</h2>
      {#if archivedCount}
        <button type="button" class="text-xs text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200" onclick={() => (showArchived = !showArchived)}>
          {showArchived ? t('customer.hideArchived') : t('customer.showArchived', { n: archivedCount })}
        </button>
      {/if}
    </div>

    <ul>
      {#each visible as p (p.id)}
        <li>
          <a href="/projects/{p.id}" class="row-link">
            <Icon name="folder" class="text-zinc-400" />
            <span class="flex-1 truncate font-medium {p.status === 'archived' ? 'text-zinc-400 line-through' : ''}">{p.name}</span>
            {#if p.status !== 'active'}<span class="badge {STATUS_BADGE[p.status]}">{t(`status.${p.status}`)}</span>{/if}
            <Progress done={p.done} total={p.total} width="w-16" compact />
          </a>
        </li>
      {:else}
        <li class="py-4 text-sm text-zinc-400">{t('customer.noProjects')}</li>
      {/each}
    </ul>
    {#if canWrite()}
      <form onsubmit={addProject} class="mt-1 flex items-center gap-2 py-1">
        <Icon name="plus" class="text-zinc-400" />
        <input class="flex-1 bg-transparent py-1 text-sm outline-none placeholder:text-zinc-400" placeholder={t('customer.newProject')} bind:value={newProject} />
        {#if newProject.trim()}<button class="btn-primary py-1!" type="submit">{t('common.create')}</button>{/if}
      </form>
    {/if}
  </section>

  {#if isAdmin()}
    <div class="mt-16 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <ConfirmButton onconfirm={remove} label={t('customer.delete')} confirmLabel={t('customer.deleteConfirm')} />
    </div>
  {/if}
{/if}
