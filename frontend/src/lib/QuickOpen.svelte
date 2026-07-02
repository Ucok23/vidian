<script>
  import { onMount, onDestroy } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let isOpen = $state(false);
  let filesList = $state([]);
  let filterText = $state('');
  let selectedIndex = $state(0);
  let inputRef = $state(null);

  // Filter list based on query
  const filteredFiles = $derived.by(() => {
    if (!filterText) return filesList.slice(0, 15);
    const query = filterText.toLowerCase();
    return filesList
      .filter(file => file.toLowerCase().includes(query))
      .slice(0, 15);
  });

  // Keep index in range
  $effect(() => {
    if (selectedIndex >= filteredFiles.length) {
      selectedIndex = Math.max(0, filteredFiles.length - 1);
    }
  });

  async function loadFiles() {
    try {
      const res = await fetch(store.apiUrl('/api/files'));
      filesList = await res.json();
    } catch (err) {
      console.error("Failed to load files for Quick Open", err);
    }
  }

  function handleKeyDown(e) {
    // Ctrl+P or Cmd+P
    if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
      e.preventDefault();
      isOpen = !isOpen;
      if (isOpen) {
        filterText = '';
        selectedIndex = 0;
        loadFiles();
        // Focus input after DOM updates
        setTimeout(() => inputRef?.focus(), 50);
      }
    }

    if (!isOpen) return;

    if (e.key === 'Escape') {
      isOpen = false;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIndex = (selectedIndex + 1) % Math.max(1, filteredFiles.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIndex = (selectedIndex - 1 + filteredFiles.length) % Math.max(1, filteredFiles.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (filteredFiles[selectedIndex]) {
        store.openFile(filteredFiles[selectedIndex]);
        isOpen = false;
      }
    }
  }

  function handleSelect(path) {
    store.openFile(path);
    isOpen = false;
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
      json: '#cbcb41',
      md: '#0891b2',
      go: '#00add8',
      py: '#3572a5',
      rs: '#dea584',
      svelte: '#ff3e00'
    };
    return colors[ext] || '#9ca3af';
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeyDown);
  });

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeyDown);
  });
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="overlay" onclick={() => isOpen = false}>
    <div class="palette-card" onclick={e => e.stopPropagation()}>
      <div class="search-input-wrapper">
        <input
          bind:this={inputRef}
          type="text"
          placeholder="Search files by name... (e.g. App.svelte)"
          bind:value={filterText}
          aria-label="Quick Open Search"
        />
      </div>

      <div class="files-list">
        {#if filteredFiles.length === 0}
          <div class="no-results">No matching files found</div>
        {:else}
          {#each filteredFiles as file, idx}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div
              class="file-row"
              class:selected={idx === selectedIndex}
              onclick={() => handleSelect(file)}
              onmouseenter={() => selectedIndex = idx}
            >
              <Icon name="file" size={16} color={getFileColor(file.split('/').pop())} />
              <div class="file-info">
                <span class="file-name">{file.split('/').pop()}</span>
                <span class="file-path">{file}</span>
              </div>
            </div>
          {/each}
        {/if}
      </div>

      <div class="palette-footer">
        <span>Use <kbd>↑</kbd> <kbd>↓</kbd> to navigate, <kbd>Enter</kbd> to open, <kbd>Esc</kbd> to close</span>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 9999;
    display: flex;
    justify-content: center;
    padding-top: 50px;
    backdrop-filter: blur(2px);
  }

  .palette-card {
    background-color: #1e1e24;
    border: 1px solid #2d2d34;
    border-radius: 8px;
    width: 600px;
    max-height: 380px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.4);
    overflow: hidden;
    animation: slideDown 0.15s ease-out;
  }

  @keyframes slideDown {
    from {
      transform: translateY(-20px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }

  .search-input-wrapper {
    padding: 12px;
    border-bottom: 1px solid #2d2d34;
  }

  .search-input-wrapper input {
    width: 100%;
    height: 32px;
    background-color: #121214;
    border: 1px solid #2d2d34;
    border-radius: 4px;
    color: #e3e3e6;
    font-size: 14px;
    padding: 0 12px;
    box-sizing: border-box;
    outline: none;
  }

  .search-input-wrapper input:focus {
    border-color: #6366f1;
  }

  .files-list {
    flex: 1;
    overflow-y: auto;
    padding: 6px 0;
  }

  .files-list::-webkit-scrollbar {
    width: 6px;
  }
  .files-list::-webkit-scrollbar-track {
    background: transparent;
  }
  .files-list::-webkit-scrollbar-thumb {
    background: #2d2d34;
    border-radius: 3px;
  }

  .no-results {
    padding: 16px;
    color: #8e8e93;
    font-size: 13px;
    text-align: center;
  }

  .file-row {
    display: flex;
    align-items: center;
    padding: 8px 16px;
    cursor: pointer;
    gap: 10px;
    user-select: none;
  }

  .file-row.selected {
    background-color: rgba(99, 102, 241, 0.15);
  }

  .file-info {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .file-name {
    color: #e3e3e6;
    font-size: 13px;
    font-weight: 500;
  }

  .file-path {
    color: #8e8e93;
    font-size: 11px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .palette-footer {
    padding: 8px 16px;
    border-top: 1px solid #2d2d34;
    background-color: #141416;
    color: #6b7280;
    font-size: 11px;
    text-align: right;
  }

  kbd {
    background-color: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 3px;
    padding: 1px 4px;
    font-family: inherit;
    font-size: 10px;
  }
</style>
