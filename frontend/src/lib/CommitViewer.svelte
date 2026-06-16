<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let { commit } = $props();

  function getStatusColor(status) {
    switch (status) {
      case 'A': return '#10b981'; // Green for added
      case 'M': return '#fbbf24'; // Amber for modified
      case 'D': return '#ef4444'; // Red for deleted
      default: return '#e2e8f0';
    }
  }
</script>

<div class="commit-viewer-container">
  <div class="commit-header-card">
    <div class="commit-title-row">
      <Icon name="gitCommit" size={24} color="#fbbf24" />
      <h2>{commit.subject}</h2>
    </div>
    
    <div class="commit-meta-grid">
      <div class="meta-item">
        <span class="meta-label">Commit Hash:</span>
        <span class="meta-value hash">{commit.hash}</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Author:</span>
        <span class="meta-value">{commit.author} &lt;{commit.email}&gt;</span>
      </div>
      <div class="meta-item">
        <span class="meta-label">Date:</span>
        <span class="meta-value">{commit.date} ({commit.relative})</span>
      </div>
    </div>
  </div>

  {#if commit.body}
    <div class="commit-body-card">
      <h3>Description</h3>
      <pre class="commit-body-text">{commit.body}</pre>
    </div>
  {/if}

  <div class="commit-files-card">
    <h3>Files Changed ({commit.files ? commit.files.length : 0})</h3>
    <div class="files-list">
      {#if commit.files && commit.files.length > 0}
        {#each commit.files as file}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div 
            class="commit-file-item"
            onclick={() => store.openDiff(
              file.path, 
              `${commit.hash}^`, 
              commit.hash, 
              `${file.path.split('/').pop()} (${commit.hash.slice(0, 7)})`
            )}
            title="Click to view file changes"
          >
            <div class="file-info-group">
              <span class="file-status-badge" style="background-color: {getStatusColor(file.status)}15; color: {getStatusColor(file.status)}">
                {file.status}
              </span>
              <span class="file-path">{file.path}</span>
            </div>
            <div class="file-action-indicator">
              <Icon name="split" size={14} color="#8e8e93" />
              <span class="action-text">View Diff</span>
            </div>
          </div>
        {/each}
      {:else}
        <div class="no-files-text">No changed files found in this commit.</div>
      {/if}
    </div>
  </div>
</div>

<style>
  .commit-viewer-container {
    padding: 2.5rem;
    height: 100%;
    overflow-y: auto;
    background-color: #1b1b20;
    color: #e3e3e6;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .commit-header-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 2rem;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  }

  .commit-title-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 1.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding-bottom: 1rem;
  }

  .commit-title-row h2 {
    margin: 0;
    font-size: 1.4rem;
    font-weight: 600;
    color: #ffffff;
    line-height: 1.3;
  }

  .commit-meta-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1rem;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .meta-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #8e8e93;
  }

  .meta-value {
    font-size: 13px;
    color: #e3e3e6;
  }

  .meta-value.hash {
    font-family: 'Fira Code', monospace;
    color: #a5b4fc;
  }

  .commit-body-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 1.5rem 2rem;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  }

  .commit-body-card h3 {
    margin-top: 0;
    margin-bottom: 1rem;
    font-size: 0.95rem;
    color: #8e8e93;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .commit-body-text {
    margin: 0;
    font-family: inherit;
    font-size: 13px;
    line-height: 1.6;
    color: #e3e3e6;
    white-space: pre-wrap;
    background: none;
    border: none;
    padding: 0;
  }

  .commit-files-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 2rem;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    margin-bottom: 2rem;
  }

  .commit-files-card h3 {
    margin-top: 0;
    margin-bottom: 1.5rem;
    font-size: 0.95rem;
    color: #8e8e93;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding-bottom: 0.75rem;
  }

  .files-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .commit-file-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 16px;
    background: rgba(255, 255, 255, 0.01);
    border: 1px solid rgba(255, 255, 255, 0.03);
    border-radius: 8px;
    cursor: pointer;
    transition: background-color 0.15s, border-color 0.15s, transform 0.1s;
  }

  .commit-file-item:hover {
    background: rgba(99, 102, 241, 0.05);
    border-color: rgba(99, 102, 241, 0.25);
    transform: translateY(-1px);
  }

  .commit-file-item:active {
    transform: translateY(0);
  }

  .file-info-group {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .file-status-badge {
    font-size: 11px;
    font-weight: 700;
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    font-family: monospace;
  }

  .file-path {
    font-size: 13px;
    font-family: 'Fira Code', monospace;
    color: #e3e3e6;
  }

  .file-action-indicator {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #8e8e93;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .commit-file-item:hover .file-action-indicator {
    opacity: 1;
  }

  .no-files-text {
    color: #8e8e93;
    font-size: 13px;
    font-style: italic;
  }
</style>
