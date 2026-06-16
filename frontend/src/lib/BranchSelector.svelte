<script>
  import { onMount, onDestroy } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let filterText = $state('');
  let selectedIndex = $state(0);
  let inputRef = $state(null);

  // Filter list of branches based on query
  const filteredBranches = $derived.by(() => {
    const branches = store.git.branches || [];
    if (!filterText) return branches.slice(0, 15);
    const query = filterText.toLowerCase();
    return branches
      .filter(branch => branch.toLowerCase().includes(query))
      .slice(0, 15);
  });

  // Keep index in range
  $effect(() => {
    if (selectedIndex >= filteredBranches.length) {
      selectedIndex = Math.max(0, filteredBranches.length - 1);
    }
  });

  // Automatically focus input on mount or when visibility changes
  $effect(() => {
    if (store.branchSelectorVisible) {
      filterText = '';
      selectedIndex = 0;
      setTimeout(() => inputRef?.focus(), 50);
    }
  });

  function handleKeyDown(e) {
    if (!store.branchSelectorVisible) return;

    if (e.key === 'Escape') {
      store.branchSelectorVisible = false;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIndex = (selectedIndex + 1) % Math.max(1, filteredBranches.length);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIndex = (selectedIndex - 1 + filteredBranches.length) % Math.max(1, filteredBranches.length);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (filteredBranches[selectedIndex]) {
        store.checkoutBranch(filteredBranches[selectedIndex]);
      }
    }
  }

  function handleSelect(branch) {
    store.checkoutBranch(branch);
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeyDown);
  });

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeyDown);
  });
</script>

{#if store.branchSelectorVisible}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="overlay" onclick={() => store.branchSelectorVisible = false}>
    <div class="palette-card" onclick={e => e.stopPropagation()}>
      {#if store.isCheckingOut}
        <div class="loading-state">
          <div class="spinner"></div>
          <span>Switching to branch...</span>
        </div>
      {:else}
        <div class="search-input-wrapper">
          <input
            bind:this={inputRef}
            type="text"
            placeholder="Search branches... (e.g. main)"
            bind:value={filterText}
            aria-label="Search branches"
          />
        </div>

        <div class="branches-list">
          {#if filteredBranches.length === 0}
            <div class="no-results">No matching branches found</div>
          {:else}
            {#each filteredBranches as branch, idx}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="branch-row"
                class:selected={idx === selectedIndex}
                class:current={branch === store.git.currentBranch}
                onclick={() => handleSelect(branch)}
                onmouseenter={() => selectedIndex = idx}
              >
                <Icon name="branch" size={14} color={branch === store.git.currentBranch ? '#6366f1' : '#8e8e93'} />
                <div class="branch-info">
                  <span class="branch-name">{branch}</span>
                  {#if branch === store.git.currentBranch}
                    <span class="current-badge">current</span>
                  {/if}
                </div>
              </div>
            {/each}
          {/if}
        </div>

        <div class="palette-footer">
          <span>Use <kbd>↑</kbd> <kbd>↓</kbd> to navigate, <kbd>Enter</kbd> to switch branch, <kbd>Esc</kbd> to close</span>
        </div>
      {/if}
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
    width: 500px;
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

  .branches-list {
    flex: 1;
    overflow-y: auto;
    padding: 6px 0;
  }

  .branches-list::-webkit-scrollbar {
    width: 6px;
  }
  .branches-list::-webkit-scrollbar-track {
    background: transparent;
  }
  .branches-list::-webkit-scrollbar-thumb {
    background: #2d2d34;
    border-radius: 3px;
  }

  .no-results {
    padding: 16px;
    color: #8e8e93;
    font-size: 13px;
    text-align: center;
  }

  .branch-row {
    display: flex;
    align-items: center;
    padding: 8px 16px;
    cursor: pointer;
    gap: 10px;
    user-select: none;
  }

  .branch-row.selected {
    background-color: rgba(99, 102, 241, 0.15);
  }

  .branch-row.current .branch-name {
    color: #6366f1;
    font-weight: 600;
  }

  .branch-info {
    display: flex;
    align-items: center;
    flex: 1;
    gap: 8px;
  }

  .branch-name {
    color: #e3e3e6;
    font-size: 13px;
    font-weight: 500;
  }

  .current-badge {
    background-color: rgba(99, 102, 241, 0.2);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.3);
    font-size: 10px;
    padding: 1px 4px;
    border-radius: 4px;
    text-transform: uppercase;
    font-weight: bold;
    letter-spacing: 0.5px;
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

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px;
    gap: 16px;
    color: #e3e3e6;
    font-size: 14px;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid rgba(255, 255, 255, 0.1);
    border-top-color: #6366f1;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
