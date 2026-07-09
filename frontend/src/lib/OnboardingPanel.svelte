<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';

  let profile = $state(null);
  let isLoading = $state(true);
  let error = $state(null);

  let hasKey = $state(false);
  let narrative = $state(null);
  let isLoadingNarrative = $state(false);
  let narrativeError = $state(null);

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

    try {
      const res = await fetch('/api/settings');
      const data = await res.json();
      hasKey = !!data.hasKey;
      if (hasKey) fetchNarrative(false);
    } catch (e) {
      // Settings check is best-effort; leave the "add a key" CTA showing.
    }
  });

  async function fetchNarrative(regenerate) {
    isLoadingNarrative = true;
    narrativeError = null;
    try {
      const path = regenerate ? '/api/git/narrative?regenerate=1' : '/api/git/narrative';
      const res = await fetch(store.apiUrl(path));
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      narrative = data.narrative;
    } catch (e) {
      narrativeError = e.message;
    } finally {
      isLoadingNarrative = false;
    }
  }

  function openPath(path) {
    store.openFile(path);
  }

  function openSettings() {
    store.settingsOpen = true;
  }
</script>

<div class="onboarding">
  {#if isLoading}
    <div class="loading">Analyzing repository…</div>
  {:else if error}
    <div class="empty">Failed to load repo profile: {error}</div>
  {:else if profile}
    <!-- AI NARRATIVE -->
    <section class="card narrative-card">
      <div class="card-title-row">
        <span class="card-title">AI Tour</span>
        {#if hasKey && narrative && !isLoadingNarrative}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <span class="regenerate-btn" onclick={() => fetchNarrative(true)}>Regenerate</span>
        {/if}
      </div>
      {#if !hasKey}
        <div class="narrative-cta">
          <p>Add an Anthropic API key to generate a written tour of this repo, grounded in the facts below.</p>
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <span class="settings-link" onclick={openSettings}>Open Settings</span>
        </div>
      {:else if isLoadingNarrative}
        <div class="loading">Writing tour…</div>
      {:else if narrativeError}
        <div class="empty">Failed to generate tour: {narrativeError}</div>
      {:else if narrative}
        <p class="narrative-text">{narrative}</p>
      {/if}
    </section>

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

  .card-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  .card-title-row .card-title {
    margin-bottom: 0;
  }

  .narrative-card {
    border-color: #38bdf833;
    background: linear-gradient(180deg, #1b1b20 0%, #17202b 100%);
  }

  .narrative-text {
    font-size: 13px;
    line-height: 1.7;
    color: #d4d4d8;
    white-space: pre-wrap;
    margin: 0;
  }

  .narrative-cta p {
    font-size: 12px;
    color: #8e8e93;
    margin: 0 0 10px;
  }

  .settings-link,
  .regenerate-btn {
    font-size: 12px;
    color: #38bdf8;
    cursor: pointer;
    font-weight: 600;
  }

  .settings-link:hover,
  .regenerate-btn:hover {
    text-decoration: underline;
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

</style>
