<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let symbols = $state([]);
  let loading = $state(false);
  let unsupported = $state(false);

  // LSP SymbolKind (1-26) -> short glyph + color. Only the common ones are
  // given a distinct look; the rest fall back to a neutral dot.
  const kindStyle = {
    5:  { g: 'C', c: '#e5c07b' }, // Class
    6:  { g: 'ƒ', c: '#61afef' }, // Method
    9:  { g: 'ƒ', c: '#61afef' }, // Constructor
    12: { g: 'ƒ', c: '#c678dd' }, // Function
    11: { g: 'I', c: '#56b6c2' }, // Interface
    23: { g: 'S', c: '#e5c07b' }, // Struct
    10: { g: 'E', c: '#e06c75' }, // Enum
    8:  { g: '•', c: '#98c379' }, // Field
    13: { g: 'v', c: '#98c379' }, // Variable
    14: { g: 'k', c: '#d19a66' }, // Constant
    2:  { g: 'M', c: '#abb2bf' }, // Module
  };
  const styleFor = (kind) => kindStyle[kind] || { g: '•', c: '#8e8e93' };

  // Flatten hierarchical DocumentSymbol[] or flat SymbolInformation[] into
  // rows with a depth for indentation and a 1-based line to jump to.
  function flatten(list, depth = 0, out = []) {
    for (const s of list) {
      const range = s.selectionRange || s.range || s.location?.range;
      const line = range ? range.start.line + 1 : 1;
      out.push({ name: s.name, kind: s.kind, detail: s.detail, line, depth });
      if (Array.isArray(s.children) && s.children.length) {
        flatten(s.children, depth + 1, out);
      }
    }
    return out;
  }

  async function load(attempt = 0) {
    const ed = window.editorInstance;
    const model = ed?.getModel?.();
    if (!model) { symbols = []; return; }
    loading = true;
    unsupported = false;
    const { documentSymbols, lspReady } = await import('./lsp.svelte.js');
    const res = await documentSymbols(model);
    if (res.length === 0) {
      // gopls et al. may still be indexing right after open — retry a few times.
      if (!lspReady() && attempt < 6) {
        setTimeout(() => load(attempt + 1), 700);
        return;
      }
      symbols = [];
      unsupported = !lspReady();
      loading = false;
      return;
    }
    symbols = flatten(res);
    loading = false;
  }

  // Reload whenever the active file changes.
  $effect(() => {
    const path = store.activePath;
    symbols = [];
    if (path && !path.includes(':')) {
      load(0);
    }
  });

  function jump(line) {
    if (store.activePath) store.openFile(store.activePath, line);
  }
</script>

<div class="outline-panel">
  <div class="outline-header">
    <span class="outline-title">Outline</span>
    <button class="outline-refresh" title="Refresh" onclick={() => load(0)}>
      <Icon name="refresh" size={12} />
    </button>
  </div>

  <div class="outline-body">
    {#if loading && symbols.length === 0}
      <div class="outline-empty">Loading symbols…</div>
    {:else if symbols.length === 0}
      <div class="outline-empty">
        {#if unsupported}
          No symbol provider for this file. Outline needs a language server
          (Go, TS/JS, Python, Rust, C/C++, Lua, Ruby).
        {:else}
          No symbols found.
        {/if}
      </div>
    {:else}
      {#each symbols as sym (sym.line + ':' + sym.name + ':' + sym.depth)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="outline-item"
          style="padding-left: {8 + sym.depth * 14}px"
          title={sym.detail || sym.name}
          onclick={() => jump(sym.line)}
        >
          <span class="outline-glyph" style="color: {styleFor(sym.kind).c}">{styleFor(sym.kind).g}</span>
          <span class="outline-name">{sym.name}</span>
          {#if sym.detail}<span class="outline-detail">{sym.detail}</span>{/if}
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .outline-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    color: #c4c4cc;
    font-size: 13px;
  }
  .outline-header {
    display: flex;
    align-items: center;
    padding: 10px 12px 8px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    border-bottom: 1px solid #2d2d34;
  }
  .outline-title {
    font-size: 11px;
    font-weight: 600;
    color: #8e8e93;
    flex: 1;
  }
  .outline-refresh {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    display: flex;
    padding: 2px;
    border-radius: 4px;
  }
  .outline-refresh:hover { background: rgba(255,255,255,0.08); color: #e3e3e6; }
  .outline-body { flex: 1; overflow-y: auto; padding: 4px 0; }
  .outline-empty {
    padding: 16px 14px;
    color: #6b7280;
    font-size: 12px;
    line-height: 1.6;
  }
  .outline-item {
    display: flex;
    align-items: baseline;
    gap: 8px;
    padding: 3px 12px 3px 8px;
    cursor: pointer;
    white-space: nowrap;
    overflow: hidden;
  }
  .outline-item:hover { background: rgba(99, 102, 241, 0.12); }
  .outline-glyph {
    font-family: 'Fira Code', monospace;
    font-size: 12px;
    width: 12px;
    text-align: center;
    flex-shrink: 0;
    font-weight: 600;
  }
  .outline-name {
    color: #e3e3e6;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .outline-detail {
    color: #5d5d66;
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
