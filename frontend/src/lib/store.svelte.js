import { tick } from 'svelte';

const API_BASE = ''; // Same host

class AppStore {
  workspace = $state({ name: '', path: '' });
  openFiles = $state([]);
  activePath = $state(null);
  sidebarTab = $state('explorer'); // 'explorer', 'search'
  searchQuery = $state('');
  searchResults = $state([]);
  isSearching = $state(false);
  cursorPos = $state({ line: 1, column: 1 });
  activeLanguage = $state('Plain Text');
  
  // File tree expanded paths
  expandedPaths = $state(new Set());
  // Cache of directory contents: path -> list of FileInfo
  dirContents = $state({});

  // Git state
  git = $state({ isGit: false, currentBranch: '', branches: [] });
  branchSelectorVisible = $state(false);
  isCheckingOut = $state(false);

  // Active file derived state
  get activeFile() {
    return this.openFiles.find(f => f.path === this.activePath) || null;
  }

  async init() {
    try {
      const res = await fetch(`${API_BASE}/api/workspace`);
      this.workspace = await res.json();
      // Load root directory
      await this.loadDir('');
      this.expandedPaths.add('');
      this.expandedPaths = new Set(this.expandedPaths);
      // Load Git info
      await this.loadGit();
    } catch (err) {
      console.error("Failed to load workspace info", err);
    }
  }

  async loadDir(path) {
    try {
      const res = await fetch(`${API_BASE}/api/dir?path=${encodeURIComponent(path)}`);
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
    // Check if already open
    const exists = this.openFiles.some(f => f.path === path);
    if (!exists) {
      // Fetch content
      try {
        const res = await fetch(`${API_BASE}/api/file?path=${encodeURIComponent(path)}`);
        const contentType = res.headers.get("content-type") || "";
        
        let fileData;
        if (contentType.includes("application/json")) {
          // It's binary info returned as JSON
          const info = await res.json();
          fileData = {
            name: path.split('/').pop(),
            path,
            isBinary: true,
            mimeType: info.mimeType,
            size: info.size,
            content: null
          };
        } else if (contentType.includes("image/")) {
          // Serve image URL directly from backend
          fileData = {
            name: path.split('/').pop(),
            path,
            isBinary: false,
            isImage: true,
            imageUrl: `${API_BASE}/api/file?path=${encodeURIComponent(path)}`,
            content: null
          };
        } else {
          // Text
          const content = await res.text();
          fileData = {
            name: path.split('/').pop(),
            path,
            isBinary: false,
            isImage: false,
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

  async search(query) {
    this.searchQuery = query;
    if (!query) {
      this.searchResults = [];
      return;
    }
    this.isSearching = true;
    try {
      const res = await fetch(`${API_BASE}/api/search?q=${encodeURIComponent(query)}`);
      this.searchResults = await res.json();
    } catch (err) {
      console.error("Search failed", err);
    } finally {
      this.isSearching = false;
    }
  }

  async loadGit() {
    try {
      const res = await fetch(`${API_BASE}/api/git/branches`);
      this.git = await res.json();
    } catch (err) {
      console.error("Failed to load git info", err);
      this.git = { isGit: false, currentBranch: '', branches: [] };
    }
  }

  async checkoutBranch(branch) {
    this.isCheckingOut = true;
    try {
      const res = await fetch(`${API_BASE}/api/git/checkout?branch=${encodeURIComponent(branch)}`, {
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
      this.branchSelectorVisible = false;
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
      try {
        const res = await fetch(`${API_BASE}/api/file?path=${encodeURIComponent(file.path)}`);
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
          file.imageUrl = `${API_BASE}/api/file?path=${encodeURIComponent(file.path)}&t=${Date.now()}`;
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
}

// Singleton store
export const store = new AppStore();
