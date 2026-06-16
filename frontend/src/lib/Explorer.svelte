<script>
  import { store } from './store.svelte.js';
  import FileNode from './FileNode.svelte';
  import Icon from './Icon.svelte';

  const rootFiles = $derived(store.dirContents[''] || []);
</script>

<div class="explorer-panel">
  <div class="panel-header">
    <span>EXPLORER</span>
  </div>

  <div class="workspace-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="workspace-header" onclick={() => store.toggleFolder('')}>
      <div class="chevron" class:open={store.expandedPaths.has('')}>
        <Icon name="chevronRight" size={14} color="#8e8e93" />
      </div>
      <span class="workspace-name">{store.workspace.name}</span>
    </div>

    {#if store.expandedPaths.has('')}
      <div class="tree-container">
        {#if rootFiles.length === 0}
          <div class="empty-state">Workspace is empty</div>
        {:else}
          {#each rootFiles as child (child.path)}
            <FileNode item={child} />
          {/each}
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .explorer-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    color: #e3e3e6;
  }

  .panel-header {
    height: 36px;
    padding: 0 16px;
    display: flex;
    align-items: center;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.8px;
    color: #8e8e93;
    border-bottom: 1px solid #2d2d34;
    text-transform: uppercase;
  }

  .workspace-section {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .workspace-header {
    display: flex;
    align-items: center;
    height: 32px;
    padding: 0 8px;
    cursor: pointer;
    font-weight: 700;
    font-size: 11px;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: #c5c5c5;
    background-color: rgba(255, 255, 255, 0.02);
    border-bottom: 1px solid rgba(255, 255, 255, 0.03);
    user-select: none;
  }

  .workspace-header:hover {
    color: #ffffff;
  }

  .chevron {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    margin-right: 4px;
    transition: transform 0.15s ease;
  }

  .chevron.open {
    transform: rotate(90deg);
  }

  .workspace-name {
    font-family: inherit;
  }

  .tree-container {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
  }

  /* Custom scrollbar for file tree */
  .tree-container::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  .tree-container::-webkit-scrollbar-track {
    background: transparent;
  }
  .tree-container::-webkit-scrollbar-thumb {
    background: #2d2d34;
    border-radius: 3px;
  }
  .tree-container::-webkit-scrollbar-thumb:hover {
    background: #3e3e4a;
  }

  .empty-state {
    padding: 16px;
    font-size: 12px;
    color: #6b7280;
    font-style: italic;
    text-align: center;
  }
</style>
