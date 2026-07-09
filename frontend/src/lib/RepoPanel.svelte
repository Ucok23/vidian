<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';
  import OnboardingPanel from './OnboardingPanel.svelte';
  import InsightsPanel from './InsightsPanel.svelte';

  // Shared, deduplicated repo stats — previously computed and styled twice
  // (Onboarding's "Activity" card + Insights' stats strip). Now fetched once
  // here and shown above both segments.
  let stats = $state(null);

  onMount(async () => {
    try {
      const res = await fetch(store.apiUrl('/api/git/stats'));
      stats = await res.json();
    } catch {
      stats = null;
    }
  });

  function select(segment) {
    store.repoSegment = segment;
  }
</script>

<div class="repo">
  <div class="repo-header">
    <div class="title-row">
      <Icon name="compass" size={20} color="#38bdf8" />
      <h1>Repo</h1>
    </div>
    <div class="segments">
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="segment" class:active={store.repoSegment === 'overview'} onclick={() => select('overview')}>
        <Icon name="compass" size={14} />
        <span>Overview</span>
      </div>
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="segment" class:active={store.repoSegment === 'insights'} onclick={() => select('insights')}>
        <Icon name="chart" size={14} />
        <span>Insights</span>
      </div>
    </div>
  </div>

  {#if stats}
    <div class="stats-strip">
      <div class="stat">
        <span class="stat-value">{stats.totalFiles.toLocaleString()}</span>
        <span class="stat-label">files</span>
      </div>
      <div class="stat-divider"></div>
      <div class="stat">
        <span class="stat-value">{stats.totalCommits.toLocaleString()}</span>
        <span class="stat-label">commits</span>
      </div>
      {#if stats.totalBranches !== undefined}
        <div class="stat-divider"></div>
        <div class="stat">
          <span class="stat-value">{stats.totalBranches}</span>
          <span class="stat-label">branches</span>
        </div>
      {/if}
      <div class="stat-divider"></div>
      <div class="stat">
        <span class="stat-value">{stats.totalContributors}</span>
        <span class="stat-label">contributors</span>
      </div>
      {#if stats.totalTags !== undefined}
        <div class="stat-divider"></div>
        <div class="stat">
          <span class="stat-value">{stats.totalTags}</span>
          <span class="stat-label">tags</span>
        </div>
      {/if}
      {#if stats.firstCommitDate}
        <div class="stat-divider"></div>
        <div class="stat">
          <span class="stat-value">{stats.firstCommitDate}</span>
          <span class="stat-label">first commit</span>
        </div>
      {/if}
    </div>
  {/if}

  <div class="repo-body">
    {#if store.repoSegment === 'insights'}
      <InsightsPanel />
    {:else}
      <OnboardingPanel />
    {/if}
  </div>
</div>

<style>
  .repo {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #121214;
    color: #e3e3e6;
    font-family: system-ui, sans-serif;
    box-sizing: border-box;
  }

  .repo-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 20px 32px 16px;
    flex-wrap: wrap;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  h1 {
    font-size: 20px;
    font-weight: 700;
    color: #f0f0f4;
    margin: 0;
  }

  .segments {
    display: flex;
    gap: 4px;
    background: #1b1b20;
    border: 1px solid #2d2d34;
    border-radius: 8px;
    padding: 3px;
  }

  .segment {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 14px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 600;
    color: #8e8e93;
    cursor: pointer;
    user-select: none;
    transition: background 0.15s, color 0.15s;
  }

  .segment:hover {
    color: #e3e3e6;
  }

  .segment.active {
    background: #6366f1;
    color: #ffffff;
  }

  .stats-strip {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    margin: 0 32px 4px;
    padding: 14px 8px;
    background: #1b1b20;
    border: 1px solid #2d2d34;
    border-radius: 10px;
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

  .repo-body {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
</style>
