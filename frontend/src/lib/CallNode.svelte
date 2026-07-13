<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';
  import Self from './CallNode.svelte';

  // node: a call-hierarchy tree node (see store.callHierarchy).
  // depth: indentation level. ancestors: Set of "path:line" up the chain, used
  // as a cycle guard when expanding.
  let { node, depth = 0, ancestors = new Set() } = $props();

  const childAncestors = $derived(new Set([...ancestors, `${node.path}:${node.line}`]));

  function toggle() {
    store.expandNode(node, ancestors);
  }

  function jump(e) {
    e.stopPropagation();
    store.openFile(node.path, node.line);
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="call-row" style="padding-left:{8 + depth * 14}px" onclick={toggle} title={node.detail}>
  <span class="call-caret">
    <Icon name={node.expanded ? 'chevronDown' : 'chevronRight'} size={12} color="#6b7280" />
  </span>
  <span class="call-name">{node.name}</span>
  {#if node.detail}<span class="call-detail">{node.detail}</span>{/if}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <span class="call-loc" onclick={jump} title="Open {node.path}:{node.line}">
    {node.path.split('/').pop()}:{node.line}
  </span>
</div>

{#if node.expanded}
  {#if node.loading}
    <div class="call-hint" style="padding-left:{22 + depth * 14}px">…</div>
  {:else if node.children.length === 0}
    <div class="call-hint" style="padding-left:{22 + depth * 14}px">
      {store.callHierarchy?.direction === 'incoming' ? 'No callers' : 'No callees'}
    </div>
  {:else}
    {#each node.children as child (child.path + ':' + child.line + ':' + child.name)}
      <Self node={child} depth={depth + 1} ancestors={childAncestors} />
    {/each}
  {/if}
{/if}

<style>
  .call-row {
    display: flex;
    align-items: baseline;
    gap: 6px;
    padding-top: 3px;
    padding-bottom: 3px;
    padding-right: 12px;
    cursor: pointer;
    white-space: nowrap;
  }
  .call-row:hover {
    background: rgba(99, 102, 241, 0.12);
  }
  .call-caret {
    display: inline-flex;
    align-self: center;
    flex-shrink: 0;
  }
  .call-name {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #e3e3e6;
    flex-shrink: 0;
  }
  .call-detail {
    font-size: 11px;
    color: #5d5d66;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .call-loc {
    margin-left: auto;
    font-size: 10px;
    color: #6b7280;
    flex-shrink: 0;
    padding-left: 8px;
  }
  .call-loc:hover {
    color: #a5b4fc;
    text-decoration: underline;
  }
  .call-hint {
    padding-top: 2px;
    padding-bottom: 4px;
    color: #5d5d66;
    font-size: 11px;
  }
</style>
