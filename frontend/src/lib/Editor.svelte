<script>
  import { onMount, onDestroy } from 'svelte';
  import { store } from './store.svelte.js';
  import monaco from './monaco.js';

  import MarkdownPreview from './MarkdownPreview.svelte';
  import Icon from './Icon.svelte';

  let editorContainer = $state(null);
  let editor = $state.raw(null);
  let models = {};
  let showPreview = $state(false);

  const isMarkdown = $derived(store.activeFile?.name && store.activeFile.name.toLowerCase().endsWith('.md'));

  $effect(() => {
    if (store.activePath) {
      const isMd = store.activeFile?.name && store.activeFile.name.toLowerCase().endsWith('.md');
      if (!isMd) {
        showPreview = false;
      }
    } else {
      showPreview = false;
    }
  });

  // Watch for active file changes using $effect
  $effect(() => {
    const file = store.activeFile;
    if (!file) {
      store.activeLanguage = 'Plain Text';
      return;
    }
    
    if (file.isImage) {
      store.activeLanguage = 'Image';
      return;
    }
    
    if (file.isBinary) {
      store.activeLanguage = 'Binary';
      return;
    }

    if (editor && !file.isBinary && !file.isImage) {
      let model = models[file.path];
      
      if (!model) {
        // Construct absolute file URI matching the LSP workspace path
        const uri = monaco.Uri.parse(`file://${store.workspace.path}/${file.path}`);
        model = monaco.editor.getModel(uri);
        
        if (!model) {
          // Explicitly resolve language ID from file extension for guaranteed highlighting
          const ext = '.' + file.name.split('.').pop().toLowerCase();
          const lang = monaco.languages.getLanguages().find(l => l.extensions && l.extensions.includes(ext));
          const languageId = lang ? lang.id : undefined;

          model = monaco.editor.createModel(file.content, languageId, uri);
          // Configure model options if needed
          model.updateOptions({ tabSize: 2, insertSpaces: true });
        }
        
        models[file.path] = model;
      }
      
      editor.setModel(model);
      const langId = (typeof model.getLanguageId === 'function') ? model.getLanguageId() : (typeof model.getModeId === 'function' ? model.getModeId() : 'plaintext');
      store.activeLanguage = langId.charAt(0).toUpperCase() + langId.slice(1);

      // Dynamically initialize or switch LSP server for active file
      import('./lsp.svelte.js').then(({ initLsp }) => {
        initLsp(store.workspace.path, file.name);
      });
    }
  });

  onMount(async () => {
    // Create Monaco instance
    editor = monaco.editor.create(editorContainer, {
      theme: 'vs-dark',
      readOnly: true,
      automaticLayout: true,
      fontSize: 14,
      fontFamily: "'Fira Code', Menlo, Monaco, 'Courier New', monospace",
      minimap: {
        enabled: true
      },
      scrollBeyondLastLine: false,
      cursorBlinking: 'smooth',
      cursorSmoothCaretAnimation: 'on',
      renderLineHighlight: 'all',
      padding: {
        top: 10,
        bottom: 10
      },
      lineHeight: 22,
      scrollbar: {
        verticalScrollbarSize: 10,
        horizontalScrollbarSize: 10,
        useShadows: false
      },
      overviewRulerLanes: 0,
      hideCursorInOverviewRuler: true
    });

    // Patch the editor's existing codeEditorService to intercept file navigation
    const rawService = editor._codeEditorService;
    if (rawService) {
      const originalOpenCodeEditor = rawService.openCodeEditor.bind(rawService);
      rawService.openCodeEditor = async (modelInput, options, sideBySide) => {
        const uri = modelInput.resource;
        if (uri) {
          const wsPath = store.workspace.path;
          let relPath = uri.path;
          if (relPath.startsWith(wsPath)) {
            relPath = relPath.slice(wsPath.length);
          }
          if (relPath.startsWith('/')) {
            relPath = relPath.slice(1);
          }
          
          const line = modelInput.options?.selection ? modelInput.options.selection.startLineNumber : 1;
          store.openFile(relPath, line);
          return editor;
        }
        return originalOpenCodeEditor(modelInput, options, sideBySide);
      };
    }

    // Save to window for global access (e.g., revealLine from search)
    window.editorInstance = editor;

    // Listen to cursor position changes
    editor.onDidChangeCursorPosition(e => {
      store.cursorPos = {
        line: e.position.lineNumber,
        column: e.position.column
      };
    });

    // Register LSP providers
    const { registerLspProviders } = await import('./lsp.svelte.js');
    registerLspProviders();
  });

  onDestroy(() => {
    if (editor) {
      editor.dispose();
    }
    // Dispose all models
    Object.values(models).forEach(model => model.dispose());
    window.editorInstance = null;
  });

  // Helper to format bytes
  function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }
</script>

<div class="editor-host">
  {#if !store.activePath}
    <!-- Empty State / Welcome Screen -->
    <div class="welcome-screen">
      <div class="logo-wrapper">
        <div class="logo-icon"></div>
        <h1>Vidian</h1>
        <p class="subtitle">Lightweight. Beautiful. Read-only code viewer.</p>
      </div>

      <div class="shortcuts-card">
        <h3>Quick Shortcuts</h3>
        <div class="shortcut-row">
          <span>Search Files</span>
          <kbd>Ctrl + P</kbd>
        </div>
        <div class="shortcut-row">
          <span>Search Content</span>
          <kbd>Ctrl + Shift + F</kbd>
        </div>
        <div class="shortcut-row">
          <span>Toggle Sidebar</span>
          <kbd>Ctrl + B</kbd>
        </div>
      </div>

      <div class="footer-info">
        Workspace: <span class="path">{store.workspace.path}</span>
      </div>
    </div>
  {:else if store.activeFile?.isImage}
    <!-- Image Previewer -->
    <div class="image-previewer">
      <div class="image-card">
        <div class="image-header">
          <span>{store.activeFile.name}</span>
        </div>
        <div class="image-body">
          <img src={store.activeFile.imageUrl} alt={store.activeFile.name} />
        </div>
      </div>
    </div>
  {:else if store.activeFile?.isBinary}
    <!-- Binary File Screen -->
    <div class="binary-screen">
      <div class="binary-card">
        <div class="binary-icon">📁</div>
        <h3>Binary File</h3>
        <p class="binary-desc">This file cannot be displayed because it is binary or has an unsupported encoding.</p>
        <div class="meta-row">
          <span>Size:</span>
          <strong>{formatBytes(store.activeFile.size)}</strong>
        </div>
        <div class="meta-row">
          <span>Mime Type:</span>
          <strong>{store.activeFile.mimeType}</strong>
        </div>
      </div>
    </div>
  {/if}

  <!-- The split workspace container -->
  <div 
    class="editor-split-container"
    style="display: {store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage ? 'flex' : 'none'}"
  >
    <div 
      bind:this={editorContainer} 
      class="monaco-container"
    ></div>

    {#if showPreview && isMarkdown}
      <div class="preview-pane">
        <MarkdownPreview content={store.activeFile.content} />
      </div>
    {/if}
  </div>

  {#if store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage && isMarkdown}
    <!-- Floating action button for markdown preview -->
    <button 
      class="preview-toggle-btn"
      class:active={showPreview}
      onclick={() => showPreview = !showPreview}
      title={showPreview ? "Close Preview" : "Open Preview to the Side"}
    >
      <Icon name={showPreview ? "close" : "split"} size={14} color="#ffffff" />
      <span>{showPreview ? "Close Preview" : "Split Preview"}</span>
    </button>
  {/if}
</div>

<style>
  .editor-host {
    position: relative;
    width: 100%;
    height: 100%;
    background-color: #1e1e24;
    overflow: hidden;
  }

  .monaco-container {
    width: 100%;
    height: 100%;
  }

  /* Welcome Screen styles */
  .welcome-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    color: #8e8e93;
    padding: 2rem;
    box-sizing: border-box;
    text-align: center;
    background: radial-gradient(circle at center, #1b1b20 0%, #121214 100%);
  }

  .logo-wrapper h1 {
    font-size: 2.2rem;
    font-weight: 700;
    margin: 0.5rem 0;
    background: linear-gradient(135deg, #a5b4fc 0%, #6366f1 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    letter-spacing: -0.02em;
  }

  .logo-icon {
    width: 64px;
    height: 64px;
    margin: 0 auto 1rem;
    background: linear-gradient(135deg, #6366f1 0%, #4f46e5 100%);
    border-radius: 16px;
    position: relative;
    box-shadow: 0 10px 25px -5px rgba(99, 102, 241, 0.4);
  }

  .logo-icon::after {
    content: '</>';
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: #ffffff;
    font-family: monospace;
    font-weight: bold;
    font-size: 22px;
  }

  .subtitle {
    font-size: 1.1rem;
    margin-bottom: 3rem;
    color: #6b7280;
  }

  .shortcuts-card {
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 1.5rem 2rem;
    width: 100%;
    max-width: 380px;
    box-shadow: 0 4px 30px rgba(0, 0, 0, 0.2);
    backdrop-filter: blur(5px);
  }

  .shortcuts-card h3 {
    margin-top: 0;
    margin-bottom: 1.2rem;
    font-size: 1rem;
    color: #e3e3e6;
    font-weight: 500;
  }

  .shortcut-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.8rem;
    font-size: 13px;
  }

  .shortcut-row:last-child {
    margin-bottom: 0;
  }

  kbd {
    background-color: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 4px;
    color: #e3e3e6;
    display: inline-block;
    font-size: 11px;
    font-family: inherit;
    font-weight: 500;
    line-height: 1;
    padding: 4px 6px;
    white-space: nowrap;
    box-shadow: 0 1px 0 rgba(0, 0, 0, 0.2);
  }

  .footer-info {
    position: absolute;
    bottom: 2rem;
    font-size: 12px;
    color: #4b5563;
  }

  .footer-info .path {
    font-family: monospace;
    color: #6b7280;
  }

  /* Image Previewer Styles */
  .image-previewer {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    background-color: #121214;
    padding: 2rem;
    box-sizing: border-box;
  }

  .image-card {
    border: 1px solid #2d2d34;
    border-radius: 8px;
    overflow: hidden;
    background-color: #1e1e24;
    max-width: 90%;
    max-height: 90%;
    display: flex;
    flex-direction: column;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  }

  .image-header {
    background-color: #141416;
    padding: 8px 16px;
    font-size: 12px;
    color: #8e8e93;
    border-bottom: 1px solid #2d2d34;
  }

  .image-body {
    padding: 24px;
    display: flex;
    justify-content: center;
    align-items: center;
    overflow: auto;
    background-image: 
      linear-gradient(45deg, #252528 25%, transparent 25%), 
      linear-gradient(-45deg, #252528 25%, transparent 25%), 
      linear-gradient(45deg, transparent 75%, #252528 75%), 
      linear-gradient(-45deg, transparent 75%, #252528 75%);
    background-size: 20px 20px;
    background-position: 0 0, 0 10px, 10px -10px, -10px 0px;
  }

  .image-body img {
    max-width: 100%;
    max-height: 70vh;
    object-fit: contain;
    border-radius: 4px;
  }

  /* Binary screen styles */
  .binary-screen {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    background-color: #121214;
  }

  .binary-card {
    background: #1e1e24;
    border: 1px solid #2d2d34;
    border-radius: 12px;
    padding: 2.5rem;
    text-align: center;
    max-width: 400px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
  }

  .binary-icon {
    font-size: 3rem;
    margin-bottom: 1rem;
  }

  .binary-card h3 {
    margin: 0;
    font-size: 1.2rem;
    color: #e3e3e6;
  }

  .binary-desc {
    color: #8e8e93;
    font-size: 13px;
    margin-top: 0.5rem;
    margin-bottom: 2rem;
    line-height: 1.5;
  }

  .meta-row {
    display: flex;
    justify-content: space-between;
    font-size: 13px;
    padding: 8px 0;
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .meta-row:last-child {
    border-bottom: none;
  }

  .meta-row span {
    color: #8e8e93;
  }

  .meta-row strong {
    color: #e3e3e6;
    font-family: monospace;
  }

  .editor-split-container {
    display: flex;
    width: 100%;
    height: 100%;
    position: relative;
  }

  .monaco-container {
    flex: 1;
    height: 100%;
    min-width: 0;
  }

  .preview-pane {
    flex: 1;
    height: 100%;
    border-left: 1px solid #2d2d34;
    min-width: 0;
    overflow: hidden;
    background-color: #1e1e24;
  }

  .preview-toggle-btn {
    position: absolute;
    top: 10px;
    right: 25px;
    z-index: 100;
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(99, 102, 241, 0.85);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: white;
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
    backdrop-filter: blur(4px);
    transition: background-color 0.15s, transform 0.15s;
  }

  .preview-toggle-btn:hover {
    background: #4f46e5;
    transform: translateY(-1px);
  }

  .preview-toggle-btn:active {
    transform: translateY(0);
  }

  .preview-toggle-btn.active {
    background: #ef4444;
  }

  .preview-toggle-btn.active:hover {
    background: #dc2626;
  }
</style>
