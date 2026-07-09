<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';

  let activity = $state({});
  let hotFiles = $state([]);
  let contributors = $state([]);
  let stats = $state(null);
  let langStats = $state([]);
  let dowStats = $state([]);
  let hourStats = $state([]);
  let wordStats = $state([]);
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
  const maxLangCount = $derived(langStats.length > 0 ? langStats[0].count : 1);
  const totalLangFiles = $derived(langStats.reduce((s, l) => s + l.count, 0) || 1);
  const maxDow = $derived(Math.max(1, ...dowStats.map(d => d.count)));
  const maxHour = $derived(Math.max(1, ...hourStats.map(h => h.count)));
  const maxWord = $derived(wordStats.length > 0 ? wordStats[0].count : 1);

  function cellColor(count) {
    if (count === 0) return '#1c1c26';
    if (count === 1) return '#1e1b4b';
    if (count <= 3) return '#3730a3';
    if (count <= 6) return '#4f46e5';
    if (count <= 10) return '#6366f1';
    return '#a5b4fc';
  }

  const DAY_LABELS = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];

  const langColors = {
    svelte: '#ff3e00', go: '#00add8', js: '#f7df1e', ts: '#3178c6',
    jsx: '#61dafb', tsx: '#61dafb', css: '#264de4', html: '#e34c26',
    md: '#0891b2', json: '#cbcb41', sh: '#89e051', yaml: '#cb171e',
    yml: '#cb171e', py: '#3572a5', rs: '#dea584', toml: '#9c4221',
    sql: '#e38c00', vue: '#42b883', rb: '#cc342d', java: '#b07219',
  };
  function langColor(ext) {
    return langColors[ext] || '#6b7280';
  }

  onMount(async () => {
    const [actRes, filesRes, contribRes, statsRes, langRes, dowRes, hourRes, wordRes] = await Promise.all([
      fetch(store.apiUrl('/api/git/activity')),
      fetch(store.apiUrl('/api/git/hot-files')),
      fetch(store.apiUrl('/api/git/contributors')),
      fetch(store.apiUrl('/api/git/stats')),
      fetch(store.apiUrl('/api/git/lang-stats')),
      fetch(store.apiUrl('/api/git/dow-stats')),
      fetch(store.apiUrl('/api/git/hour-stats')),
      fetch(store.apiUrl('/api/git/word-stats')),
    ]);
    activity = await actRes.json();
    hotFiles = await filesRes.json();
    contributors = await contribRes.json();
    stats = await statsRes.json();
    langStats = await langRes.json();
    dowStats = await dowRes.json();
    hourStats = await hourRes.json();
    wordStats = await wordRes.json();
    isLoading = false;
  });
</script>

<div class="insights">
  {#if isLoading}
    <div class="loading">Loading insights…</div>
  {:else}
    <!-- ACTIVITY HEATMAP -->
    <section class="card">
      <div class="card-title-row">
        <span class="card-title">Commit Activity</span>
        {#if stats?.longestStreak}
          <div class="streak-badges">
            {#if stats.currentStreak > 0}
              <span class="streak-badge current" title="Current streak">🔥 {stats.currentStreak}d streak</span>
            {/if}
            <span class="streak-badge best" title="Longest streak">🏆 {stats.longestStreak}d best</span>
          </div>
        {/if}
      </div>
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
      <!-- LANGUAGE BREAKDOWN -->
      <section class="card">
        <div class="card-title">Languages <span class="card-sub">{langStats.length} types</span></div>
        {#if langStats.length === 0}
          <div class="empty">No files found</div>
        {:else}
          <div class="lang-list">
            {#each langStats as l}
              {@const pct = Math.round((l.count / totalLangFiles) * 100)}
              <div class="lang-row">
                <span class="lang-dot" style="background:{langColor(l.ext)}"></span>
                <span class="lang-name">.{l.ext}</span>
                <div class="bar-track">
                  <div class="bar-fill" style="width:{Math.round((l.count/maxLangCount)*100)}%; background:{langColor(l.ext)}"></div>
                </div>
                <span class="lang-pct">{pct}%</span>
                <span class="bar-value">{l.count}</span>
              </div>
            {/each}
          </div>
        {/if}
      </section>

      <!-- DAY OF WEEK -->
      <section class="card">
        <div class="card-title">By Day of Week</div>
        {#if dowStats.length === 0}
          <div class="empty">No data</div>
        {:else}
          <div class="dow-chart">
            {#each dowStats as d}
              {@const pct = Math.round((d.count / maxDow) * 100)}
              <div class="dow-col">
                <span class="dow-count">{d.count}</span>
                <div class="dow-bar-track">
                  <div class="dow-bar-fill" style="height:{pct}%"></div>
                </div>
                <span class="dow-label">{d.day}</span>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    </div>

    <!-- COMMIT HOUR DISTRIBUTION -->
    <section class="card">
      <div class="card-title">Commit Hours <span class="card-sub">all time</span></div>
      {#if hourStats.length === 0}
        <div class="empty">No data</div>
      {:else}
        <div class="hour-chart">
          {#each hourStats as h}
            {@const pct = Math.round((h.count / maxHour) * 100)}
            <div class="hour-col" title="{h.hour}:00 — {h.count} commit{h.count !== 1 ? 's' : ''}">
              <div class="hour-bar-track">
                <div class="hour-bar-fill" style="height:{pct}%"></div>
              </div>
              {#if h.hour % 6 === 0}
                <span class="hour-label">{h.hour}h</span>
              {:else}
                <span class="hour-label"></span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <!-- COMMIT WORDS -->
    {#if wordStats.length > 0}
    <section class="card">
      <div class="card-title">Commit Message Words <span class="card-sub">top {wordStats.length}</span></div>
      <div class="word-cloud">
        {#each wordStats as w, i}
          {@const scale = 0.7 + (w.count / maxWord) * 0.9}
          {@const opacity = 0.45 + (w.count / maxWord) * 0.55}
          <span class="word-tag" style="font-size:{(scale * 14).toFixed(1)}px; opacity:{opacity}" title="{w.count} occurrences">{w.word}</span>
        {/each}
      </div>
    </section>
    {/if}

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

  /* HEATMAP TITLE ROW */
  .card-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }

  .card-title-row .card-title {
    margin-bottom: 0;
  }

  .streak-badges {
    display: flex;
    gap: 8px;
  }

  .streak-badge {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 20px;
    white-space: nowrap;
  }

  .streak-badge.current {
    background: rgba(251, 146, 60, 0.15);
    color: #fb923c;
    border: 1px solid rgba(251, 146, 60, 0.25);
  }

  .streak-badge.best {
    background: rgba(250, 204, 21, 0.1);
    color: #facc15;
    border: 1px solid rgba(250, 204, 21, 0.2);
  }

  /* LANGUAGE BREAKDOWN */
  .lang-list {
    display: flex;
    flex-direction: column;
    gap: 7px;
  }

  .lang-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .lang-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .lang-name {
    font-size: 11px;
    font-family: 'Fira Code', monospace;
    color: #c0c0c8;
    width: 52px;
    flex-shrink: 0;
  }

  .lang-pct {
    font-size: 10px;
    color: #5d5d66;
    width: 28px;
    text-align: right;
    flex-shrink: 0;
  }

  /* DAY OF WEEK */
  .dow-chart {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 6px;
    height: 120px;
    padding-top: 20px;
  }

  .dow-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    flex: 1;
    height: 100%;
    gap: 4px;
  }

  .dow-count {
    font-size: 10px;
    color: #8e8e93;
    min-height: 14px;
  }

  .dow-bar-track {
    flex: 1;
    width: 100%;
    background: #2a2a34;
    border-radius: 3px 3px 0 0;
    display: flex;
    align-items: flex-end;
    overflow: hidden;
  }

  .dow-bar-fill {
    width: 100%;
    background: #6366f1;
    border-radius: 3px 3px 0 0;
    transition: height 0.4s ease;
    min-height: 2px;
  }

  .dow-label {
    font-size: 10px;
    color: #5d5d66;
    font-weight: 600;
  }

  /* HOUR DISTRIBUTION */
  .hour-chart {
    display: flex;
    align-items: flex-end;
    gap: 3px;
    height: 80px;
    padding-top: 4px;
  }

  .hour-col {
    display: flex;
    flex-direction: column;
    align-items: center;
    flex: 1;
    height: 100%;
    gap: 3px;
    cursor: default;
  }

  .hour-bar-track {
    flex: 1;
    width: 100%;
    background: #2a2a34;
    border-radius: 2px 2px 0 0;
    display: flex;
    align-items: flex-end;
    overflow: hidden;
  }

  .hour-bar-fill {
    width: 100%;
    background: linear-gradient(to top, #4f46e5, #818cf8);
    border-radius: 2px 2px 0 0;
    transition: height 0.4s ease;
    min-height: 1px;
  }

  .hour-col:hover .hour-bar-fill {
    background: linear-gradient(to top, #6366f1, #a5b4fc);
  }

  .hour-label {
    font-size: 9px;
    color: #5d5d66;
    min-height: 12px;
    line-height: 12px;
  }

  /* COMMIT WORD CLOUD */
  .word-cloud {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 10px;
    align-items: baseline;
    padding: 4px 0;
  }

  .word-tag {
    color: #818cf8;
    font-weight: 500;
    cursor: default;
    transition: color 0.15s;
    line-height: 1.4;
  }

  .word-tag:hover {
    color: #a5b4fc;
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
