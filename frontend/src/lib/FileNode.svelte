<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let { item, depth = 0 } = $props();

  const isExpanded = $derived(store.expandedPaths.has(item.path));
  const childItems = $derived(store.dirContents[item.path] || []);

  function handleToggle() {
    if (item.isDir) {
      store.toggleFolder(item.path);
    } else {
      store.openFile(item.path);
    }
  }

  function getFileColor(name) {
    const ext = name.split('.').pop().toLowerCase();
    const colors = {
      js: '#f7df1e',
      jsx: '#61dafb',
      ts: '#3178c6',
      tsx: '#61dafb',
      html: '#e34c26',
      css: '#264de4',
      scss: '#cc6699',
      json: '#cbcb41',
      md: '#0891b2',
      go: '#00add8',
      py: '#3572a5',
      rs: '#dea584',
      sh: '#4eaa25',
      svelte: '#ff3e00',
      svg: '#ffb13b',
      png: '#a3e635',
      jpg: '#a3e635',
      gif: '#a3e635'
    };
    return colors[ext] || '#9ca3af'; // default grey
  }
</script>

<div class="node-wrapper">
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="node-row"
    class:active={store.activePath === item.path}
    style="padding-left: {depth * 12 + 6}px"
    onclick={handleToggle}
  >
    <div class="chevron-wrapper" class:rotated={isExpanded} class:hidden={!item.isDir}>
      <Icon name="chevronRight" size={14} color="#8e8e93" />
    </div>

    <div class="icon-wrapper">
      {#if item.isDir}
        <Icon name={isExpanded ? 'folderOpen' : 'folder'} size={16} color={isExpanded ? '#6366f1' : '#818cf8'} />
      {:else}
        <Icon name="file" size={16} color={getFileColor(item.name)} />
      {/if}
    </div>

    <span class="node-name">{item.name}</span>
  </div>

  {#if item.isDir && isExpanded}
    <div class="children-container">
      {#if !store.dirContents[item.path]}
        <div class="loading-node" style="padding-left: {(depth + 1) * 12 + 20}px">
          Loading...
        </div>
      {:else}
        {#each childItems as child (child.path)}
          <svelte:self item={child} depth={depth + 1} />
        {:else}
          <div class="loading-node" style="padding-left: {(depth + 1) * 12 + 20}px">
            (Empty folder)
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>

<style>
  .node-wrapper {
    display: flex;
    flex-direction: column;
    width: 100%;
    user-select: none;
  }

  .node-row {
    display: flex;
    align-items: center;
    height: 24px;
    cursor: pointer;
    font-size: 13px;
    color: #c5c5c5;
    transition: background-color 0.15s, color 0.15s;
    border-radius: 4px;
    margin: 1px 4px;
  }

  .node-row:hover {
    background-color: rgba(255, 255, 255, 0.05);
    color: #e3e3e6;
  }

  .node-row.active {
    background-color: rgba(99, 102, 241, 0.2);
    color: #ffffff;
    font-weight: 500;
  }

  .chevron-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    margin-right: 2px;
    transition: transform 0.15s ease;
  }

  .chevron-wrapper.rotated {
    transform: rotate(90deg);
  }

  .chevron-wrapper.hidden {
    visibility: hidden;
  }

  .icon-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 6px;
  }

  .node-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .children-container {
    display: flex;
    flex-direction: column;
  }

  .loading-node {
    font-size: 11px;
    color: #6b7280;
    font-style: italic;
    height: 20px;
    display: flex;
    align-items: center;
  }
</style>
