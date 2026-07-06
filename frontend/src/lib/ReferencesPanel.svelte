<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  const refs = $derived(store.references);
  const total = $derived(
    refs?.groups?.reduce((n, g) => n + g.items.length, 0) ?? 0
  );

  function fileName(path) {
    return path.split('/').pop();
  }

  function fileDir(path) {
    const parts = path.split('/');
    parts.pop();
    return parts.join('/');
  }

  function jump(path, line) {
    store.openFile(path, line);
  }
</script>

<div class="refs-panel">
  <div class="refs-header">
    <span class="refs-title">References</span>
    {#if refs && !refs.loading}
      <span class="refs-count">{total} in {refs.groups.length} file{refs.groups.length === 1 ? '' : 's'}</span>
    {/if}
    <button class="refs-close" title="Clear" onclick={() => store.clearReferences()}>
      <Icon name="close" size={13} />
    </button>
  </div>

  {#if refs}
    <div class="refs-symbol">
      <Icon name="references" size={13} color="#818cf8" />
      <code>{refs.symbol}</code>
    </div>
  {/if}

  <div class="refs-body">
    {#if !refs}
      <div class="refs-empty">
        Right-click a symbol and choose <em>Find All References</em>, press
        <kbd>Shift</kbd>+<kbd>F12</kbd>, or <kbd>Alt</kbd>+click it.
      </div>
    {:else if refs.loading}
      <div class="refs-empty">Searching…</div>
    {:else if refs.error}
      <div class="refs-empty">{refs.error}</div>
    {:else if total === 0}
      <div class="refs-empty">No references found for <code>{refs.symbol}</code>.</div>
    {:else}
      {#each refs.groups as group (group.path)}
        <div class="refs-group">
          <div class="refs-file" title={group.path}>
            <Icon name="file" size={13} color="#9ca3af" />
            <span class="refs-file-name">{fileName(group.path)}</span>
            {#if fileDir(group.path)}
              <span class="refs-file-dir">{fileDir(group.path)}</span>
            {/if}
            <span class="refs-file-count">{group.items.length}</span>
          </div>
          {#each group.items as item (item.line + ':' + item.column)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="refs-item" onclick={() => jump(group.path, item.line)}>
              <span class="refs-line">{item.line}</span>
              <span class="refs-preview">{item.preview || '…'}</span>
            </div>
          {/each}
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .refs-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    color: #c4c4cc;
    font-size: 13px;
  }

  .refs-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px 8px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .refs-title {
    font-size: 11px;
    font-weight: 600;
    color: #8e8e93;
  }

  .refs-count {
    font-size: 10px;
    color: #6b7280;
    text-transform: none;
    letter-spacing: 0;
  }

  .refs-close {
    margin-left: auto;
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    display: flex;
    padding: 2px;
    border-radius: 4px;
  }
  .refs-close:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #e3e3e6;
  }

  .refs-symbol {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 12px 8px;
    border-bottom: 1px solid #2d2d34;
  }
  .refs-symbol code {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #e3e3e6;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .refs-body {
    flex: 1;
    overflow-y: auto;
  }

  .refs-empty {
    padding: 16px 14px;
    color: #6b7280;
    font-size: 12px;
    line-height: 1.6;
  }
  .refs-empty code {
    font-family: 'Fira Code', monospace;
    color: #a5b4fc;
  }
  .refs-empty kbd {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 3px;
    padding: 1px 4px;
    font-size: 10px;
    font-family: inherit;
  }

  .refs-group {
    padding: 4px 0;
  }

  .refs-file {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 12px;
    position: sticky;
    top: 0;
    background: #1b1b20;
    z-index: 1;
  }
  .refs-file-name {
    color: #e3e3e6;
    font-weight: 500;
    font-size: 12px;
  }
  .refs-file-dir {
    color: #5d5d66;
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
  }
  .refs-file-count {
    margin-left: auto;
    color: #6b7280;
    font-size: 10px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    padding: 0 6px;
    flex-shrink: 0;
  }

  .refs-item {
    display: flex;
    align-items: baseline;
    gap: 10px;
    padding: 3px 12px 3px 30px;
    cursor: pointer;
  }
  .refs-item:hover {
    background: rgba(99, 102, 241, 0.12);
  }
  .refs-line {
    color: #6b7280;
    font-size: 11px;
    font-family: 'Fira Code', monospace;
    min-width: 28px;
    text-align: right;
    flex-shrink: 0;
  }
  .refs-preview {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #b4b4bc;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .refs-item:hover .refs-preview {
    color: #e3e3e6;
  }
</style>
