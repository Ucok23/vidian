<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';
  import MarkdownPreview from './MarkdownPreview.svelte';

  const ai = $derived(store.aiExplain);
</script>

<div class="ai-panel">
  <div class="ai-header">
    <Icon name="sparkles" size={14} color="#a5b4fc" />
    <span class="ai-title">AI Explain</span>
    <button class="ai-close" title="Clear" onclick={() => store.clearExplain()}>
      <Icon name="close" size={13} />
    </button>
  </div>

  {#if ai}
    <div class="ai-subject" title={ai.title}>{ai.title}</div>
  {/if}

  <div class="ai-body">
    {#if !ai}
      <div class="ai-empty">
        Open a file and click <em>Explain</em> in the editor toolbar (or select
        code first to explain just that snippet). Requires an AI provider in Settings.
      </div>
    {:else if ai.loading}
      <div class="ai-empty">
        <span class="ai-spinner"></span> Thinking…
      </div>
    {:else if ai.error}
      <div class="ai-error">{ai.error}</div>
    {:else}
      <div class="ai-markdown">
        <MarkdownPreview content={ai.text} />
      </div>
    {/if}
  </div>
</div>

<style>
  .ai-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    color: #c4c4cc;
    font-size: 13px;
  }
  .ai-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px 8px;
  }
  .ai-title {
    font-size: 11px;
    font-weight: 600;
    color: #8e8e93;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    flex: 1;
  }
  .ai-close {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    display: flex;
    padding: 2px;
    border-radius: 4px;
  }
  .ai-close:hover { background: rgba(255,255,255,0.08); color: #e3e3e6; }
  .ai-subject {
    padding: 0 12px 8px;
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    color: #a5b4fc;
    border-bottom: 1px solid #2d2d34;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .ai-body { flex: 1; overflow-y: auto; position: relative; }
  .ai-empty {
    padding: 16px 14px;
    color: #6b7280;
    font-size: 12px;
    line-height: 1.6;
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }
  .ai-error {
    margin: 12px;
    padding: 10px 12px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    border-radius: 6px;
    color: #f87171;
    font-size: 12px;
    white-space: pre-wrap;
  }
  /* MarkdownPreview fills its host; give it a scrollable area. */
  .ai-markdown {
    position: absolute;
    inset: 0;
  }
  .ai-spinner {
    width: 12px;
    height: 12px;
    border: 2px solid rgba(165, 180, 252, 0.3);
    border-top-color: #a5b4fc;
    border-radius: 50%;
    display: inline-block;
    animation: ai-spin 0.7s linear infinite;
  }
  @keyframes ai-spin { to { transform: rotate(360deg); } }
</style>
