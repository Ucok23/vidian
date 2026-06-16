<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let inputVal = $state('');
  let searchTimeout = null;

  // Debounced search trigger
  function handleInput(e) {
    inputVal = e.target.value;
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      store.search(inputVal);
    }, 300);
  }

  function handleClear() {
    inputVal = '';
    store.search('');
  }

  // Group search results by filepath for VS Code-like appearance
  const groupedResults = $derived.by(() => {
    const groups = {};
    store.searchResults.forEach(res => {
      if (!groups[res.path]) {
        groups[res.path] = [];
      }
      groups[res.path].push(res);
    });
    return Object.entries(groups).map(([path, matches]) => ({
      path,
      name: path.split('/').pop(),
      matches
    }));
  });

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
</script>

<div class="search-panel">
  <div class="panel-header">
    <span>SEARCH</span>
  </div>

  <div class="search-box">
    <div class="input-wrapper">
      <input
        type="text"
        placeholder="Search in workspace..."
        value={inputVal}
        oninput={handleInput}
        aria-label="Search text"
      />
      {#if inputVal}
        <button class="clear-btn" onclick={handleClear} aria-label="Clear search">
          <Icon name="close" size={12} color="#8e8e93" />
        </button>
      {/if}
    </div>
  </div>

  <div class="results-container">
    {#if store.isSearching}
      <div class="search-status">Searching...</div>
    {:else}
      {#if inputVal && store.searchResults.length === 0}
        <div class="search-status">No results found.</div>
      {:else if store.searchResults.length > 0}
        <div class="results-count">
          {store.searchResults.length} result{store.searchResults.length > 1 ? 's' : ''} found
        </div>
        
        <div class="results-list">
          {#each groupedResults as group (group.path)}
            <div class="file-group">
              <div class="file-header">
                <Icon name="file" size={14} color={getFileColor(group.name)} />
                <span class="file-name">{group.name}</span>
                <span class="file-path">{group.path}</span>
              </div>
              <div class="matches-list">
                {#each group.matches as match}
                  <!-- svelte-ignore a11y_click_events_have_key_events -->
                  <!-- svelte-ignore a11y_no_static_element_interactions -->
                  <div
                    class="match-row"
                    onclick={() => store.openFile(match.path, match.lineNumber)}
                  >
                    {#if match.lineNumber > 0}
                      <span class="line-num">{match.lineNumber}</span>
                      <span class="line-content">{match.lineContent}</span>
                    {:else}
                      <span class="line-num">-</span>
                      <span class="line-content filename-match">{match.lineContent}</span>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
</div>

<style>
  .search-panel {
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

  .search-box {
    padding: 12px 16px;
    border-bottom: 1px solid #2d2d34;
  }

  .input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
    background-color: #1e1e24;
    border: 1px solid #2d2d34;
    border-radius: 4px;
    padding: 2px 6px;
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.2);
  }

  .input-wrapper:focus-within {
    border-color: #6366f1;
    box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  }

  input {
    width: 100%;
    height: 24px;
    background: transparent;
    border: none;
    outline: none;
    color: #e3e3e6;
    font-size: 13px;
    padding-right: 20px;
  }

  input::placeholder {
    color: #5d5d66;
  }

  .clear-btn {
    position: absolute;
    right: 6px;
    background: transparent;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2px;
  }

  .clear-btn:hover :global(svg) {
    stroke: #ffffff;
  }

  .results-container {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0;
  }

  /* Custom scrollbar for search results */
  .results-container::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  .results-container::-webkit-scrollbar-track {
    background: transparent;
  }
  .results-container::-webkit-scrollbar-thumb {
    background: #2d2d34;
    border-radius: 3px;
  }
  .results-container::-webkit-scrollbar-thumb:hover {
    background: #3e3e4a;
  }

  .search-status {
    padding: 16px;
    font-size: 12px;
    color: #8e8e93;
    text-align: center;
    font-style: italic;
  }

  .results-count {
    padding: 0 16px 8px;
    font-size: 12px;
    color: #8e8e93;
    font-weight: 500;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .file-group {
    display: flex;
    flex-direction: column;
  }

  .file-header {
    display: flex;
    align-items: center;
    height: 28px;
    padding: 0 16px;
    background-color: rgba(255, 255, 255, 0.01);
    font-size: 12px;
    user-select: none;
    gap: 6px;
    border-top: 1px solid rgba(255, 255, 255, 0.02);
    border-bottom: 1px solid rgba(255, 255, 255, 0.02);
  }

  .file-name {
    color: #c5c5c5;
    font-weight: 600;
  }

  .file-path {
    color: #5d5d66;
    font-size: 10px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 150px;
    margin-left: 4px;
  }

  .matches-list {
    display: flex;
    flex-direction: column;
    margin-top: 2px;
  }

  .match-row {
    display: flex;
    align-items: center;
    padding: 4px 16px 4px 28px;
    cursor: pointer;
    font-size: 12px;
    color: #a0a0a5;
    transition: background-color 0.1s, color 0.1s;
    user-select: none;
  }

  .match-row:hover {
    background-color: rgba(255, 255, 255, 0.04);
    color: #e3e3e6;
  }

  .line-num {
    color: #5d5d66;
    font-family: monospace;
    width: 24px;
    text-align: right;
    margin-right: 12px;
    flex-shrink: 0;
  }

  .line-content {
    font-family: monospace;
    white-space: pre;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .filename-match {
    color: #6366f1;
    font-style: italic;
  }
</style>
