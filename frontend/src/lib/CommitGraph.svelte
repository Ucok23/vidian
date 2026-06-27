<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let commits = $state([]);
  let isLoading = $state(true);
  let error = $state('');

  // Layout constants
  const ROW_H = 28;
  const COL_W = 16;
  const NODE_R = 4.5;
  const LEFT_PAD = 14;

  const LANE_COLORS = [
    '#6366f1', '#10b981', '#f59e0b', '#ef4444',
    '#ec4899', '#06b6d4', '#8b5cf6', '#84cc16',
    '#f97316', '#14b8a6'
  ];
  const laneColor = (col) => LANE_COLORS[((col % LANE_COLORS.length) + LANE_COLORS.length) % LANE_COLORS.length];

  // Derived layout: column per commit, edges, and overall graph width.
  const layout = $derived.by(() => {
    const rowOf = new Map();
    commits.forEach((c, i) => rowOf.set(c.hash, i));

    const colOf = new Map();
    let lanes = []; // lanes[i] = hash that lane i is currently waiting for, or null
    let maxCol = 0;

    const takeFreeLane = () => {
      const free = lanes.indexOf(null);
      if (free !== -1) return free;
      lanes.push(null);
      return lanes.length - 1;
    };

    for (let i = 0; i < commits.length; i++) {
      const c = commits[i];

      // Column for this commit = first lane already waiting for it, else a fresh lane.
      let col = lanes.indexOf(c.hash);
      if (col === -1) col = takeFreeLane();
      colOf.set(c.hash, col);
      maxCol = Math.max(maxCol, col);

      // Other lanes waiting for the same commit are merges that converge here → free them.
      for (let j = 0; j < lanes.length; j++) {
        if (j !== col && lanes[j] === c.hash) lanes[j] = null;
      }

      const parents = c.parents.filter((p) => rowOf.has(p));
      if (parents.length === 0) {
        lanes[col] = null; // root commit ends this lane
      } else {
        lanes[col] = parents[0]; // first parent continues in this column
        for (let k = 1; k < parents.length; k++) {
          const p = parents[k];
          if (lanes.indexOf(p) === -1) {
            const pcol = takeFreeLane();
            lanes[pcol] = p;
            maxCol = Math.max(maxCol, pcol);
          }
        }
      }
    }

    // Build edges (child → each existing parent).
    const colX = (col) => LEFT_PAD + col * COL_W;
    const rowY = (row) => row * ROW_H + ROW_H / 2;
    const edges = [];
    for (const c of commits) {
      const cr = rowOf.get(c.hash);
      const cc = colOf.get(c.hash);
      const x1 = colX(cc);
      const y1 = rowY(cr);
      for (const p of c.parents) {
        if (!rowOf.has(p)) continue;
        const pr = rowOf.get(p);
        const pc = colOf.get(p);
        const x2 = colX(pc);
        const y2 = rowY(pr);
        let d;
        if (x1 === x2) {
          d = `M ${x1} ${y1} L ${x2} ${y2}`;
        } else {
          // Bend within the first row gap, then run straight down the parent's lane.
          const bendY = y1 + ROW_H;
          d = `M ${x1} ${y1} C ${x1} ${y1 + ROW_H * 0.45}, ${x2} ${y1 + ROW_H * 0.55}, ${x2} ${bendY} L ${x2} ${y2}`;
        }
        // Colour the edge by the lane it settles into (the parent's column).
        edges.push({ d, color: laneColor(pc) });
      }
    }

    return {
      colOf,
      edges,
      width: LEFT_PAD + maxCol * COL_W + COL_W,
      colX,
      rowY
    };
  });

  function parseRefs(refs) {
    if (!refs) return [];
    return refs.split(',').map((r) => r.trim()).filter(Boolean).map((r) => {
      if (r.startsWith('tag:')) return { label: r.slice(4).trim(), kind: 'tag' };
      if (r.includes('HEAD ->')) return { label: r.replace('HEAD ->', '').trim(), kind: 'head' };
      if (r === 'HEAD') return { label: 'HEAD', kind: 'head' };
      if (r.startsWith('origin/') || r.includes('/')) return { label: r, kind: 'remote' };
      return { label: r, kind: 'branch' };
    });
  }

  function shortDate(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    if (isNaN(d)) return '';
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
  }

  async function load() {
    isLoading = true;
    error = '';
    try {
      const res = await fetch('/api/git/graph');
      commits = (await res.json()) || [];
    } catch (e) {
      error = 'Failed to load commit graph';
      commits = [];
    } finally {
      isLoading = false;
    }
  }

  onMount(load);
</script>

<div class="graph-view">
  <div class="graph-header">
    <Icon name="branch" size={15} color="#6366f1" />
    <span class="graph-title">Commit Graph</span>
    <span class="graph-count">{commits.length} commits · all refs</span>
    <button class="refresh-btn" onclick={load} title="Refresh">
      <Icon name="refresh" size={13} />
    </button>
  </div>

  {#if isLoading}
    <div class="graph-msg">Loading commit graph…</div>
  {:else if error}
    <div class="graph-msg error">{error}</div>
  {:else if commits.length === 0}
    <div class="graph-msg">No commits to display.</div>
  {:else}
    <div class="graph-scroll">
      <div class="graph-inner" style="height: {commits.length * ROW_H}px;">
        <!-- Lane lines + nodes -->
        <svg
          class="graph-svg"
          width={layout.width}
          height={commits.length * ROW_H}
          style="width: {layout.width}px;"
        >
          {#each layout.edges as e}
            <path d={e.d} stroke={e.color} fill="none" stroke-width="1.6" />
          {/each}
          {#each commits as c, i}
            <circle
              cx={layout.colX(layout.colOf.get(c.hash))}
              cy={layout.rowY(i)}
              r={NODE_R}
              fill={laneColor(layout.colOf.get(c.hash))}
              stroke="#1e1e24"
              stroke-width="1.5"
            />
          {/each}
        </svg>

        <!-- Commit rows -->
        <div class="rows" style="padding-left: {layout.width + 6}px;">
          {#each commits as c (c.hash)}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div
              class="row"
              style="height: {ROW_H}px;"
              onclick={() => store.openCommit(c.hash)}
              title={c.subject}
            >
              {#each parseRefs(c.refs) as ref}
                <span class="ref-pill {ref.kind}">
                  {#if ref.kind === 'tag'}<Icon name="tag" size={10} />{/if}
                  {ref.label}
                </span>
              {/each}
              <span class="subject">{c.subject}</span>
              <span class="meta hash">{c.hash.slice(0, 7)}</span>
              <span class="meta author">{c.author}</span>
              <span class="meta date">{shortDate(c.date)}</span>
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .graph-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #1e1e24;
    color: #c9c9d0;
    overflow: hidden;
  }

  .graph-header {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 10px 16px;
    border-bottom: 1px solid #2d2d34;
    background: #16161a;
    flex-shrink: 0;
  }
  .graph-title { font-size: 13px; font-weight: 600; color: #e3e3e6; }
  .graph-count { font-size: 11px; color: #5d5d66; margin-left: 2px; }
  .refresh-btn {
    margin-left: auto;
    background: transparent; border: none; color: #8e8e93;
    cursor: pointer; padding: 4px; border-radius: 4px; display: flex;
  }
  .refresh-btn:hover { background: rgba(255,255,255,.06); color: #e3e3e6; }

  .graph-msg { padding: 24px 16px; font-size: 13px; color: #6b6b78; }
  .graph-msg.error { color: #f87171; }

  .graph-scroll { flex: 1; overflow: auto; }
  .graph-inner { position: relative; min-width: 100%; }

  .graph-svg {
    position: absolute;
    top: 0; left: 0;
    height: 100%;
    pointer-events: none;
    z-index: 1;
  }
  .graph-svg path { stroke-linecap: round; }

  .rows { position: relative; z-index: 2; }

  .row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 16px 0 0;
    cursor: pointer;
    font-size: 12.5px;
    white-space: nowrap;
    border-bottom: 1px solid transparent;
  }
  .row:hover { background: rgba(99,102,241,.08); }

  .subject {
    color: #d4d4dc;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-shrink: 1;
    min-width: 0;
  }

  .meta { font-size: 11px; flex-shrink: 0; }
  .meta.hash {
    font-family: 'Fira Code', monospace;
    color: #fbbf24;
    margin-left: auto;
  }
  .meta.author { color: #7d8590; max-width: 140px; overflow: hidden; text-overflow: ellipsis; }
  .meta.date { color: #5d5d66; min-width: 88px; text-align: right; }

  .ref-pill {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 10px;
    font-weight: 600;
    padding: 1px 7px;
    border-radius: 9px;
    flex-shrink: 0;
    border: 1px solid transparent;
  }
  .ref-pill.head   { background: rgba(16,185,129,.16); color: #34d399; border-color: rgba(16,185,129,.35); }
  .ref-pill.branch { background: rgba(99,102,241,.16); color: #a5b4fc; border-color: rgba(99,102,241,.32); }
  .ref-pill.remote { background: rgba(125,133,144,.14); color: #9da7b3; border-color: rgba(125,133,144,.3); }
  .ref-pill.tag    { background: rgba(245,158,11,.15); color: #fbbf24; border-color: rgba(245,158,11,.35); }
</style>
