<script>
  import { onMount, onDestroy } from 'svelte';
  import { store } from './store.svelte.js';
  import monaco from './monaco.js';

  import MarkdownPreview from './MarkdownPreview.svelte';
  import CsvViewer from './CsvViewer.svelte';
  import SqliteViewer from './SqliteViewer.svelte';
  import Icon from './Icon.svelte';
  import CommitViewer from './CommitViewer.svelte';
  import CommitGraph from './CommitGraph.svelte';
  import InsightsPanel from './InsightsPanel.svelte';

  let editorContainer = $state(null);
  let editor = $state.raw(null);
  let models = {};
  let showPreview = $state(true);

  // Diff Editor state
  let diffContainer = $state(null);
  let diffEditor = null;

  // Blame panel / selection state
  let blamePanel = $state(null);
  let blamePanelDetail = $state(null);
  const commitDetailsCache = {};
  const panelDetailLoading = new Set();

  // Selection / line history state
  let hasSelection = $state(false);
  let selectionStart = $state(1);
  let selectionEnd = $state(1);
  let loadingLineHistory = $state(false);
  let loadingPrevRev = $state(false);

  const isMarkdown = $derived(store.activeFile?.name && store.activeFile.name.toLowerCase().endsWith('.md'));

  $effect(() => {
    if (store.activePath) {
      const isMd = store.activeFile?.name && store.activeFile.name.toLowerCase().endsWith('.md');
      // Auto-enable preview for markdown, disable for non-markdown
      showPreview = !!isMd;
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

    if (file.isVideo || file.isAudio) {
      store.activeLanguage = file.isVideo ? 'Video' : 'Audio';
      return;
    }

    if (file.isCSV) {
      store.activeLanguage = 'CSV';
      return;
    }

    if (file.isSQLite) {
      store.activeLanguage = 'SQLite';
      return;
    }

    if (file.isBinary) {
      store.activeLanguage = 'Binary';
      return;
    }

    if (editor && !file.isBinary && !file.isImage && !file.isVideo && !file.isAudio && !file.isCSV && !file.isSQLite) {
      let model = models[file.path];
      // Read content here so this effect re-runs when a Refresh re-pulls the
      // file from disk, letting us sync an already-cached model's contents.
      const latestContent = file.content;

      if (!model) {
        // Construct absolute file URI matching the LSP workspace path
        const uri = monaco.Uri.parse(`file://${store.workspace.path}/${file.path}`);
        model = monaco.editor.getModel(uri);
        
        if (!model) {
          // Explicitly resolve language ID from file extension for guaranteed highlighting
          const ext = '.' + file.name.split('.').pop().toLowerCase();
          const lang = monaco.languages.getLanguages().find(l => l.extensions && l.extensions.includes(ext));
          const languageId = lang ? lang.id : undefined;

          model = monaco.editor.createModel(latestContent, languageId, uri);
          model.updateOptions({ tabSize: 2, insertSpaces: true });
        }

        models[file.path] = model;
      } else if (typeof latestContent === 'string' && model.getValue() !== latestContent) {
        // File was reloaded from disk (Refresh) — update the cached model in place.
        model.setValue(latestContent);
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

  // Watch for activeDiff to build/rebuild the Monaco Diff Editor
  $effect(() => {
    const activeDiff = store.activeDiff;
    let originalModel = null;
    let modifiedModel = null;

    if (activeDiff && diffContainer) {
      if (diffEditor) {
        diffEditor.dispose();
      }

      diffEditor = monaco.editor.createDiffEditor(diffContainer, {
        theme: 'vs-dark',
        readOnly: true,
        automaticLayout: true,
        fontSize: 14,
        fontFamily: "'Fira Code', Menlo, Monaco, 'Courier New', monospace",
        originalEditable: false,
        lineHeight: 22,
        renderSideBySide: true
      });

      const ext = '.' + activeDiff.path.split('.').pop().toLowerCase();
      const lang = monaco.languages.getLanguages().find(l => l.extensions && l.extensions.includes(ext));
      const languageId = lang ? lang.id : undefined;

      originalModel = monaco.editor.createModel(activeDiff.originalContent, languageId);
      modifiedModel = monaco.editor.createModel(activeDiff.modifiedContent, languageId);

      diffEditor.setModel({
        original: originalModel,
        modified: modifiedModel
      });
    }

    return () => {
      if (diffEditor) {
        diffEditor.dispose();
        diffEditor = null;
      }
      if (originalModel) {
        originalModel.dispose();
      }
      if (modifiedModel) {
        modifiedModel.dispose();
      }
    };
  });

  // Dispose models when tabs are closed to prevent memory leaks
  $effect(() => {
    const openPaths = new Set(store.openFiles.map(f => f.path));
    Object.keys(models).forEach(path => {
      if (!openPaths.has(path)) {
        if (models[path]) {
          models[path].dispose();
          delete models[path];
        }
      }
    });
  });

  // Load blame data when active file changes
  $effect(() => {
    const file = store.activeFile;
    store.currentLineBlame = null;
    blamePanel = null;
    blamePanelDetail = null;
    if (editor) {
      editor.deltaDecorations(blameDecorations, []);
      editor.deltaDecorations(heatmapDecorations, []);
      blameDecorations = [];
      heatmapDecorations = [];
    }

    if (file && !file.isBinary && !file.isImage && !file.isVideo && !file.isAudio && !file.isCSV && !file.isSQLite && store.git.isGit) {
      fetch(`/api/git/blame?path=${encodeURIComponent(file.path)}`)
        .then(res => res.json())
        .then(data => {
          currentBlame = data;
          window._vidianBlameLens = data;
          window._vidianFireBlameLens?.();
        })
        .catch(err => console.error("Failed to load blame info", err));
    }
  });

  let showAllBlame = $state(false);
  let showHeatmap = $state(false);
  let showCodeLens = $state(false);
  let currentBlame = $state([]);
  let blameDecorations = [];
  let heatmapDecorations = [];

  // Apply decorations when blame data or toggle changes
  $effect(() => {
    if (!editor || currentBlame.length === 0) return;
    if (showAllBlame || blamePanel) {
      applyAllBlameDecorations();
    } else {
      updateBlameDecoration();
    }
  });

  // Heatmap effect
  $effect(() => {
    if (!editor) return;
    if (showHeatmap && currentBlame.length > 0) {
      applyHeatmapDecorations();
    } else {
      heatmapDecorations = editor.deltaDecorations(heatmapDecorations, []);
    }
  });

  // Code lens effect
  $effect(() => {
    if (!editor) return;
    editor.updateOptions({ codeLens: showCodeLens });
    if (showCodeLens) {
      window._vidianBlameLens = currentBlame;
      window._vidianFireBlameLens?.();
    }
  });

  function heatClass(dateStr) {
    const days = Math.floor((Date.now() - new Date(dateStr).getTime()) / 86400000);
    if (days <= 7)   return 'heat-hot';
    if (days <= 30)  return 'heat-warm';
    if (days <= 90)  return 'heat-mild';
    if (days <= 365) return 'heat-cool';
    return 'heat-cold';
  }

  function applyHeatmapDecorations() {
    if (!editor || currentBlame.length === 0) return;
    const decorations = currentBlame.map(b => ({
      range: new monaco.Range(b.line, 1, b.line, 1),
      options: { linesDecorationsClassName: heatClass(b.date) }
    }));
    heatmapDecorations = editor.deltaDecorations(heatmapDecorations, decorations);
  }

  async function viewPreviousRevision() {
    if (!store.activeFile || loadingPrevRev) return;
    loadingPrevRev = true;
    try {
      const res = await fetch(`/api/git/log?path=${encodeURIComponent(store.activeFile.path)}`);
      const history = await res.json();
      if (history.length < 2) { loadingPrevRev = false; return; }
      const name = store.activeFile.path.split('/').pop();
      await store.openDiff(store.activeFile.path, history[1].hash, history[0].hash, `${name} ← prev revision`);
    } catch (e) { console.error(e); }
    loadingPrevRev = false;
  }

  async function viewLineHistory() {
    if (!store.activeFile || !hasSelection || loadingLineHistory) return;
    loadingLineHistory = true;
    try {
      const path = store.activeFile.path;
      const res = await fetch(`/api/git/line-history?path=${encodeURIComponent(path)}&start=${selectionStart}&end=${selectionEnd}`);
      const commits = await res.json();
      store.lineHistory = { path, start: selectionStart, end: selectionEnd, commits };
      store.sidebarTab = 'git';
    } catch (e) { console.error(e); }
    loadingLineHistory = false;
  }

  function applyAllBlameDecorations() {
    if (!editor || currentBlame.length === 0) return;
    const model = editor.getModel();
    if (!model) return;

    const decorations = currentBlame.map(record => {
      const lineContent = model.getLineContent(record.line);
      const endColumn = lineContent.length + 1;
      const author = record.author.length > 14 ? record.author.slice(0, 13) + '…' : record.author.padEnd(14);
      const summary = record.summary.length > 55 ? record.summary.slice(0, 54) + '…' : record.summary;
      return {
        range: new monaco.Range(record.line, endColumn, record.line, endColumn),
        options: {
          after: {
            content: `    ${author}  ${record.date}  •  ${summary}`,
            inlineClassName: 'monaco-git-blame-inline'
          }
        }
      };
    });

    blameDecorations = editor.deltaDecorations(blameDecorations, decorations);
  }

  function updateBlameDecoration() {
    if (showAllBlame || !editor || !store.activeFile || currentBlame.length === 0) return;

    const position = editor.getPosition();
    if (!position) return;

    const line = position.lineNumber;
    const record = currentBlame.find(r => r.line === line);

    if (!record) {
      blameDecorations = editor.deltaDecorations(blameDecorations, []);
      return;
    }

    const model = editor.getModel();
    if (!model) return;
    const lineContent = model.getLineContent(line);
    const endColumn = lineContent.length + 1;

    blameDecorations = editor.deltaDecorations(blameDecorations, [
      {
        range: new monaco.Range(line, endColumn, line, endColumn),
        options: {
          after: {
            content: `    ${record.author}, ${record.date} • ${record.summary}`,
            inlineClassName: 'monaco-git-blame-inline'
          }
        }
      }
    ]);
  }

  function handleBlameClick(line) {
    const record = currentBlame.find(r => r.line === line);
    if (!record || !store.activeFile) return;
    blamePanel = { line, record };
    blamePanelDetail = null;
    panelDetailLoading.delete(record.commit);
    if (!panelDetailLoading.has(record.commit)) {
      panelDetailLoading.add(record.commit);
      blamePanelDetail = null;
      fetch(`/api/git/commit?hash=${record.commit.slice(0, 8)}`)
        .then(res => res.ok ? res.json() : Promise.reject())
        .then(detail => {
          blamePanelDetail = detail;
          panelDetailLoading.delete(record.commit);
        })
        .catch(() => panelDetailLoading.delete(record.commit));
    }
    if (editor) editor.focus();
  }

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

    window.editorInstance = editor;

    // Code lens provider (registered once globally)
    if (!window._vidianCodeLensRegistered) {
      window._vidianCodeLensRegistered = true;
      window._vidianBlameLens = [];
      window._vidianBlameLensListeners = [];
      window._vidianFireBlameLens = () => window._vidianBlameLensListeners.forEach(fn => fn());
      monaco.languages.registerCodeLensProvider('*', {
        onDidChangeCodeLenses: listener => {
          window._vidianBlameLensListeners.push(listener);
          return { dispose: () => { window._vidianBlameLensListeners = window._vidianBlameLensListeners.filter(l => l !== listener); } };
        },
        provideCodeLenses(model) {
          const blame = window._vidianBlameLens || [];
          const lenses = [];
          let prevCommit = null;
          for (const b of blame) {
            if (b.commit !== prevCommit) {
              lenses.push({
                range: { startLineNumber: b.line, startColumn: 1, endLineNumber: b.line, endColumn: 1 },
                id: String(b.line),
                command: { id: 'vidian.blame.noop', title: `${b.author}  ·  ${b.date}  —  ${b.summary.slice(0, 60)}` }
              });
              prevCommit = b.commit;
            }
          }
          return { lenses, dispose: () => {} };
        },
        resolveCodeLens(model, lens) { return lens; }
      });
    }

    editor.onDidChangeCursorPosition(e => {
      store.cursorPos = { line: e.position.lineNumber, column: e.position.column };
      store.currentLineBlame = currentBlame.find(r => r.line === e.position.lineNumber) || null;
      updateBlameDecoration();
    });

    editor.onDidChangeCursorSelection(e => {
      const sel = e.selection;
      hasSelection = sel.startLineNumber !== sel.endLineNumber;
      selectionStart = sel.startLineNumber;
      selectionEnd = sel.endLineNumber;
    });

    const noopMouseMove = () => {};
    const noopMouseLeave = () => {};

    editor.onMouseMove(noopMouseMove);
    editor.onMouseLeave(noopMouseLeave);

    // Click handler for inline blame annotations: open the blame panel for the clicked line
    editor.getDomNode().addEventListener('click', (e) => {
      if (!currentBlame.length) return;
      const target = e.target;
      if (!(target instanceof HTMLElement)) return;
      const blameEl = target.closest('.monaco-git-blame-inline');
      if (!blameEl) return;

      const monacoEl = editor.getDomNode();
      if (!monacoEl) return;
      const rows = monacoEl.querySelectorAll('.view-line');
      let matchedLine = null;
      for (const row of rows) {
        if (row.contains(blameEl)) {
          const lineAttr = row.getAttribute('line');
          if (lineAttr) matchedLine = parseInt(lineAttr, 10);
          break;
        }
      }
      if (matchedLine && store.activeFile) {
        const record = currentBlame.find(r => r.line === matchedLine);
        if (record) {
          handleBlameClick(matchedLine);
        }
      }
    });

    // Register LSP providers
    const { registerLspProviders } = await import('./lsp.svelte.js');
    registerLspProviders();
  });

  onDestroy(() => {
    if (editor) {
      editor.dispose();
    }
    if (diffEditor) {
      diffEditor.dispose();
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
  {#if store.activeDiff}
    <!-- Diff Editor Container -->
    <div class="diff-editor-container-outer">
      <div bind:this={diffContainer} class="diff-editor-container-inner"></div>
    </div>
  {:else if store.activeFile?.isCommit}
    <!-- Commit Viewer -->
    <CommitViewer commit={store.activeFile.commitInfo} />
  {:else if store.activeFile?.isGraph}
    <!-- Commit Graph -->
    <CommitGraph />
  {:else if store.activeFile?.isInsights}
    <!-- Repo Insights -->
    <InsightsPanel />
  {:else if !store.activePath}
    <!-- Empty State / Welcome Screen -->
    <div class="welcome-screen">
      <div class="logo-wrapper">
        <img src="/favicon.svg" alt="Vidian Logo" class="logo-img" />
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
  {:else if store.activeFile?.isVideo}
    <!-- Video Player -->
    <div class="media-previewer">
      <div class="media-card">
        <div class="media-header">
          <span>{store.activeFile.name}</span>
        </div>
        <div class="media-body">
          <!-- svelte-ignore a11y_media_has_caption -->
          <video controls preload="metadata" src={store.activeFile.mediaUrl}>
            Your browser does not support video playback.
          </video>
        </div>
      </div>
    </div>
  {:else if store.activeFile?.isAudio}
    <!-- Audio Player -->
    <div class="media-previewer">
      <div class="media-card audio-card">
        <div class="media-header">
          <span>{store.activeFile.name}</span>
        </div>
        <div class="media-body audio-body">
          <div class="audio-icon">🎵</div>
          <audio controls preload="metadata" src={store.activeFile.mediaUrl}>
            Your browser does not support audio playback.
          </audio>
        </div>
      </div>
    </div>
  {:else if store.activeFile?.isCSV}
    <!-- CSV Table Viewer -->
    <CsvViewer content={store.activeFile.content} />
  {:else if store.activeFile?.isSQLite}
    <!-- SQLite Browser -->
    <SqliteViewer path={store.activeFile.path} />
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

  <!-- Full preview pane (markdown default) -->
  {#if isMarkdown && showPreview && !store.activeDiff && store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage && !store.activeFile?.isVideo && !store.activeFile?.isAudio && !store.activeFile?.isCSV && !store.activeFile?.isSQLite && !store.activeFile?.isCommit}
    <div class="preview-pane-full">
      <MarkdownPreview content={store.activeFile.content} />
    </div>
  {/if}

  <!-- The split workspace container (Monaco + optional preview side-by-side) -->
  <div 
    class="editor-split-container"
    style="display: {store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage && !store.activeFile?.isVideo && !store.activeFile?.isAudio && !store.activeFile?.isCSV && !store.activeFile?.isSQLite && !store.activeFile?.isCommit && (!isMarkdown || !showPreview) ? 'flex' : 'none'}"
  >
    <div 
      bind:this={editorContainer} 
      class="monaco-container"
    ></div>
  </div>

  {#if store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage && !store.activeFile?.isVideo && !store.activeFile?.isAudio && !store.activeFile?.isCSV && !store.activeFile?.isSQLite && !store.activeFile?.isCommit && isMarkdown}
    <!-- Floating action button: toggle split raw view -->
    <button
      class="preview-toggle-btn"
      class:active={!showPreview}
      onclick={() => showPreview = !showPreview}
      title={showPreview ? "View Raw Markdown" : "Back to Preview"}
    >
      <Icon name={showPreview ? "split" : "close"} size={14} color="#ffffff" />
      <span>{showPreview ? "View Raw" : "Close Raw"}</span>
    </button>
  {/if}

  {#if store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage && !store.activeFile?.isVideo && !store.activeFile?.isAudio && !store.activeFile?.isCSV && !store.activeFile?.isSQLite && !store.activeFile?.isCommit && !store.activeFile?.isGraph && !store.activeDiff}
    <div class="editor-actions">
      {#if store.git.isGit}
        <button
          class:active={showAllBlame}
          onclick={() => { showAllBlame = !showAllBlame; if (!showAllBlame) { blameDecorations = editor?.deltaDecorations(blameDecorations, []); updateBlameDecoration(); } }}
          title="Toggle inline blame annotations"
        ><Icon name="gitCommit" size={11} /> Blame</button>

        <button
          class:active={showHeatmap}
          onclick={() => showHeatmap = !showHeatmap}
          title="Toggle gutter heatmap (line age)"
        >Heatmap</button>

        <button
          class:active={showCodeLens}
          onclick={() => showCodeLens = !showCodeLens}
          title="Toggle code lens (commit info above blocks)"
        >Lens</button>

        <div class="act-sep"></div>

        <button
          class:loading={loadingPrevRev}
          onclick={viewPreviousRevision}
          title="Diff this file vs its previous revision"
        >Prev Rev</button>

        {#if store.currentLineBlame}
          <div class="act-sep"></div>
          <button
            onclick={() => store.openCommit(store.currentLineBlame.commit)}
            title="Open commit for current line"
          >Open Commit</button>
        {/if}
      {/if}

      {#if hasSelection && store.git.isGit}
        <div class="act-sep"></div>
        <button
          class:loading={loadingLineHistory}
          onclick={viewLineHistory}
          title="Show git history for selected lines"
        >Line History</button>
      {/if}
    </div>
  {/if}

  <!-- Blame popup removed: replaced with inline blame accordion panel. -->

  {#if blamePanel}
    <div class="blame-panel">
      <div class="bp-header">
        <div>
          <span class="bp-hash">{blamePanel.record.commit.slice(0, 8)}</span>
          <span class="bp-date">{blamePanel.record.date}</span>
        </div>
        <button class="bp-close" onclick={() => blamePanel = null} title="Close">×</button>
      </div>
      <div class="bp-author">{blamePanel.record.author}</div>
      <div class="bp-message">{blamePanel.record.summary}</div>
      <div class="bp-actions">
        <button
          class="bp-action"
          onclick={() => store.openFileAtCommit(store.activeFile?.path, blamePanel.record.commit)}
          title="Open this file version at this commit"
        >Open File Diff</button>
        <button
          class="bp-action"
          onclick={() => store.openCommit(blamePanel.record.commit)}
          title="Open commit details"
        >Open Commit</button>
      </div>
      {#if blamePanelDetail}
        <div class="bp-detail">
          <div class="bp-detail-row">
            <span>Files:</span>
            <strong>{blamePanelDetail.files?.length ?? 0}</strong>
          </div>
          <div class="bp-detail-row">
            <span>Insertions:</span>
            <strong class="diff-added">+{blamePanelDetail.stats?.insertions ?? 0}</strong>
          </div>
          <div class="bp-detail-row">
            <span>Deletions:</span>
            <strong class="diff-removed">-{blamePanelDetail.stats?.deletions ?? 0}</strong>
          </div>
          {#if blamePanelDetail.body}
            <div class="bp-body">{blamePanelDetail.body}</div>
          {/if}
        </div>
      {/if}
    </div>
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

  .diff-editor-container-outer {
    width: 100%;
    height: 100%;
    background-color: #1e1e24;
  }

  .diff-editor-container-inner {
    width: 100%;
    height: 100%;
  }

  :global(.monaco-git-blame-inline) {
    color: #8e8e93;
    opacity: 0.55;
    font-style: italic;
    font-size: 11px;
    pointer-events: auto;
    user-select: none;
    cursor: pointer;
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

  .logo-img {
    width: 80px;
    height: 80px;
    margin: 0 auto 1.5rem;
    filter: drop-shadow(0 10px 20px rgba(99, 102, 241, 0.25));
    animation: pulse-glow 3.5s infinite ease-in-out;
  }

  @keyframes pulse-glow {
    0%, 100% {
      filter: drop-shadow(0 10px 20px rgba(99, 102, 241, 0.25));
      transform: scale(1);
    }
    50% {
      filter: drop-shadow(0 10px 30px rgba(245, 158, 11, 0.45));
      transform: scale(1.025);
    }
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

  /* Media Player Styles */
  .media-previewer {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    background-color: #121214;
    padding: 2rem;
    box-sizing: border-box;
  }

  .media-card {
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

  .media-header {
    background-color: #141416;
    padding: 8px 16px;
    font-size: 12px;
    color: #8e8e93;
    border-bottom: 1px solid #2d2d34;
  }

  .media-body {
    padding: 0;
    display: flex;
    justify-content: center;
    align-items: center;
    background-color: #0a0a0c;
  }

  .media-body video {
    max-width: 100%;
    max-height: 70vh;
    outline: none;
  }

  .audio-card {
    min-width: 400px;
  }

  .audio-body {
    flex-direction: column;
    padding: 2rem;
    gap: 1.5rem;
    background-color: #1e1e24;
  }

  .audio-icon {
    font-size: 3rem;
  }

  .audio-body audio {
    width: 100%;
    outline: none;
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

  .preview-pane-full {
    position: absolute;
    inset: 0;
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

  /* Editor action toolbar */
  .editor-actions {
    position: absolute;
    top: 8px;
    right: 130px;
    z-index: 50;
    display: flex;
    align-items: center;
    gap: 1px;
    background: rgba(18, 18, 22, 0.92);
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: 6px;
    padding: 2px 3px;
    backdrop-filter: blur(6px);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.35);
  }

  .editor-actions button {
    background: none;
    border: none;
    color: #8e8e93;
    cursor: pointer;
    padding: 3px 7px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 4px;
    transition: background-color 0.12s, color 0.12s;
    white-space: nowrap;
  }

  .editor-actions button:hover {
    background: rgba(255, 255, 255, 0.07);
    color: #e3e3e6;
  }

  .editor-actions button.active {
    background: rgba(99, 102, 241, 0.18);
    color: #818cf8;
  }

  .editor-actions button.loading {
    opacity: 0.5;
    pointer-events: none;
  }

  .act-sep {
    width: 1px;
    height: 14px;
    background: rgba(255, 255, 255, 0.1);
    margin: 0 2px;
  }

  /* Gutter heatmap classes */
  :global(.heat-hot)  { background: #ef4444; width: 3px; margin-left: 1px; border-radius: 2px; }
  :global(.heat-warm) { background: #f97316; width: 3px; margin-left: 1px; border-radius: 2px; }
  :global(.heat-mild) { background: #eab308; width: 3px; margin-left: 1px; border-radius: 2px; }
  :global(.heat-cool) { background: #3b82f6; width: 3px; margin-left: 1px; border-radius: 2px; }
  :global(.heat-cold) { background: #6b7280; width: 3px; margin-left: 1px; border-radius: 2px; }

  .blame-panel {
    position: absolute;
    right: 170px;
    bottom: 18px;
    z-index: 9999;
    background: #1e1e2a;
    border: 1px solid #3a3a50;
    border-radius: 10px;
    padding: 12px 14px;
    width: 320px;
    max-width: 360px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
    font-size: 12px;
    line-height: 1.45;
  }

  .blame-panel .bp-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .blame-panel .bp-close {
    background: none;
    border: none;
    color: #6b7280;
    cursor: pointer;
    font-size: 14px;
    line-height: 1;
    padding: 0 2px;
    border-radius: 4px;
  }

  .blame-panel .bp-close:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #e3e3e6;
  }

  .blame-panel .bp-hash {
    font-family: monospace;
    font-size: 11px;
    color: #818cf8;
    background: rgba(99, 102, 241, 0.12);
    padding: 1px 6px;
    border-radius: 4px;
  }

  .blame-panel .bp-date {
    font-size: 11px;
    color: #8e8e93;
    margin-left: 8px;
  }

  .blame-panel .bp-author {
    font-weight: 600;
    color: #e3e3e6;
    margin-bottom: 3px;
    margin-top: 2px;
  }

  .blame-panel .bp-message {
    color: #c4c4cc;
    font-size: 12px;
    margin-bottom: 8px;
  }

  .blame-panel .bp-actions {
    display: flex;
    gap: 6px;
    margin-bottom: 8px;
  }

  .blame-panel .bp-action {
    flex: 1;
    background: rgba(99, 102, 241, 0.12);
    color: #818cf8;
    border: 1px solid rgba(99, 102, 241, 0.25);
    border-radius: 5px;
    padding: 4px 7px;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    transition: background-color 0.12s, border-color 0.12s;
  }

  .blame-panel .bp-action:hover {
    background: rgba(99, 102, 241, 0.25);
    border-color: rgba(99, 102, 241, 0.55);
  }

  .blame-panel .bp-detail {
    border-top: 1px solid #2d2d40;
    padding-top: 8px;
  }

  .blame-panel .bp-detail-row {
    display: flex;
    justify-content: space-between;
    font-size: 11px;
    color: #8e8e93;
    margin-bottom: 3px;
  }

  .blame-panel .bp-detail-row strong {
    color: #e3e3e6;
  }

  .blame-panel .diff-added {
    color: #4ade80;
  }

  .blame-panel .diff-removed {
    color: #f87171;
  }

  .blame-panel .bp-body {
    margin-top: 6px;
    font-size: 11px;
    color: #c4c4cc;
    max-height: 140px;
    overflow: auto;
  }

  .blame-popup {
    position: fixed;
    z-index: 9999;
    background: #1e1e2a;
    border: 1px solid #3a3a50;
    border-radius: 8px;
    padding: 10px 14px;
    min-width: 280px;
    max-width: 380px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.45);
    pointer-events: none;
    font-size: 12px;
    line-height: 1.5;
  }

  .bp-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 5px;
  }

  .bp-hash {
    font-family: monospace;
    font-size: 11px;
    color: #818cf8;
    background: rgba(99, 102, 241, 0.12);
    padding: 1px 6px;
    border-radius: 4px;
  }

  .bp-date {
    font-size: 11px;
    color: #8e8e93;
  }

  .bp-author {
    font-weight: 600;
    color: #e3e3e6;
    margin-bottom: 3px;
  }

  .bp-message {
    color: #c4c4cc;
    font-size: 12px;
  }

  .bp-files {
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px solid #2d2d40;
    font-size: 11px;
    color: #8e8e93;
  }
</style>
