<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let { commit } = $props();

  let copied = $state(false);
  let parentHash = $state(null);
  let loadingParent = $state(false);

  async function loadParentCommit() {
    if (loadingParent || parentHash) return;
    loadingParent = true;
    try {
      const res = await fetch(store.apiUrl(`/api/git/log?path=`));
      // Use git show to get parent
      const pRes = await fetch(store.apiUrl(`/api/git/commit?hash=${commit.hash}%5E`)); // ^
      if (pRes.ok) {
        const p = await pRes.json();
        parentHash = p.hash;
      }
    } catch (_) {}
    loadingParent = false;
  }

  function openParentCommit() {
    if (parentHash) store.openCommit(parentHash);
    else {
      // Try to open commit^
      store.openCommit(commit.hash + '^');
    }
  }

  function copyHash() {
    navigator.clipboard.writeText(commit.hash);
    copied = true;
    setTimeout(() => {
      copied = false;
    }, 2000);
  }

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
    
    <div class="commit-actions-row">
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="commit-action-btn" onclick={openParentCommit} title="Open parent commit">
        ← Parent Commit
      </div>
    </div>

    <div class="commit-meta-list">
      <div class="meta-item">
        <span class="meta-label">Commit:</span>
        <span class="meta-value hash-wrapper">
          <span class="hash">{commit.hash}</span>
          <button class="copy-btn" onclick={copyHash} title="Copy full hash">
            {#if copied}
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#10b981" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
              <span style="color: #10b981;">Copied!</span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
              <span>Copy</span>
            {/if}
          </button>
        </span>
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
          <div class="commit-file-item" title={file.path}>
            <div class="file-info-group">
              <span class="file-status-badge" style="background-color: {getStatusColor(file.status)}15; color: {getStatusColor(file.status)}">
                {file.status}
              </span>
              <span class="file-path">{file.path}</span>
            </div>
            <div class="file-actions">
              {#if file.status !== 'D'}
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <span class="file-action-btn" onclick={() => store.openFile(file.path)} title="Open file">Open</span>
              {/if}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <span class="file-action-btn primary" onclick={() => store.openDiff(file.path, `${commit.hash}^`, commit.hash, `${file.path.split('/').pop()} (${commit.hash.slice(0, 7)})`)} title="View diff">Diff</span>
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

  .commit-actions-row {
    display: flex;
    gap: 8px;
    margin-bottom: 1.5rem;
  }

  .commit-action-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 5px;
    color: #a1a1aa;
    padding: 4px 10px;
    font-size: 11px;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
    user-select: none;
  }
  .commit-action-btn:hover {
    background: rgba(99, 102, 241, 0.12);
    border-color: rgba(99, 102, 241, 0.25);
    color: #ffffff;
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

  .commit-meta-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .meta-item {
    display: flex;
    align-items: flex-start;
    gap: 1.5rem;
    font-size: 13px;
    line-height: 1.5;
  }

  .meta-label {
    width: 90px;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #8e8e93;
    flex-shrink: 0;
    padding-top: 2px;
  }

  .meta-value {
    color: #e3e3e6;
    word-break: break-all;
  }

  .hash-wrapper {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .hash {
    font-family: 'Fira Code', monospace;
    color: #a5b4fc;
    word-break: break-all;
  }

  .copy-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 4px;
    color: #a1a1aa;
    padding: 2px 6px;
    font-size: 11px;
    cursor: pointer;
    transition: background-color 0.15s, color 0.15s, border-color 0.15s;
    user-select: none;
  }

  .copy-btn:hover {
    background: rgba(99, 102, 241, 0.12);
    border-color: rgba(99, 102, 241, 0.25);
    color: #ffffff;
  }

  .copy-btn svg {
    flex-shrink: 0;
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

  .file-actions {
    display: flex;
    gap: 4px;
    opacity: 0;
    transition: opacity 0.15s;
  }
  .commit-file-item:hover .file-actions { opacity: 1; }

  .file-action-btn {
    font-size: 10px;
    padding: 2px 7px;
    border-radius: 3px;
    background: rgba(255,255,255,0.06);
    color: #8e8e93;
    cursor: pointer;
    user-select: none;
    transition: background 0.12s, color 0.12s;
  }
  .file-action-btn:hover { background: rgba(255,255,255,0.12); color: #e3e3e6; }
  .file-action-btn.primary { background: rgba(99,102,241,0.15); color: #818cf8; }
  .file-action-btn.primary:hover { background: rgba(99,102,241,0.28); color: #a5b4fc; }

  .no-files-text {
    color: #8e8e93;
    font-size: 13px;
    font-style: italic;
  }
</style>
