<script lang="ts">
  // Files of a project: upload (button or drag & drop, max. 25 MB), download, delete.
  import { api, errorText } from '$lib/api';
  import { fmtDateTime, shortName, t } from '$lib/i18n.svelte';
  import { toast } from '$lib/state.svelte';
  import type { ProjectFile } from '$lib/types';
  import ConfirmButton from './ConfirmButton.svelte';
  import Icon from './Icon.svelte';

  let { projectId, editable, onchange }: { projectId: number; editable: boolean; onchange?: (count: number) => void } = $props();

  const MAX = 25 * 1024 * 1024;
  let files = $state<ProjectFile[] | null>(null);
  let uploading = $state(0);
  let dragOver = $state(false);
  let input = $state<HTMLInputElement>();

  $effect(() => {
    load(projectId);
  });
  async function load(id: number) {
    try {
      files = await api.get<ProjectFile[]>(`/projects/${id}/files`);
      onchange?.(files.length);
    } catch (e) {
      toast(errorText(e));
    }
  }

  async function upload(list: FileList | File[]) {
    for (const f of list) {
      if (f.size > MAX) {
        toast(t('files.tooLarge', { name: f.name }));
        continue;
      }
      const body = new FormData();
      body.append('file', f);
      uploading++;
      try {
        await api.post(`/projects/${projectId}/files`, body);
      } catch (e) {
        toast(`${f.name}: ${errorText(e)}`);
      } finally {
        uploading--;
      }
    }
    await load(projectId);
  }

  async function remove(f: ProjectFile) {
    try {
      await api.del(`/files/${f.id}`);
      await load(projectId);
    } catch (e) {
      toast(errorText(e));
    }
  }

  function size(n: number) {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)} KB`;
    return `${(n / 1024 / 1024).toFixed(1)} MB`;
  }
  const icon = (mime: string) => (mime.startsWith('image/') ? 'grid' : 'fileText');
</script>

{#if editable}
  <div
    role="region"
    aria-label={t('files.drop')}
    class="mb-3 flex flex-col items-center justify-center gap-1.5 rounded-xl border-2 border-dashed px-4 py-6 text-center text-sm transition-colors
      {dragOver ? 'border-accent-400 bg-accent-50 dark:bg-accent-950/30' : 'border-zinc-200 text-zinc-500 dark:border-zinc-800'}"
    ondragover={(e) => {
      if (!e.dataTransfer?.types.includes('Files')) return;
      e.preventDefault();
      dragOver = true;
    }}
    ondragleave={() => (dragOver = false)}
    ondrop={(e) => {
      e.preventDefault();
      dragOver = false;
      if (e.dataTransfer?.files.length) upload(e.dataTransfer.files);
    }}>
    <Icon name="download" size={18} class="rotate-180 text-zinc-400" />
    <p>
      {t('files.drop')}
      <button type="button" class="font-medium text-accent-600 hover:underline dark:text-accent-400" onclick={() => input?.click()}>{t('files.choose')}</button>
    </p>
    <p class="text-xs text-zinc-400">{t('files.limit')}</p>
    {#if uploading}<p class="text-xs text-accent-600 dark:text-accent-400">{t('files.uploading', { n: uploading })}</p>{/if}
    <input
      bind:this={input}
      type="file"
      multiple
      class="hidden"
      onchange={(e) => {
        const fl = e.currentTarget.files;
        if (fl?.length) upload([...fl]);
        e.currentTarget.value = '';
      }} />
  </div>
{/if}

{#if files === null}
  <p class="py-2 text-sm text-zinc-400">{t('common.loading')}</p>
{:else if !files.length}
  <p class="py-2 text-sm text-zinc-400">{t('files.none')}</p>
{:else}
  <ul class="surface divide-y divide-zinc-100 overflow-hidden rounded-xl border border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
    {#each files as f (f.id)}
      <li class="group flex items-center gap-3 px-3 py-2.5 text-sm">
        <Icon name={icon(f.mime)} size={16} class="shrink-0 text-zinc-400" />
        <a href="/api/files/{f.id}" class="min-w-0 flex-1 truncate font-medium hover:text-accent-700 hover:underline dark:hover:text-accent-400" download={f.name} title={f.name}>{f.name}</a>
        <span class="hidden shrink-0 text-xs text-zinc-400 tabular-nums sm:inline">{size(f.size)}</span>
        <span class="hidden w-40 shrink-0 truncate text-right text-xs text-zinc-400 md:inline" title={fmtDateTime(f.created_at)}>
          {shortName(f.uploaded_by_name)} · {fmtDateTime(f.created_at)}
        </span>
        <a href="/api/files/{f.id}" download={f.name} class="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800 dark:hover:text-zinc-200" title={t('files.download')} aria-label="{t('files.download')}: {f.name}">
          <Icon name="download" size={15} />
        </a>
        {#if editable}
          <ConfirmButton class="btn-ghost px-1.5! text-zinc-400 hover:text-red-600" onconfirm={() => remove(f)} confirmLabel={t('common.confirmDelete')}>
            <Icon name="trash" size={14} />
          </ConfirmButton>
        {/if}
      </li>
    {/each}
  </ul>
{/if}
