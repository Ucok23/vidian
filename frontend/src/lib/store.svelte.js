import { tick } from 'svelte';

const API_BASE = ''; // Same host

class AppStore {
  workspace = $state({ name: '', path: '' });
  workspaces = $state([]);
  currentWorkspaceId = $state(null);
  openFiles = $state([]);
  activePath = $state(null);
  sidebarTab = $state('explorer'); // 'explorer', 'search'
  searchQuery = $state('');
  searchResults = $state([]);
  isSearching = $state(false);
  cursorPos = $state({ line: 1, column: 1 });
  activeLanguage = $state('Plain Text');
  currentLineBlame = $state(null);
  lineHistory = $state(null);    // { path, start, end, commits: [] }
  compareResult = $state(null);  // { ref1, ref2, files: [], stat: '' }
  
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
