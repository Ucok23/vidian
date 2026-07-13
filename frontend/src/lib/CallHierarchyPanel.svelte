<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';
  import CallNode from './CallNode.svelte';

  const ch = $derived(store.callHierarchy);
</script>

<div class="ch-panel">
  <div class="ch-header">
    <span class="ch-title">Call Hierarchy</span>
    {#if ch && !ch.loading}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <button
        class="ch-dir"
        onclick={() => store.toggleCallDirection()}
        title="Toggle callers / callees"
      >
        {ch.direction === 'incoming' ? 'Callers' : 'Callees'}
      </button>
    {/if}
    <button class="ch-close" title="Clear" onclick={() => store.clearCallHierarchy()}>
      <Icon name="close" size={13} />
    </button>
  </div>

  {#if ch}
    <div class="ch-symbol">
      <Icon name="network" size={13} color="#818cf8" />
      <code>{ch.symbol}</code>
    </div>
  {/if}

  <div class="ch-body">
    {#if !ch}
      <div class="ch-empty">
        Right-click a function and choose <em>Show Call Hierarchy</em>, or press
        <kbd>Shift</kbd>+<kbd>Alt</kbd>+<kbd>H</kbd>.
      </div>
    {:else if ch.loading}
      <div class="ch-empty">Resolving…</div>
    {:else if ch.error}
      <div class="ch-empty">{ch.error}</div>
    {:else if ch.roots.length === 0}
      <div class="ch-empty">No call hierarchy for <code>{ch.symbol}</code>.</div>
    {:else}
      {#each ch.roots as node (node.path + ':' + node.line + ':' + node.name)}
        <CallNode {node} depth={0} />
      {/each}
    {/if}
  </div>
</div>

<style>
  .ch-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    color: #c4c4cc;
    font-size: 13px;
  }

  .ch-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px 8px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .ch-title {
    font-size: 11px;
    font-weight: 600;
    color: #8e8e93;
  }

  .ch-dir {
    font-size: 10px;
    color: #a5b4fc;
    text-transform: none;
    letter-spacing: 0;
    background: rgba(129, 140, 248, 0.12);
    border: 1px solid rgba(129, 140, 248, 0.25);
    border-radius: 4px;
    padding: 1px 7px;
    cursor: pointer;
  }
  .ch-dir:hover {
    background: rgba(129, 140, 248, 0.22);
  }

  .ch-close {
    margin-left: auto;
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    display: flex;
    padding: 2px;
    border-radius: 4px;
  }
  .ch-close:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #e3e3e6;
  }

  .ch-symbol {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 12px 8px;
    border-bottom: 1px solid #2d2d34;
  }
  .ch-symbol code {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #e3e3e6;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ch-body {
    flex: 1;
    overflow: auto;
    padding: 4px 0;
  }

  .ch-empty {
    padding: 16px 14px;
    color: #6b7280;
    font-size: 12px;
    line-height: 1.6;
  }
  .ch-empty code {
    font-family: 'Fira Code', monospace;
    color: #a5b4fc;
  }
  .ch-empty kbd {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 3px;
    padding: 1px 4px;
    font-size: 10px;
    font-family: inherit;
  }
</style>
