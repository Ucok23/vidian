<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let changes = $state([]);
  let commits = $state([]);
  let branches = $state([]);
  let currentBranch = $state('');
  let isLoadingChanges = $state(false);
  let isLoadingCommits = $state(false);
  let fileHistory = $state([]);
  let isLoadingFileHistory = $state(false);

  // New sections state
  let contributors = $state([]);
  let isLoadingContributors = $state(false);
  let stashes = $state([]);
  let isLoadingStashes = $state(false);
  let tags = $state([]);
  let isLoadingTags = $state(false);
  let searchQuery = $state('');
  let searchAuthor = $state('');
  let searchResults = $state([]);
  let isSearching = $state(false);
  let compareRef1 = $state('');
  let compareRef2 = $state('');
  let isComparing = $state(false);

  // Accordion open/collapse states
  let showChanges = $state(true);
  let showCommits = $state(true);
  let showBranches = $state(false);
  let showFileHistory = $state(true);
  let showLineHistory = $state(true);
  let showSearch = $state(false);
  let showContributors = $state(false);
  let showStashes = $state(false);
  let showTags = $state(false);
  let showCompare = $state(false);

  async function loadChanges() {
    isLoadingChanges = true;
    try {
      const res = await fetch('/api/git/changes');
      changes = (await res.json()) || [];
    } catch (err) {
      console.error("Failed to load changes", err);
    } finally {
      isLoadingChanges = false;
    }
  }

  async function loadFileHistory(path) {
    isLoadingFileHistory = true;
    try {
      const res = await fetch(`/api/git/log?path=${encodeURIComponent(path)}`);
      fileHistory = (await res.json()) || [];
    } catch (err) {
      console.error("Failed to load file history", err);
      fileHistory = [];
    } finally {
      isLoadingFileHistory = false;
    }
  }

  $effect(() => {
    const path = store.activePath;
    if (!path || path.startsWith('commit:')) {
      fileHistory = [];
      return;
    }
    loadFileHistory(path);
  });

  async function loadContributors() {
    isLoadingContributors = true;
    try {
      const res = await fetch('/api/git/contributors');
      contributors = (await res.json()) || [];
    } catch (e) { contributors = []; }
    finally { isLoadingContributors = false; }
  }

  async function loadStashes() {
    isLoadingStashes = true;
    try {
      const res = await fetch('/api/git/stashes');
      stashes = (await res.json()) || [];
    } catch (e) { stashes = []; }
    finally { isLoadingStashes = false; }
  }

  async function loadTags() {
    isLoadingTags = true;
    try {
      const res = await fetch('/api/git/tags');
      tags = (await res.json()) || [];
    } catch (e) { tags = []; }
    finally { isLoadingTags = false; }
  }

  async function runSearch() {
    if (!searchQuery && !searchAuthor) return;
    isSearching = true;
    try {
      const params = new URLSearchParams();
      if (searchQuery) params.set('q', searchQuery);
      if (searchAuthor) params.set('author', searchAuthor);
      const res = await fetch(`/api/git/search?${params}`);
      searchResults = (await res.json()) || [];
    } catch (e) { searchResults = []; }
    finally { isSearching = false; }
  }

  async function runCompare() {
    if (!compareRef1 || !compareRef2) return;
    isComparing = true;
    try {
      const res = await fetch(`/api/git/compare?ref1=${encodeURIComponent(compareRef1)}&ref2=${encodeURIComponent(compareRef2)}`);
      store.compareResult = await res.json();
    } catch (e) { store.compareResult = null; }
    finally { isComparing = false; }
  }

  async function loadCommits() {
    isLoadingCommits = true;
    try {
      const res = await fetch('/api/git/log');
      commits = (await res.json()) || [];
    } catch (err) {
      console.error("Failed to load commits", err);
    } finally {
      isLoadingCommits = false;
    }
  }

  function getStatusColor(status) {
    if (!status) return '#8e8e93';
    switch (status.toUpperCase()) {
      case 'M': return '#e2c08d'; // Modified (amber)
      case 'A':
      case '??': return '#73c991'; // Added/Untracked (green)
      case 'D': return '#f14c4c'; // Deleted (red)
      default: return '#8e8e93';
    }
  }

  function getStatusLabel(status) {
    if (status === '??') return 'U';
    return status;
  }

  onMount(async () => {
    await loadChanges();
    await loadCommits();
    branches = store.git.branches || [];
    currentBranch = store.git.currentBranch || '';
  });
</script>

<div class="git-panel">
  <!-- TOOLBAR -->
  <div class="panel-toolbar">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <button class="graph-btn" onclick={() => store.openGraph()} title="Open the commit graph in the main area">
      <Icon name="graph" size={15} />
      <span>Open Commit Graph</span>
    </button>
  </div>

  <!-- CHANGES SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => showChanges = !showChanges}>
      <Icon name={showChanges ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">CHANGES</span>
      {#if changes.length > 0}
        <span class="badge">{changes.length}</span>
      {/if}
    </div>

    {#if showChanges}
      <div class="section-content">
        {#if isLoadingChanges}
          <div class="loading-text">Loading changes...</div>
        {:else if changes.length === 0}
          <div class="empty-state">No uncommitted changes</div>
        {:else}
          <div class="changes-list">
            {#each changes as change}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="change-item" onclick={() => store.openDiff(change.path, 'HEAD', '', `${change.path.split('/').pop()} (Working Tree ↔ HEAD)`)}>
                <Icon name="file" size={14} color="#8e8e93" />
                <div class="file-info">
                  <span class="file-name">{change.path.split('/').pop()}</span>
                  <span class="file-path">{change.path}</span>
                </div>
                <span class="status-badge" style="color: {getStatusColor(change.status)}">
                  {getStatusLabel(change.status)}
                </span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- COMMITS / HISTORY SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => showCommits = !showCommits}>
      <Icon name={showCommits ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">COMMITS (HISTORY)</span>
    </div>

    {#if showCommits}
      <div class="section-content">
        {#if isLoadingCommits}
          <div class="loading-text">Loading history...</div>
        {:else if commits.length === 0}
          <div class="empty-state">No commits found</div>
        {:else}
          <div class="commits-list">
            {#each commits as commit}
              <div 
                class="commit-item" 
                class:active={store.activePath === `commit:${commit.hash}`}
              >
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="commit-summary" onclick={() => store.openCommit(commit.hash)}>
                  <div class="commit-header">
                    <span class="commit-author">{commit.author}</span>
                    <span class="commit-hash">{commit.hash.slice(0, 7)}</span>
                  </div>
                  <div class="commit-msg">{commit.summary}</div>
                  <div class="commit-date">{commit.relative}</div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- FILE HISTORY SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => showFileHistory = !showFileHistory}>
      <Icon name={showFileHistory ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">FILE HISTORY</span>
      {#if store.activePath && !store.activePath.startsWith('commit:')}
        <span class="file-history-name">{store.activePath.split('/').pop()}</span>
      {/if}
    </div>

    {#if showFileHistory}
      <div class="section-content">
        {#if !store.activePath || store.activePath.startsWith('commit:')}
          <div class="empty-state">Open a file to see its history</div>
        {:else if isLoadingFileHistory}
          <div class="loading-text">Loading history...</div>
        {:else if fileHistory.length === 0}
          <div class="empty-state">No history found</div>
        {:else}
          <div class="commits-list">
            {#each fileHistory as commit}
              <div
                class="commit-item"
                class:active={store.activePath === `commit:${commit.hash}`}
              >
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="commit-summary" onclick={() => store.openCommit(commit.hash)}>
                  <div class="commit-header">
                    <span class="commit-author">{commit.author}</span>
                    <span class="commit-hash">{commit.hash.slice(0, 7)}</span>
                  </div>
                  <div class="commit-msg">{commit.summary}</div>
                  <div class="commit-date">{commit.relative}</div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- LINE HISTORY SECTION (appears when triggered from editor) -->
  {#if store.lineHistory}
    <div class="panel-section">
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="section-header" onclick={() => showLineHistory = !showLineHistory}>
        <Icon name={showLineHistory ? 'chevronDown' : 'chevronRight'} size={14} />
        <span class="title">LINE HISTORY</span>
        <span class="file-history-name">L{store.lineHistory.start}–{store.lineHistory.end}</span>
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <span class="close-section" onclick={(e) => { e.stopPropagation(); store.lineHistory = null; }}>✕</span>
      </div>
      {#if showLineHistory}
        <div class="section-content">
          {#if store.lineHistory.commits.length === 0}
            <div class="empty-state">No history found for these lines</div>
          {:else}
            <div class="commits-list">
              {#each store.lineHistory.commits as commit}
                <div class="commit-item" class:active={store.activePath === `commit:${commit.hash}`}>
                  <!-- svelte-ignore a11y_click_events_have_key_events -->
                  <!-- svelte-ignore a11y_no_static_element_interactions -->
                  <div class="commit-summary" onclick={() => store.openCommit(commit.hash)}>
                    <div class="commit-header">
                      <span class="commit-author">{commit.author}</span>
                      <span class="commit-hash">{commit.hash.slice(0, 7)}</span>
                    </div>
                    <div class="commit-msg">{commit.summary}</div>
                    <div class="commit-date">{commit.relative}</div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  {/if}

  <!-- SEARCH COMMITS SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => showSearch = !showSearch}>
      <Icon name={showSearch ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">SEARCH COMMITS</span>
    </div>
    {#if showSearch}
      <div class="section-content search-section">
        <input class="search-input" placeholder="Message / keyword…" bind:value={searchQuery} onkeydown={(e) => e.key === 'Enter' && runSearch()} />
        <input class="search-input" placeholder="Author…" bind:value={searchAuthor} onkeydown={(e) => e.key === 'Enter' && runSearch()} />
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="search-btn" onclick={runSearch} class:disabled={isSearching}>
          {isSearching ? 'Searching…' : 'Search'}
        </div>
        {#if searchResults.length > 0}
          <div class="commits-list">
            {#each searchResults as commit}
              <div class="commit-item">
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="commit-summary" onclick={() => store.openCommit(commit.hash)}>
                  <div class="commit-header">
                    <span class="commit-author">{commit.author}</span>
                    <span class="commit-hash">{commit.hash.slice(0, 7)}</span>
                  </div>
                  <div class="commit-msg">{commit.summary}</div>
                  <div class="commit-date">{commit.relative}</div>
                </div>
              </div>
            {/each}
          </div>
        {:else if !isSearching && (searchQuery || searchAuthor)}
          <div class="empty-state">No commits found</div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- COMPARE REFS SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => showCompare = !showCompare}>
      <Icon name={showCompare ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">COMPARE REFS</span>
    </div>
    {#if showCompare}
      <div class="section-content search-section">
        <input class="search-input" placeholder="Base ref (e.g. main)" bind:value={compareRef1} />
        <input class="search-input" placeholder="Compare ref (e.g. HEAD)" bind:value={compareRef2} />
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="search-btn" onclick={runCompare} class:disabled={isComparing || !compareRef1 || !compareRef2}>
          {isComparing ? 'Comparing…' : 'Compare'}
        </div>
        {#if store.compareResult}
          <div class="compare-stat">{store.compareResult.stat}</div>
          <div class="changes-list">
            {#each store.compareResult.files as file}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="change-item" onclick={() => store.openDiff(file.path, compareRef1, compareRef2, `${file.path.split('/').pop()} (${compareRef1}…${compareRef2})`)}>
                <div class="file-info">
                  <span class="file-name">{file.path.split('/').pop()}</span>
                  <span class="file-path">{file.path}</span>
                </div>
                <span class="status-badge" style="color: {file.status === 'A' ? '#73c991' : file.status === 'D' ? '#f14c4c' : '#e2c08d'}">{file.status}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- CONTRIBUTORS SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => { showContributors = !showContributors; if (showContributors && contributors.length === 0) loadContributors(); }}>
      <Icon name={showContributors ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">CONTRIBUTORS</span>
    </div>
    {#if showContributors}
      <div class="section-content">
        {#if isLoadingContributors}
          <div class="loading-text">Loading…</div>
        {:else if contributors.length === 0}
          <div class="empty-state">No contributors found</div>
        {:else}
          <div class="contributors-list">
            {#each contributors as c, i}
              <div class="contributor-item">
                <div class="contributor-avatar">{c.name.charAt(0).toUpperCase()}</div>
                <div class="contributor-info">
                  <span class="contributor-name">{c.name}</span>
                  <span class="contributor-email">{c.email}</span>
                </div>
                <span class="contributor-count">{c.commits}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- STASHES SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => { showStashes = !showStashes; if (showStashes && stashes.length === 0) loadStashes(); }}>
      <Icon name={showStashes ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">STASHES</span>
    </div>
    {#if showStashes}
      <div class="section-content">
        {#if isLoadingStashes}
          <div class="loading-text">Loading…</div>
        {:else if stashes.length === 0}
          <div class="empty-state">No stashes</div>
        {:else}
          <div class="commits-list">
            {#each stashes as stash}
              <div class="commit-item">
                <div class="commit-summary no-click">
                  <div class="commit-header">
                    <span class="commit-author">stash@{'{' + stash.index + '}'}</span>
                    <span class="commit-hash">{stash.hash.slice(0, 7)}</span>
                  </div>
                  <div class="commit-msg">{stash.message}</div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- TAGS SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => { showTags = !showTags; if (showTags && tags.length === 0) loadTags(); }}>
      <Icon name={showTags ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">TAGS</span>
    </div>
    {#if showTags}
      <div class="section-content">
        {#if isLoadingTags}
          <div class="loading-text">Loading…</div>
        {:else if tags.length === 0}
          <div class="empty-state">No tags</div>
        {:else}
          <div class="tags-list">
            {#each tags as tag}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="tag-item" onclick={() => store.openCommit(tag.hash)}>
                <div class="tag-info">
                  <span class="tag-name">{tag.name}</span>
                  {#if tag.message}<span class="tag-msg">{tag.message}</span>{/if}
                </div>
                <span class="tag-date">{tag.date}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- BRANCHES SECTION -->
  <div class="panel-section">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="section-header" onclick={() => showBranches = !showBranches}>
      <Icon name={showBranches ? 'chevronDown' : 'chevronRight'} size={14} />
      <span class="title">BRANCHES</span>
    </div>

    {#if showBranches}
      <div class="section-content">
        <div class="branches-list">
          {#each branches as branch}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div
              class="branch-item"
              class:current={branch === currentBranch}
              onclick={() => store.checkoutBranch(branch)}
            >
              <Icon name="branch" size={12} color={branch === currentBranch ? '#6366f1' : '#8e8e93'} />
              <span class="branch-name">{branch}</span>
              {#if branch === currentBranch}
                <span class="current-badge">current</span>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .git-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    color: #e3e3e6;
    background-color: #1b1b1f;
    font-size: 13px;
    overflow-y: auto;
  }

  .panel-section {
    border-bottom: 1px solid #2d2d34;
  }

  .section-header {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    cursor: pointer;
    background-color: #232329;
    user-select: none;
    font-weight: 600;
    font-size: 11px;
    letter-spacing: 0.5px;
    color: #8e8e93;
    gap: 6px;
  }

  .section-header:hover {
    color: #e3e3e6;
  }

  .section-header .title {
    flex: 1;
  }

  .file-history-name {
    font-size: 10px;
    color: #6366f1;
    font-weight: normal;
    font-family: monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100px;
  }

  .badge {
    background-color: #3e3e4a;
    color: #e3e3e6;
    font-size: 10px;
    padding: 1px 6px;
    border-radius: 10px;
    font-weight: bold;
  }

  .section-content {
    background-color: #1b1b1f;
  }

  .loading-text, .empty-state {
    padding: 16px 12px;
    color: #8e8e93;
    font-style: italic;
    font-size: 12px;
    text-align: center;
  }

  .changes-list {
    display: flex;
    flex-direction: column;
    padding: 4px 0;
  }

  .change-item {
    display: flex;
    align-items: center;
    padding: 6px 16px;
    cursor: pointer;
    gap: 8px;
    user-select: none;
  }

  .change-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
  }

  .file-info {
    display: flex;
    flex-direction: column;
    flex: 1;
    overflow: hidden;
  }

  .file-name {
    color: #e3e3e6;
    font-size: 13px;
  }

  .file-path {
    color: #8e8e93;
    font-size: 11px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .status-badge {
    font-size: 11px;
    font-weight: bold;
    padding: 0 4px;
  }

  /* Commits style */
  .commits-list {
    display: flex;
    flex-direction: column;
  }

  .commit-item {
    border-bottom: 1px solid #24242b;
  }

  .commit-summary {
    padding: 8px 16px;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .commit-summary:hover {
    background-color: rgba(255, 255, 255, 0.03);
  }

  .commit-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .commit-author {
    font-weight: bold;
    color: #818cf8;
    font-size: 12px;
  }

  .commit-hash {
    font-family: monospace;
    color: #8e8e93;
    font-size: 11px;
  }

  .commit-msg {
    color: #e3e3e6;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .commit-date {
    color: #8e8e93;
    font-size: 11px;
  }

  .commit-item.active .commit-summary {
    background-color: rgba(99, 102, 241, 0.12);
    border-left: 2px solid #6366f1;
    padding-left: 14px;
  }

  /* Branches list */
  .branches-list {
    display: flex;
    flex-direction: column;
    padding: 6px 0;
  }

  .branch-item {
    display: flex;
    align-items: center;
    padding: 8px 16px;
    cursor: pointer;
    gap: 8px;
  }

  .branch-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
  }

  .branch-item.current {
    background-color: rgba(99, 102, 241, 0.05);
  }

  .branch-item.current .branch-name {
    color: #6366f1;
    font-weight: bold;
  }

  .branch-name {
    color: #e3e3e6;
    font-size: 12px;
    flex: 1;
  }

  .close-section {
    font-size: 11px;
    color: #6b6b74;
    cursor: pointer;
    padding: 0 2px;
    border-radius: 3px;
    margin-left: 4px;
  }
  .close-section:hover { color: #e3e3e6; }

  .search-section {
    padding: 8px 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .search-input {
    background: #141418;
    border: 1px solid #2d2d34;
    border-radius: 4px;
    color: #e3e3e6;
    font-size: 12px;
    padding: 5px 8px;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }
  .search-input:focus { border-color: #6366f1; }

  .search-btn {
    background: rgba(99, 102, 241, 0.15);
    border: 1px solid rgba(99, 102, 241, 0.3);
    color: #818cf8;
    border-radius: 4px;
    padding: 5px 10px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    text-align: center;
    user-select: none;
    transition: background 0.15s;
  }
  .search-btn:hover { background: rgba(99, 102, 241, 0.25); }
  .search-btn.disabled { opacity: 0.4; pointer-events: none; }

  .compare-stat {
    font-family: 'Fira Code', monospace;
    font-size: 10px;
    color: #8e8e93;
    background: #141418;
    border: 1px solid #2d2d34;
    border-radius: 4px;
    padding: 6px 8px;
    white-space: pre;
    overflow-x: auto;
  }

  .contributors-list { display: flex; flex-direction: column; padding: 4px 0; }

  .contributor-item {
    display: flex;
    align-items: center;
    padding: 7px 16px;
    gap: 10px;
  }

  .contributor-avatar {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: rgba(99, 102, 241, 0.2);
    color: #818cf8;
    font-size: 12px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .contributor-info { flex: 1; min-width: 0; }
  .contributor-name { display: block; font-size: 12px; color: #e3e3e6; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .contributor-email { display: block; font-size: 10px; color: #8e8e93; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .contributor-count { font-size: 11px; font-weight: 700; color: #818cf8; }

  .tags-list { display: flex; flex-direction: column; padding: 4px 0; }

  .tag-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px 16px;
    cursor: pointer;
    gap: 8px;
  }
  .tag-item:hover { background: rgba(255,255,255,0.04); }

  .tag-info { display: flex; flex-direction: column; min-width: 0; }
  .tag-name { font-size: 12px; color: #fbbf24; font-weight: 600; }
  .tag-msg { font-size: 11px; color: #8e8e93; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .tag-date { font-size: 11px; color: #8e8e93; white-space: nowrap; flex-shrink: 0; }

  .commit-summary.no-click { cursor: default; }
  .commit-summary.no-click:hover { background: none; }

  .panel-toolbar {
    padding: 8px 10px;
    border-bottom: 1px solid #2d2d34;
    flex-shrink: 0;
  }
  .graph-btn {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 7px 10px;
    background: rgba(99, 102, 241, 0.12);
    border: 1px solid rgba(99, 102, 241, 0.3);
    border-radius: 6px;
    color: #a5b4fc;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }
  .graph-btn:hover {
    background: rgba(99, 102, 241, 0.22);
    color: #c7d2fe;
  }

  .current-badge {
    background-color: rgba(99, 102, 241, 0.2);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.3);
    font-size: 9px;
    padding: 1px 4px;
    border-radius: 4px;
    text-transform: uppercase;
    font-weight: bold;
  }
</style>
