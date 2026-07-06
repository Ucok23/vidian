import { tick } from 'svelte';

const API_BASE = ''; // Same host

class AppStore {
  workspace = $state({ name: '', path: '' });
  workspaces = $state([]);
  currentWorkspaceId = $state(null);
  openFiles = $state([]);
  activePath = $state(null);
  sidebarTab = $state('explorer'); // 'explorer', 'search'
  settingsOpen = $state(false);
  searchQuery = $state('');
  searchResults = $state([]);
  isSearching = $state(false);
  cursorPos = $state({ line: 1, column: 1 });
  activeLanguage = $state('Plain Text');
  currentLineBlame = $state(null);
  lineHistory = $state(null);    // { path, start, end, commits: [] }
  compareResult = $state(null);  // { ref1, ref2, files: [], stat: '' }
  // References ("find all callers") result for the symbol under the cursor.
  // { symbol, loading, error, groups: [{ path, items: [{ line, column, preview }] }] }
  references = $state(null);
  // AI explanation of the active file/selection.
  // { title, loading, error, text }
  aiExplain = $state(null);
  
  // File tree expanded paths
  expandedPaths = $state(new Set());
  // Cache of directory contents: path -> list of FileInfo
  dirContents = $state({});

  // Git state
  git = $state({ isGit: false, currentBranch: '', branches: [] });
  isCheckingOut = $state(false);
  activeDiff = $state(null); // { path, originalContent, modifiedContent, title }

  // Active file derived state
  get activeFile() {
    return this.openFiles.find(f => f.path === this.activePath) || null;
  }

  // apiUrl builds a request URL scoped to the current workspace by appending the
  // ws query parameter. All backend endpoints (except /api/workspaces and
  // /api/ping) require it.
  apiUrl(path) {
    if (!this.currentWorkspaceId) return `${API_BASE}${path}`;
    const sep = path.includes('?') ? '&' : '?';
    return `${API_BASE}${path}${sep}ws=${encodeURIComponent(this.currentWorkspaceId)}`;
  }

  async init() {
    try {
      await this.loadWorkspaces();
      if (this.workspaces.length === 0) {
        console.error("No workspaces registered");
        return;
      }
      const urlWs = new URLSearchParams(window.location.search).get('ws');
      const current = this.workspaces.find(w => w.id === urlWs) || this.workspaces[0];
      this.currentWorkspaceId = current.id;
      this.workspace = { name: current.name, path: current.path };
      this.syncUrl(current.id);
      await this.loadWorkspaceData();

      // Restore a deep-linked file/line from the URL, if present and valid.
      const params = new URLSearchParams(window.location.search);
      const file = params.get('file');
      if (file) {
        const line = parseInt(params.get('line') || '', 10);
        await this.openFile(file, Number.isFinite(line) ? line : null);
      }
    } catch (err) {
      console.error("Failed to load workspace info", err);
    }
  }

  // loadWorkspaces refreshes the list of directories served by this instance.
  async loadWorkspaces() {
    const res = await fetch(`${API_BASE}/api/workspaces`);
    const list = await res.json();
    this.workspaces = Array.isArray(list) ? list : [];
  }

  // loadWorkspaceData (re)loads the file tree and git info for the current
  // workspace, resetting any per-workspace tree state first.
  async loadWorkspaceData() {
    this.dirContents = {};
    this.expandedPaths = new Set();
    await this.loadDir('');
    this.expandedPaths.add('');
    this.expandedPaths = new Set(this.expandedPaths);
    await this.loadGit();
  }

  // syncUrl keeps ?ws=<id> in the address bar so a refresh stays on the same
  // workspace without pushing a new history entry.
  syncUrl(id) {
    const url = new URL(window.location.href);
    if (url.searchParams.get('ws') !== id) {
      url.searchParams.set('ws', id);
      window.history.replaceState({}, '', url);
    }
  }

  async switchWorkspace(id) {
    if (id === this.currentWorkspaceId) return;
    const ws = this.workspaces.find(w => w.id === id);
    if (!ws) return;
    this.currentWorkspaceId = id;
    this.workspace = { name: ws.name, path: ws.path };
    this.syncUrl(id);
    // Reset all per-workspace UI state before loading the new tree.
    this.closeAllFiles();
    this.searchQuery = '';
    this.searchResults = [];
    this.currentLineBlame = null;
    this.lineHistory = null;
    this.compareResult = null;
    this.references = null;
    this.aiExplain = null;
    await this.loadWorkspaceData();
  }

  async loadDir(path) {
    try {
      const res = await fetch(this.apiUrl(`/api/dir?path=${encodeURIComponent(path)}`));
      const data = await res.json();
      // Sort: directories first, then files alphabetically
      data.sort((a, b) => {
        if (a.isDir && !b.isDir) return -1;
        if (!a.isDir && b.isDir) return 1;
        return a.name.localeCompare(b.name);
      });
      this.dirContents = { ...this.dirContents, [path]: data };
    } catch (err) {
      console.error("Failed to load directory", path, err);
    }
  }

  toggleFolder(path) {
    if (this.expandedPaths.has(path)) {
      this.expandedPaths.delete(path);
      this.expandedPaths = new Set(this.expandedPaths);
    } else {
      this.expandedPaths.add(path);
      this.expandedPaths = new Set(this.expandedPaths);
      if (!this.dirContents[path]) {
        this.loadDir(path);
      }
    }
  }

  async openFile(path, jumpToLine = null) {
    this.activeDiff = null;
    // Check if already open
    const exists = this.openFiles.some(f => f.path === path);
    if (!exists) {
      // Fetch content
      try {
        const res = await fetch(this.apiUrl(`/api/file?path=${encodeURIComponent(path)}`));
        const contentType = res.headers.get("content-type") || "";
        
        let fileData;
        if (contentType.includes("application/json")) {
          const info = await res.json();
          if (info.isSQLite) {
            fileData = {
              name: path.split('/').pop(),
              path,
              isSQLite: true,
              size: info.size,
              content: null
            };
          } else {
            fileData = {
              name: path.split('/').pop(),
              path,
              isBinary: true,
              mimeType: info.mimeType,
              size: info.size,
              content: null
            };
          }
        } else if (contentType.includes("image/")) {
          fileData = {
            name: path.split('/').pop(),
            path,
            isBinary: false,
            isImage: true,
            imageUrl: this.apiUrl(`/api/file?path=${encodeURIComponent(path)}`),
            content: null
          };
        } else if (contentType.includes("video/") || contentType.includes("audio/")) {
          fileData = {
            name: path.split('/').pop(),
            path,
            isBinary: false,
            isVideo: contentType.includes("video/"),
            isAudio: contentType.includes("audio/"),
            mediaUrl: this.apiUrl(`/api/file?path=${encodeURIComponent(path)}`),
            mimeType: contentType,
            content: null
          };
        } else {
          // Text
          const content = await res.text();
          const fileName = path.split('/').pop().toLowerCase();
          fileData = {
            name: path.split('/').pop(),
            path,
            isBinary: false,
            isImage: false,
            isCSV: fileName.endsWith('.csv') || fileName.endsWith('.tsv'),
            content
          };
        }
        
        this.openFiles.push(fileData);
      } catch (err) {
        console.error("Failed to load file content", path, err);
        return;
      }
    }

    this.activePath = path;
    this.syncFileUrl(path, jumpToLine);

    // If we need to jump to a line
    if (jumpToLine && window.editorInstance) {
      // Wait for Monaco to load content
      setTimeout(() => {
        if (window.editorInstance && typeof window.editorInstance.revealLineInCenter === 'function') {
          window.editorInstance.revealLineInCenter(jumpToLine);
          window.editorInstance.setPosition({ lineNumber: jumpToLine, column: 1 });
          window.editorInstance.focus();
        }
      }, 150);
    }
  }

  // uriToRelPath converts an LSP file:// URI into a workspace-relative path,
  // mirroring the codeEditorService patch in Editor.svelte.
  uriToRelPath(uri) {
    let p = uri.replace(/^file:\/\//, '');
    try { p = decodeURIComponent(p); } catch { /* leave as-is */ }
    const wsPath = this.workspace.path;
    if (wsPath && p.startsWith(wsPath)) p = p.slice(wsPath.length);
    if (p.startsWith('/')) p = p.slice(1);
    return p;
  }

  // beginReferences primes the references panel with a loading state for the
  // given symbol, so the sidebar can open immediately while the LSP query and
  // preview fetches complete.
  beginReferences(symbol) {
    this.references = { symbol: symbol || 'symbol', loading: true, error: null, groups: [] };
    this.sidebarTab = 'references';
  }

  clearReferences() {
    this.references = null;
  }

  // runExplain asks the backend AI provider to explain code (a whole file, or a
  // selected snippet) and streams the result into the AI sidebar panel.
  async runExplain(path, code) {
    const title = path || 'Selection';
    this.aiExplain = { title, loading: true, error: null, text: '' };
    this.sidebarTab = 'ai';
    try {
      const res = await fetch(this.apiUrl('/api/ai/explain'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, code })
      });
      if (!res.ok) {
        const msg = (await res.text()).trim() || `Request failed (${res.status})`;
        this.aiExplain = { title, loading: false, error: msg, text: '' };
        return;
      }
      const data = await res.json();
      this.aiExplain = { title, loading: false, error: null, text: data.explanation || '' };
    } catch (e) {
      this.aiExplain = { title, loading: false, error: e.message, text: '' };
    }
  }

  clearExplain() {
    this.aiExplain = null;
  }

  // syncFileUrl keeps ?file=&line= in the address bar so the current file and
  // position can be shared or restored on reload. Uses replaceState to avoid
  // polluting browser history on every navigation.
  syncFileUrl(path, line) {
    const url = new URL(window.location.href);
    if (path) {
      url.searchParams.set('file', path);
      if (line) url.searchParams.set('line', String(line));
      else url.searchParams.delete('line');
    } else {
      url.searchParams.delete('file');
      url.searchParams.delete('line');
    }
    window.history.replaceState({}, '', url);
  }

  // deepLink builds a shareable absolute URL to the active file and line.
  deepLink() {
    const url = new URL(window.location.href);
    if (this.currentWorkspaceId) url.searchParams.set('ws', this.currentWorkspaceId);
    if (this.activePath) {
      url.searchParams.set('file', this.activePath);
      if (this.cursorPos?.line) url.searchParams.set('line', String(this.cursorPos.line));
    }
    return url.toString();
  }

  // buildReferences turns raw LSP Location objects into the grouped, preview-rich
  // shape the ReferencesPanel renders. It fetches each referenced file once
  // (reusing already-open content when possible) to slice a one-line preview.
  async buildReferences(symbol, locations) {
    if (!locations || locations.length === 0) {
      this.references = { symbol: symbol || 'symbol', loading: false, error: null, groups: [] };
      return;
    }

    // Group locations by workspace-relative path, preserving line order.
    const byPath = new Map();
    for (const loc of locations) {
      if (!loc || !loc.uri || !loc.range) continue;
      const path = this.uriToRelPath(loc.uri);
      const line = loc.range.start.line + 1;
      const column = loc.range.start.character + 1;
      if (!byPath.has(path)) byPath.set(path, []);
      byPath.get(path).push({ line, column });
    }

    // Fetch each file's lines once so we can attach a source-line preview.
    const groups = [];
    for (const [path, items] of byPath) {
      let lines = null;
      const open = this.openFiles.find(f => f.path === path && typeof f.content === 'string');
      if (open) {
        lines = open.content.split('\n');
      } else {
        try {
          const res = await fetch(this.apiUrl(`/api/file?path=${encodeURIComponent(path)}`));
          if (res.ok && (res.headers.get('content-type') || '').indexOf('application/json') === -1) {
            lines = (await res.text()).split('\n');
          }
        } catch { /* preview is best-effort */ }
      }

      items.sort((a, b) => a.line - b.line);
      for (const it of items) {
        it.preview = lines && lines[it.line - 1] !== undefined ? lines[it.line - 1].trim() : '';
      }
      groups.push({ path, items });
    }

    groups.sort((a, b) => a.path.localeCompare(b.path));
    this.references = { symbol: symbol || 'symbol', loading: false, error: null, groups };
  }

  closeFile(path) {
    const idx = this.openFiles.findIndex(f => f.path === path);
    if (idx !== -1) {
      this.openFiles.splice(idx, 1);
      if (this.activePath === path) {
        if (this.openFiles.length > 0) {
          // Switch to adjacent tab
          const nextActiveIdx = Math.max(0, idx - 1);
          this.activePath = this.openFiles[nextActiveIdx].path;
        } else {
          this.activePath = null;
        }
      }
    }
  }

  closeOtherFiles(path) {
    this.openFiles = this.openFiles.filter(f => f.path === path);
    this.activePath = path;
  }

  closeFilesToTheRight(path) {
    const idx = this.openFiles.findIndex(f => f.path === path);
    if (idx !== -1) {
      this.openFiles = this.openFiles.slice(0, idx + 1);
      if (!this.openFiles.some(f => f.path === this.activePath)) {
        this.activePath = path;
      }
    }
  }

  closeAllFiles() {
    this.openFiles = [];
    this.activePath = null;
    this.activeDiff = null;
    this.syncFileUrl(null);
  }

  async search(query) {
    this.searchQuery = query;
    if (!query) {
      this.searchResults = [];
      return;
    }
    this.isSearching = true;
    try {
      const res = await fetch(this.apiUrl(`/api/search?q=${encodeURIComponent(query)}`));
      this.searchResults = await res.json();
    } catch (err) {
      console.error("Search failed", err);
    } finally {
      this.isSearching = false;
    }
  }

  async loadGit() {
    try {
      const res = await fetch(this.apiUrl(`/api/git/branches`));
      this.git = await res.json();
    } catch (err) {
      console.error("Failed to load git info", err);
      this.git = { isGit: false, currentBranch: '', branches: [] };
    }
  }

  async checkoutBranch(branch) {
    this.isCheckingOut = true;
    try {
      const res = await fetch(this.apiUrl(`/api/git/checkout?branch=${encodeURIComponent(branch)}`), {
        method: 'POST'
      });
      if (!res.ok) {
        const errText = await res.text();
        alert(`Failed to checkout branch: ${errText}`);
        return;
      }
      await this.reloadWorkspace();
    } catch (err) {
      console.error("Failed to checkout branch", err);
      alert(`Failed to checkout branch: ${err.message}`);
    } finally {
      this.isCheckingOut = false;
    }
  }

  async reloadWorkspace() {
    await this.loadGit();
    const pathsToReload = Array.from(this.expandedPaths);
    this.dirContents = {};
    for (const path of pathsToReload) {
      await this.loadDir(path);
    }
    const filesToKeep = [];
    for (const file of this.openFiles) {
      // Virtual tabs (commit details, commit graph, insights) aren't backed by a
      // file on disk — keep them as-is instead of trying to re-fetch and dropping them.
      if (file.isCommit || file.isGraph || file.isInsights || file.isOnboarding) {
        filesToKeep.push(file);
        continue;
      }
      try {
        const res = await fetch(this.apiUrl(`/api/file?path=${encodeURIComponent(file.path)}`));
        if (res.status === 404) {
          continue;
        }
        const contentType = res.headers.get("content-type") || "";
        if (contentType.includes("application/json")) {
          const info = await res.json();
          file.isBinary = true;
          file.mimeType = info.mimeType;
          file.size = info.size;
          file.content = null;
        } else if (contentType.includes("image/")) {
          file.isBinary = false;
          file.isImage = true;
          file.imageUrl = this.apiUrl(`/api/file?path=${encodeURIComponent(file.path)}&t=${Date.now()}`);
          file.content = null;
        } else if (contentType.includes("video/") || contentType.includes("audio/")) {
          file.isBinary = false;
          file.isVideo = contentType.includes("video/");
          file.isAudio = contentType.includes("audio/");
          file.mediaUrl = this.apiUrl(`/api/file?path=${encodeURIComponent(file.path)}&t=${Date.now()}`);
          file.mimeType = contentType;
          file.content = null;
        } else {
          const content = await res.text();
          file.isBinary = false;
          file.isImage = false;
          file.content = content;
        }
        filesToKeep.push(file);
      } catch (err) {
        console.error("Failed to reload file after branch switch", file.path, err);
      }
    }
    this.openFiles = filesToKeep;
    if (this.activePath && !this.openFiles.some(f => f.path === this.activePath)) {
      if (this.openFiles.length > 0) {
        this.activePath = this.openFiles[0].path;
      } else {
        this.activePath = null;
      }
    }
  }

  async openDiff(path, originalCommit, modifiedCommit, title) {
   try {
     let originalContent = '';
     if (originalCommit) {
       const res = await fetch(this.apiUrl(`/api/git/show?path=${encodeURIComponent(path)}&commit=${originalCommit}`));
       if (res.ok) {
         originalContent = await res.text();
       }
     }

     let modifiedContent = '';
     if (modifiedCommit) {
       const res = await fetch(this.apiUrl(`/api/git/show?path=${encodeURIComponent(path)}&commit=${modifiedCommit}`));
       if (res.ok) {
         modifiedContent = await res.text();
       }
     } else {
       const res = await fetch(this.apiUrl(`/api/file?path=${encodeURIComponent(path)}`));
       if (res.ok) {
         modifiedContent = await res.text();
       }
     }

     this.activeDiff = {
       path,
       originalContent,
       modifiedContent,
       title
     };
     this.activePath = null;
   } catch (err) {
     console.error("Failed to load diff contents", path, err);
   }
 }

 async openFileAtCommit(path, commitHash) {
   if (!path) return;
   const short = commitHash.slice(0, 7);
   const title = `${path.split('/').pop()} @ ${short}`;
   await this.openDiff(path, `${commitHash}~1`, commitHash, title);
 }

  openGraph() {
    this.activeDiff = null;
    const tabPath = 'graph:';
    const exists = this.openFiles.some(f => f.path === tabPath);
    if (!exists) {
      this.openFiles.push({
        name: 'Commit Graph',
        path: tabPath,
        isGraph: true
      });
    }
    this.activePath = tabPath;
  }

  openInsights() {
    this.activeDiff = null;
    const tabPath = 'insights:';
    const exists = this.openFiles.some(f => f.path === tabPath);
    if (!exists) {
      this.openFiles.push({
        name: 'Repo Insights',
        path: tabPath,
        isInsights: true
      });
    }
    this.activePath = tabPath;
  }

  openOnboarding() {
    this.activeDiff = null;
    const tabPath = 'onboarding:';
    const exists = this.openFiles.some(f => f.path === tabPath);
    if (!exists) {
      this.openFiles.push({
        name: 'Onboarding',
        path: tabPath,
        isOnboarding: true
      });
    }
    this.activePath = tabPath;
  }

  async openCommit(hash) {
    this.activeDiff = null;
    const tabPath = `commit:${hash}`;
    // Check if already open
    const exists = this.openFiles.some(f => f.path === tabPath);
    if (!exists) {
      try {
        const res = await fetch(this.apiUrl(`/api/git/commit?hash=${hash}`));
        if (res.ok) {
          const details = await res.json();
          const commitTab = {
            name: `Commit: ${hash.slice(0, 7)}`,
            path: tabPath,
            isCommit: true,
            commitInfo: details
          };
          this.openFiles.push(commitTab);
        }
      } catch (err) {
        console.error("Failed to load commit details", hash, err);
      }
    }
    this.activePath = tabPath;
  }
}

// Singleton store
export const store = new AppStore();
