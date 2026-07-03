<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let profile = $state(null);
  let isLoading = $state(true);
  let error = $state(null);

  onMount(async () => {
    try {
      const res = await fetch(store.apiUrl('/api/git/profile'));
      if (!res.ok) throw new Error(`request failed: ${res.status}`);
      profile = await res.json();
    } catch (e) {
      error = e.message;
    } finally {
      isLoading = false;
    }
  });

  function openPath(path) {
    store.openFile(path);
  }
</script>

<div class="onboarding">
  <div class="header">
    <Icon name="compass" size={20} color="#38bdf8" />
    <h1>Onboarding</h1>
    <span class="subtitle">Deterministic repo profile — no AI required</span>
  </div>

  {#if isLoading}
    <div class="loading">Analyzing repository…</div>
  {:else if error}
    <div class="empty">Failed to load repo profile: {error}</div>
  {:else if profile}
    <div class="two-col">
      <!-- STACK -->
      <section class="card">
        <div class="card-title">Stack</div>
        {#if profile.stack.length === 0}
          <div class="empty">No recognized manifests found</div>
        {:else}
          <div class="chip-list">
            {#each profile.stack as s}
              <div class="chip" title={s.detail}>
                <span class="chip-name">{s.name}</span>
                <span class="chip-detail">{s.detail}</span>
              </div>
            {/each}
          </div>
        {/if}
      </section>

      <!-- ENTRY POINTS -->
      <section class="card">
        <div class="card-title">Entry Points <span class="card-sub">where to start reading</span></div>
        {#if profile.entryPoints.length === 0}
          <div class="empty">No conventional entry points detected</div>
        {:else}
          <div class="file-list">
            {#each profile.entryPoints as e}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="file-row" onclick={() => openPath(e.path)} title={e.path}>
                <span class="file-path">{e.path}</span>
                <span class="file-reason">{e.label}</span>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    </div>

    <!-- KEY FILES -->
    <section class="card">
      <div class="card-title">Key Files <span class="card-sub">README, manifests, most frequently changed</span></div>
      {#if profile.keyFiles.length === 0}
        <div class="empty">No key files detected</div>
      {:else}
        <div class="file-list">
          {#each profile.keyFiles as k}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div class="file-row" onclick={() => openPath(k.path)} title={k.path}>
              <span class="file-path">{k.path}</span>
              <span class="file-reason">{k.reason}</span>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <!-- ACTIVITY -->
    {#if profile.stats}
      <section class="card">
        <div class="card-title">Activity</div>
        <div class="stats-strip">
          <div class="stat">
            <span class="stat-value">{profile.stats.totalFiles.toLocaleString()}</span>
            <span class="stat-label">files</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat">
            <span class="stat-value">{profile.stats.totalCommits.toLocaleString()}</span>
            <span class="stat-label">commits</span>
          </div>
          <div class="stat-divider"></div>
          <div class="stat">
            <span class="stat-value">{profile.stats.totalContributors}</span>
            <span class="stat-label">contributors</span>
          </div>
          {#if profile.stats.firstCommitDate}
            <div class="stat-divider"></div>
            <div class="stat">
              <span class="stat-value">{profile.stats.firstCommitDate}</span>
              <span class="stat-label">first commit</span>
            </div>
          {/if}
        </div>
      </section>
    {/if}
  {/if}
</div>

<style>
  .onboarding {
    height: 100%;
    overflow-y: auto;
    padding: 28px 32px;
    background: #121214;
    color: #e3e3e6;
    font-family: system-ui, sans-serif;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 28px;
  }

  h1 {
    font-size: 20px;
    font-weight: 700;
    color: #f0f0f4;
    margin: 0;
  }

  .subtitle {
    font-size: 13px;
    color: #8e8e93;
    margin-left: 4px;
  }

  .loading {
    color: #8e8e93;
    font-style: italic;
    padding: 40px;
    text-align: center;
  }

  .two-col {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
  }

  .card {
    background: #1b1b20;
    border: 1px solid #2d2d34;
    border-radius: 10px;
    padding: 20px 24px;
    margin-bottom: 20px;
  }

  .card-title {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.6px;
    color: #8e8e93;
    text-transform: uppercase;
    margin-bottom: 16px;
  }

  .card-sub {
    font-weight: 400;
    text-transform: none;
    font-size: 11px;
    color: #5d5d66;
    letter-spacing: 0;
  }

  .empty {
    color: #5d5d66;
    font-style: italic;
    font-size: 12px;
    padding: 12px 0;
  }

  .chip-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .chip {
    display: flex;
    align-items: baseline;
    gap: 8px;
    padding: 8px 10px;
    background: #232329;
    border-radius: 6px;
  }

  .chip-name {
    font-size: 12px;
    font-weight: 700;
    color: #a5b4fc;
  }

  .chip-detail {
    font-size: 11px;
    color: #8e8e93;
    font-family: 'Fira Code', monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .file-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 8px 10px;
    border-radius: 6px;
    cursor: pointer;
  }

  .file-row:hover {
    background: #232329;
  }

  .file-path {
    font-size: 12px;
    font-family: 'Fira Code', monospace;
    color: #c0c0c8;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-row:hover .file-path {
    color: #a5b4fc;
  }

  .file-reason {
    font-size: 10px;
    color: #5d5d66;
    text-transform: uppercase;
    letter-spacing: 0.4px;
    flex-shrink: 0;
  }

  .stats-strip {
    display: flex;
    align-items: center;
    gap: 0;
    flex-wrap: wrap;
  }

  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0 20px;
    flex: 1;
    min-width: 80px;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 700;
    color: #a5b4fc;
    font-variant-numeric: tabular-nums;
    line-height: 1.2;
  }

  .stat-label {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.5px;
    color: #5d5d66;
    text-transform: uppercase;
    margin-top: 3px;
  }

  .stat-divider {
    width: 1px;
    height: 32px;
    background: #2d2d34;
    flex-shrink: 0;
  }
</style>
