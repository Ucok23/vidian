<script>
  import { onMount } from 'svelte';
  import { store } from './lib/store.svelte.js';
  import Icon from './lib/Icon.svelte';
  import Explorer from './lib/Explorer.svelte';
  import Search from './lib/Search.svelte';
  import Editor from './lib/Editor.svelte';
  import QuickOpen from './lib/QuickOpen.svelte';
  import GitPanel from './lib/GitPanel.svelte';

  let sidebarVisible = $state(true);
  let sidebarWidth = $state(260);
  let isResizing = $state(false);

  // Tab context menu state
  let tabContextMenu = $state({ visible: false, x: 0, y: 0, path: null });

  function showTabContextMenu(e, path) {
    e.preventDefault();
    tabContextMenu = { visible: true, x: e.clientX, y: e.clientY, path };
  }

  function hideTabContextMenu() {
    tabContextMenu = { visible: false, x: 0, y: 0, path: null };
  }

  function handleTabMenuAction(action) {
    const path = tabContextMenu.path;
    hideTabContextMenu();
    if (!path) return;

    switch (action) {
      case 'close':
        store.closeFile(path);
        break;
      case 'closeOthers':
        store.closeOtherFiles(path);
        break;
      case 'closeRight':
        store.closeFilesToTheRight(path);
        break;
      case 'closeAll':
        store.closeAllFiles();
        break;
      case 'copyPath':
        navigator.clipboard.writeText(path);
        break;
    }
  }

  // Resize logic
  function startResize(e) {
    e.preventDefault();
    isResizing = true;
    window.addEventListener('mousemove', handleResize);
    window.addEventListener('mouseup', stopResize);
  }

  function handleResize(e) {
    if (isResizing) {
      // Keep width within a reasonable range
      sidebarWidth = Math.max(160, Math.min(500, e.clientX - 48)); // 48px is the activity bar width
    }
  }

  function stopResize() {
    isResizing = false;
    window.removeEventListener('mousemove', handleResize);
    window.removeEventListener('mouseup', stopResize);
  }

  function toggleSidebarTab(tab) {
    if (store.sidebarTab === tab) {
      sidebarVisible = !sidebarVisible;
    } else {
      store.sidebarTab = tab;
      sidebarVisible = true;
    }
  }

  // Handle global shortcuts
  function handleKeyDown(e) {
    // Ctrl+B toggles sidebar
    if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
      e.preventDefault();
      sidebarVisible = !sidebarVisible;
    }

    // Ctrl+Shift+F focus search
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'F') {
      e.preventDefault();
      store.sidebarTab = 'search';
      sidebarVisible = true;
      // Focus search input
      setTimeout(() => {
        const input = document.querySelector('.search-box input');
        input?.focus();
      }, 50);
    }
  }

  function getFileColor(name) {
    const ext = name.split('.').pop().toLowerCase();
    const colors = {
      js: '#f7df1e',
      jsx: '#61dafb',
      ts: '#3178c6',
      tsx: '#61dafb',
      html: '#e34c26',
      css: '#264de4',
      json: '#cbcb41',
      md: '#0891b2',
      go: '#00add8',
      py: '#3572a5',
      rs: '#dea584',
      svelte: '#ff3e00'
    };
    return colors[ext] || '#9ca3af';
  }

  // Open git sidebar when line history is triggered from editor
  $effect(() => {
    if (store.lineHistory) {
      store.sidebarTab = 'git';
      sidebarVisible = true;
    }
  });

  onMount(async () => {
    await store.init();
    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('click', hideTabContextMenu);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('click', hideTabContextMenu);
    };
  });
</script>

<div class="app-container">
  <!-- Activity Bar (Leftmost thin bar) -->
  <div class="activity-bar">
    <div class="top-items">
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="activity-btn"
        class:active={sidebarVisible && store.sidebarTab === 'explorer'}
        onclick={() => toggleSidebarTab('explorer')}
        title="Explorer (Ctrl+B)"
      >
        <Icon name="files" size={24} />
      </div>

      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="activity-btn"
        class:active={sidebarVisible && store.sidebarTab === 'search'}
        onclick={() => toggleSidebarTab('search')}
        title="Search (Ctrl+Shift+F)"
      >
        <Icon name="search" size={24} />
      </div>

      {#if store.git.isGit}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="activity-btn"
          class:active={sidebarVisible && store.sidebarTab === 'git'}
          onclick={() => toggleSidebarTab('git')}
          title="Source Control"
        >
          <Icon name="branch" size={24} />
        </div>
      {/if}
    </div>

    <div class="bottom-items">
      <div class="activity-btn" title="About Vidian">
        <Icon name="info" size={22} color="#8e8e93" />
      </div>
      <div class="activity-btn" title="Settings">
        <Icon name="settings" size={22} color="#8e8e93" />
      </div>
    </div>
  </div>

  <!-- Sidebar Panel -->
  {#if sidebarVisible}
    <div class="sidebar" style="width: {sidebarWidth}px">
      {#if store.sidebarTab === 'explorer'}
        <Explorer />
      {:else if store.sidebarTab === 'search'}
        <Search />
      {:else if store.sidebarTab === 'git'}
        <GitPanel />
      {/if}
    </div>

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="resize-handle"
      class:resizing={isResizing}
      onmousedown={startResize}
    ></div>
  {/if}

  <!-- Main View Area -->
  <div class="main-area">
    <!-- Tabs Bar -->
    <div class="tabs-bar">
      {#each store.openFiles as file (file.path)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="tab"
          class:active={store.activePath === file.path}
          onclick={() => {
            if (file.isCommit || file.isGraph) {
              store.activeDiff = null;
              store.activePath = file.path;
            } else {
              store.openFile(file.path);
            }
          }}
          oncontextmenu={(e) => showTabContextMenu(e, file.path)}
          title={file.path}
        >
          {#if file.isGraph}
            <Icon name="branch" size={14} color="#6366f1" />
          {:else if file.isCommit}
            <Icon name="gitCommit" size={14} color="#fbbf24" />
          {:else}
            <Icon name="file" size={14} color={getFileColor(file.name)} />
          {/if}
          <span class="tab-name">{file.name}</span>
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <span
            class="close-icon"
            onclick={(e) => {
              e.stopPropagation();
              store.closeFile(file.path);
            }}
          >
            <Icon name="close" size={12} />
          </span>
        </div>
      {/each}
      {#if store.activeDiff}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="tab active" onclick={() => {}} title={store.activeDiff.path}>
          <Icon name="split" size={14} color="#6366f1" />
          <span class="tab-name">{store.activeDiff.title}</span>
          <span
            class="close-icon"
            onclick={(e) => {
              e.stopPropagation();
              store.activeDiff = null;
              if (store.openFiles.length > 0) {
                store.openFile(store.openFiles[0].path);
              }
            }}
          >
            <Icon name="close" size={12} />
          </span>
        </div>
      {/if}
    </div>

    <!-- Tab Context Menu -->
    {#if tabContextMenu.visible}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <div
        class="tab-context-menu"
        style="left: {tabContextMenu.x}px; top: {tabContextMenu.y}px"
        onclick={(e) => e.stopPropagation()}
      >
        <div class="context-menu-item" onclick={() => handleTabMenuAction('close')}>Close</div>
        <div class="context-menu-item" onclick={() => handleTabMenuAction('closeOthers')}>Close Others</div>
        <div class="context-menu-item" onclick={() => handleTabMenuAction('closeRight')}>Close to the Right</div>
        <div class="context-menu-separator"></div>
        <div class="context-menu-item" onclick={() => handleTabMenuAction('closeAll')}>Close All</div>
        <div class="context-menu-separator"></div>
        <div class="context-menu-item" onclick={() => handleTabMenuAction('copyPath')}>Copy Path</div>
      </div>
    {/if}

    <!-- Editor Area -->
    <div class="editor-container">
      <Editor />
    </div>

    <!-- Status Bar -->
    <div class="status-bar">
      <div class="status-left">
        <div class="status-item bg-indigo">
          <Icon name="folder" size={12} color="#ffffff" />
          <span>{store.workspace.name || 'loading...'}</span>
        </div>
        <div class="status-item clickable" onclick={() => store.init()}>
          <Icon name="refresh" size={12} />
          <span>Synchronized</span>
        </div>
        {#if store.activePath}
          <div class="status-item text-dimmed">
            <span>{store.activePath}</span>
          </div>
        {/if}
      </div>

      <div class="status-right">
        {#if store.currentLineBlame && store.git.isGit && store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage}
          <div class="status-item status-blame" title="{store.currentLineBlame.commit.slice(0, 8)} — {store.currentLineBlame.summary}">
            <Icon name="gitCommit" size={11} />
            <span>{store.currentLineBlame.author}, {store.currentLineBlame.date} • {store.currentLineBlame.summary.length > 40 ? store.currentLineBlame.summary.slice(0, 39) + '…' : store.currentLineBlame.summary}</span>
          </div>
        {/if}
        {#if store.activePath && !store.activeFile?.isBinary && !store.activeFile?.isImage}
          <div class="status-item">
            <span>Ln {store.cursorPos.line}, Col {store.cursorPos.column}</span>
          </div>
        {/if}
        <div class="status-item">
          <span>UTF-8</span>
        </div>
        <div class="status-item">
          <span>{store.activeLanguage}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- Quick Open Palette -->
  <QuickOpen />
</div>

<style>
  .app-container {
    display: flex;
    width: 100vw;
    height: 100vh;
    background-color: #121214;
    overflow: hidden;
  }

  /* Activity Bar Styling */
  .activity-bar {
    width: 48px;
    height: 100%;
    background-color: #16161a;
    border-right: 1px solid #2d2d34;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
    flex-shrink: 0;
  }

  .activity-btn {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    position: relative;
    color: #8e8e93;
    transition: color 0.15s;
  }

  .activity-btn:hover {
    color: #e3e3e6;
  }

  .activity-btn.active {
    color: #6366f1;
  }

  .activity-btn.active::before {
    content: '';
    position: absolute;
    left: 0;
    top: 6px;
    bottom: 6px;
    width: 3px;
    background-color: #6366f1;
    border-radius: 0 4px 4px 0;
  }

  /* Sidebar Styling */
  .sidebar {
    height: 100%;
    background-color: #1b1b20;
    border-right: 1px solid #2d2d34;
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    overflow: hidden;
  }

  /* Resize Handle */
  .resize-handle {
    width: 4px;
    height: 100%;
    cursor: col-resize;
    background-color: transparent;
    transition: background-color 0.2s;
    z-index: 10;
    margin-left: -2px;
    margin-right: -2px;
    flex-shrink: 0;
  }

  .resize-handle:hover, .resize-handle.resizing {
    background-color: #6366f1;
  }

  /* Main View Area Styling */
  .main-area {
    display: flex;
    flex-direction: column;
    flex: 1;
    height: 100%;
    overflow: hidden;
    background-color: #1e1e24;
  }

  /* Tabs Bar Styling */
  .tabs-bar {
    height: 35px;
    background-color: #141416;
    border-bottom: 1px solid #2d2d34;
    display: flex;
    overflow-x: auto;
    overflow-y: hidden;
    flex-shrink: 0;
  }

  .tabs-bar::-webkit-scrollbar {
    height: 4px;
  }
  .tabs-bar::-webkit-scrollbar-track {
    background: transparent;
  }
  .tabs-bar::-webkit-scrollbar-thumb {
    background: #3d3d50;
    border-radius: 9999px;
  }
  .tabs-bar::-webkit-scrollbar-thumb:hover {
    background: #6366f1;
  }

  .tab {
    display: flex;
    align-items: center;
    padding: 0 16px;
    height: 100%;
    border-right: 1px solid #2d2d34;
    background-color: #141416;
    cursor: pointer;
    font-size: 13px;
    color: #8e8e93;
    gap: 8px;
    user-select: none;
    transition: background-color 0.15s, color 0.15s;
    min-width: 120px;
    max-width: 200px;
    position: relative;
  }

  .tab:hover {
    background-color: rgba(255, 255, 255, 0.02);
    color: #e3e3e6;
  }

  .tab.active {
    background-color: #1e1e24;
    color: #ffffff;
  }

  .tab.active::after {
    content: '';
    position: absolute;
    bottom: -1px;
    left: 0;
    right: 0;
    height: 2px;
    background-color: #6366f1;
  }

  .tab-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
  }

  .close-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border-radius: 4px;
    color: transparent;
    transition: color 0.1s, background-color 0.1s;
  }

  .tab:hover .close-icon, .tab.active .close-icon {
    color: #8e8e93;
  }

  .close-icon:hover {
    background-color: rgba(255, 255, 255, 0.1);
    color: #ffffff !important;
  }

  /* Editor Container */
  .editor-container {
    flex: 1;
    overflow: hidden;
    position: relative;
  }

  /* Status Bar Styling */
  .status-bar {
    height: 22px;
    background-color: #16161a;
    border-top: 1px solid #2d2d34;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 8px;
    font-size: 11px;
    color: #8e8e93;
    user-select: none;
    flex-shrink: 0;
  }

  .status-left, .status-right {
    display: flex;
    align-items: center;
    height: 100%;
  }

  .status-item {
    display: flex;
    align-items: center;
    height: 100%;
    padding: 0 8px;
    gap: 5px;
  }

  .status-item.clickable {
    cursor: pointer;
  }
  .status-item.clickable:hover {
    background-color: rgba(255, 255, 255, 0.05);
    color: #ffffff;
  }

  .bg-indigo {
    background-color: #6366f1;
    color: #ffffff;
    font-weight: 500;
  }

  .text-dimmed {
    color: #5d5d66;
  }

  .status-blame {
    max-width: 380px;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    color: #8e8e93;
    font-size: 11px;
    opacity: 0.85;
  }

  /* Tab Context Menu */
  .tab-context-menu {
    position: fixed;
    z-index: 1000;
    background-color: #252529;
    border: 1px solid #3d3d44;
    border-radius: 6px;
    padding: 4px 0;
    min-width: 180px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  }

  .context-menu-item {
    padding: 6px 16px;
    font-size: 13px;
    color: #e3e3e6;
    cursor: pointer;
    user-select: none;
  }

  .context-menu-item:hover {
    background-color: #6366f1;
    color: #ffffff;
  }

  .context-menu-separator {
    height: 1px;
    background-color: #3d3d44;
    margin: 4px 0;
  }
</style>
