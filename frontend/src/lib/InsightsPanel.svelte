<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let activity = $state({});       // { "2025-06-28": 3, ... }
  let hotFiles = $state([]);        // [{ path, commits }, ...]
  let contributors = $state([]);    // [{ name, email, commits }, ...]
  let isLoading = $state(true);

  // Build a 52-week heatmap grid: array of weeks, each week is 7 days (Sun–Sat)
  const heatmap = $derived.by(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    // Snap start back to the Sunday that is ~52 weeks ago
    const start = new Date(today);
    start.setDate(today.getDate() - 364);
    start.setDate(start.getDate() - start.getDay()); // back to Sunday

    const weeks = [];
    const cursor = new Date(start);

    while (cursor <= today) {
      const week = [];
      for (let dow = 0; dow < 7; dow++) {
        const dateStr = cursor.toISOString().slice(0, 10);
        week.push({
          date: dateStr,
          count: activity[dateStr] || 0,
          future: cursor > today
        });
        cursor.setDate(cursor.getDate() + 1);
      }
      weeks.push(week);
    }
    return weeks;
  });

  // Month labels: find the week index where each month starts
  const monthLabels = $derived.by(() => {
    const labels = [];
    let lastMonth = -1;
    heatmap.forEach((week, wi) => {
      const d = new Date(week[0].date);
      const m = d.getMonth();
      if (m !== lastMonth) {
        labels.push({ wi, label: d.toLocaleString('default', { month: 'short' }) });
        lastMonth = m;
      }
    });
    return labels;
  });

  const totalCommits = $derived(Object.values(activity).reduce((s, v) => s + v, 0));
  const maxCount = $derived(Math.max(1, ...Object.values(activity)));
  const maxFileCommits = $derived(hotFiles.length > 0 ? hotFiles[0].commits : 1);
  const totalAuthorCommits = $derived(contributors.reduce((s, c) => s + c.commits, 0) || 1);

  function cellColor(count) {
    if (count === 0) return '#1c1c26';
    if (count === 1) return '#1e1b4b';
    if (count <= 3) return '#3730a3';
    if (count <= 6) return '#4f46e5';
    if (count <= 10) return '#6366f1';
    return '#a5b4fc';
  }

  const DAY_LABELS = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];

  onMount(async () => {
    const [actRes, filesRes, contribRes] = await Promise.all([
      fetch('/api/git/activity'),
      fetch('/api/git/hot-files'),
      fetch('/api/git/contributors')
    ]);
    activity = await actRes.json();
    hotFiles = await filesRes.json();
    contributors = await contribRes.json();
    isLoading = false;
  });
</script>

<div class="insights">
  <div class="header">
    <Icon name="chart" size={20} color="#6366f1" />
    <h1>Repository Insights</h1>
    {#if !isLoading}
      <span class="subtitle">{totalCommits} commits in the past year</span>
    {/if}
  </div>

  {#if isLoading}
    <div class="loading">Loading insights…</div>
  {:else}
    <!-- ACTIVITY HEATMAP -->
    <section class="card">
      <div class="card-title">Commit Activity</div>
      <div class="heatmap-wrap">
        <!-- Month labels row -->
        <div class="month-row">
          <div class="day-col-spacer"></div>
          <div class="month-labels" style="grid-template-columns: repeat({heatmap.length}, 13px)">
            {#each monthLabels as { wi, label }}
              <span class="month-label" style="grid-column: {wi + 1}">{label}</span>
            {/each}
          </div>
        </div>
        <!-- Grid -->
        <div class="heatmap-grid-row">
          <!-- Day-of-week labels -->
          <div class="day-labels">
            {#each DAY_LABELS as d, i}
              <span class="day-label" class:dim={i % 2 === 0}>{d}</span>
            {/each}
          </div>
          <!-- Cells -->
          <div class="heatmap-grid" style="grid-template-columns: repeat({heatmap.length}, 11px)">
            {#each heatmap as week, wi}
              {#each week as cell, di}
                <div
                  class="cell"
                  class:future={cell.future}
                  style="background:{cell.future ? '#1c1c26' : cellColor(cell.count)}; grid-column:{wi+1}; grid-row:{di+1};"
                  title="{cell.date}: {cell.count} commit{cell.count !== 1 ? 's' : ''}"
                ></div>
              {/each}
            {/each}
          </div>
        </div>
        <!-- Legend -->
        <div class="legend">
          <span>Less</span>
          {#each [0, 1, 3, 6, 11] as v}
            <div class="cell legend-cell" style="background:{cellColor(v)}"></div>
          {/each}
          <span>More</span>
        </div>
      </div>
    </section>

    <div class="two-col">
      <!-- HOT FILES -->
      <section class="card">
        <div class="card-title">Most Changed Files <span class="card-sub">past year</span></div>
        {#if hotFiles.length === 0}
          <div class="empty">No file changes found</div>
        {:else}
          <div class="bar-list">
            {#each hotFiles as f}
              {@const pct = Math.round((f.commits / maxFileCommits) * 100)}
              <!-- svelte-ignore a11y_click_events_have_key_events -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div class="bar-row" onclick={() => store.openFile(f.path)} title={f.path}>
                <span class="bar-label">{f.path.split('/').pop()}</span>
                <div class="bar-track">
                  <div class="bar-fill" style="width:{pct}%"></div>
                </div>
                <span class="bar-value">{f.commits}</span>
              </div>
            {/each}
          </div>
        {/if}
      </section>

      <!-- AUTHORS -->
      <section class="card">
        <div class="card-title">Authors <span class="card-sub">all time</span></div>
        {#if contributors.length === 0}
          <div class="empty">No contributors found</div>
        {:else}
          <div class="bar-list">
            {#each contributors as c}
              {@const pct = Math.round((c.commits / totalAuthorCommits) * 100)}
              <div class="author-row">
                <div class="avatar">{c.name.charAt(0).toUpperCase()}</div>
                <div class="author-info">
                  <span class="author-name">{c.name}</span>
                  <div class="bar-track small">
                    <div class="bar-fill author-fill" style="width:{pct}%"></div>
                  </div>
                </div>
                <div class="author-stats">
                  <span class="bar-value">{c.commits}</span>
                  <span class="author-pct">{pct}%</span>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    </div>
  {/if}
</div>

<style>
  .insights {
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

  .two-col {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
  }

  /* HEATMAP */
  .heatmap-wrap {
    display: flex;
    flex-direction: column;
    gap: 4px;
    overflow-x: auto;
  }

  .month-row {
    display: flex;
    align-items: flex-end;
    gap: 0;
    padding-bottom: 2px;
  }

  .day-col-spacer {
    width: 20px;
    flex-shrink: 0;
  }

  .month-labels {
    display: grid;
    gap: 2px;
  }

  .month-label {
    font-size: 10px;
    color: #8e8e93;
    white-space: nowrap;
  }

  .heatmap-grid-row {
    display: flex;
    gap: 4px;
  }

  .day-labels {
    display: grid;
    grid-template-rows: repeat(7, 13px);
    gap: 2px;
    width: 16px;
    flex-shrink: 0;
  }

  .day-label {
    font-size: 9px;
    color: #5d5d66;
    line-height: 13px;
    text-align: center;
  }

  .day-label.dim {
    opacity: 0;
  }

  .heatmap-grid {
    display: grid;
    grid-template-rows: repeat(7, 11px);
    gap: 2px;
  }

  .cell {
    width: 11px;
    height: 11px;
    border-radius: 2px;
    transition: opacity 0.1s;
    cursor: default;
  }

  .cell:hover {
    opacity: 0.8;
    outline: 1px solid rgba(255,255,255,0.2);
  }

  .cell.future {
    opacity: 0.2;
  }

  .legend {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 8px;
    font-size: 10px;
    color: #5d5d66;
  }

  .legend-cell {
    cursor: default;
  }

  /* BARS */
  .bar-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .bar-row {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    padding: 4px 0;
  }

  .bar-row:hover .bar-label {
    color: #a5b4fc;
  }

  .bar-label {
    font-size: 12px;
    color: #c0c0c8;
    width: 130px;
    flex-shrink: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-family: 'Fira Code', monospace;
    font-size: 11px;
  }

  .bar-track {
    flex: 1;
    height: 6px;
    background: #2a2a34;
    border-radius: 3px;
    overflow: hidden;
  }

  .bar-track.small {
    height: 4px;
    margin-top: 4px;
  }

  .bar-fill {
    height: 100%;
    background: #6366f1;
    border-radius: 3px;
    transition: width 0.4s ease;
  }

  .author-fill {
    background: #818cf8;
  }

  .bar-value {
    font-size: 11px;
    color: #818cf8;
    font-weight: 700;
    width: 28px;
    text-align: right;
    flex-shrink: 0;
  }

  /* AUTHORS */
  .author-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0;
  }

  .avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: rgba(99, 102, 241, 0.2);
    color: #818cf8;
    font-size: 13px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .author-info {
    flex: 1;
    min-width: 0;
  }

  .author-name {
    font-size: 12px;
    color: #e3e3e6;
    display: block;
  }

  .author-stats {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    flex-shrink: 0;
  }

  .author-pct {
    font-size: 10px;
    color: #5d5d66;
  }

  .empty {
    color: #5d5d66;
    font-style: italic;
    font-size: 12px;
    padding: 12px 0;
  }
</style>
