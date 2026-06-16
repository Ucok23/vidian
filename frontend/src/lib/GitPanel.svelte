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

  // Accordion open/collapse states
  let showChanges = $state(true);
  let showCommits = $state(true);
  let showBranches = $state(false);

  // Expanded commit hash for detailed changed files list
  let expandedCommit = $state(null);
  let commitFiles = $state({}); // commitHash -> array of changed files
  let isLoadingCommitFiles = $state(false);

  async function loadChanges() {
    isLoadingChanges = true;
    try {
      const res = await fetch('/api/git/changes');
      changes = await res.json();
    } catch (err) {
      console.error("Failed to load changes", err);
    } finally {
      isLoadingChanges = false;
    }
  }

  async function loadCommits() {
    isLoadingCommits = true;
    try {
      const res = await fetch('/api/git/log');
      commits = await res.json();
    } catch (err) {
      console.error("Failed to load commits", err);
    } finally {
      isLoadingCommits = false;
    }
  }

  async function toggleCommit(hash) {
    if (expandedCommit === hash) {
      expandedCommit = null;
      return;
    }
    expandedCommit = hash;
    if (!commitFiles[hash]) {
      isLoadingCommitFiles = true;
      try {
        const res = await fetch(`/api/git/commit/files?commit=${hash}`);
        commitFiles = { ...commitFiles, [hash]: await res.json() };
      } catch (err) {
        console.error("Failed to load commit files", hash, err);
      } finally {
        isLoadingCommitFiles = false;
      }
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
              <div class="commit-item" class:expanded={expandedCommit === commit.hash}>
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="commit-summary" onclick={() => toggleCommit(commit.hash)}>
                  <div class="commit-header">
                    <span class="commit-author">{commit.author}</span>
                    <span class="commit-hash">{commit.hash.slice(0, 7)}</span>
                  </div>
                  <div class="commit-msg">{commit.summary}</div>
                  <div class="commit-date">{commit.relative}</div>
                </div>

                {#if expandedCommit === commit.hash}
                  <div class="commit-details">
                    <div class="details-meta">
                      <div><strong>Author:</strong> {commit.author} &lt;{commit.email}&gt;</div>
                      <div><strong>Date:</strong> {commit.date}</div>
                    </div>
                    <div class="details-files">
                      <div class="files-title">Files Changed:</div>
                      {#if isLoadingCommitFiles}
                        <div class="loading-text">Loading files...</div>
                      {:else if commitFiles[commit.hash]}
                        {#each commitFiles[commit.hash] as file}
                          <!-- svelte-ignore a11y_click_events_have_key_events -->
                          <!-- svelte-ignore a11y_no_static_element_interactions -->
                          <div class="commit-file-row" onclick={() => store.openDiff(file.path, `${commit.hash}^`, commit.hash, `${file.path.split('/').pop()} (${commit.hash.slice(0, 7)})`)}>
                            <span class="status-marker" style="color: {getStatusColor(file.status)}">
                              {file.status}
                            </span>
                            <span class="file-path-link">{file.path}</span>
                          </div>
                        {/each}
                      {/if}
                    </div>
                  </div>
                {/if}
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

  .commit-details {
    background-color: #141416;
    padding: 10px 16px;
    border-top: 1px solid #24242b;
    font-size: 12px;
  }

  .details-meta {
    color: #8e8e93;
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 10px;
    font-size: 11px;
  }

  .details-files {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .files-title {
    font-weight: bold;
    color: #e3e3e6;
    font-size: 11px;
    margin-bottom: 2px;
  }

  .commit-file-row {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    padding: 2px 0;
  }

  .commit-file-row:hover .file-path-link {
    text-decoration: underline;
    color: #e3e3e6;
  }

  .status-marker {
    font-family: monospace;
    font-weight: bold;
    font-size: 11px;
    width: 12px;
    text-align: center;
  }

  .file-path-link {
    color: #a1a1aa;
    font-size: 11px;
    word-break: break-all;
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
