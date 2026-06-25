<script>
  import { onMount } from 'svelte';
  import Icon from './Icon.svelte';

  let { path = '' } = $props();

  let tables = $state([]);
  let activeTable = $state(null);
  let columns = $state([]);
  let rows = $state([]);
  let totalRows = $state(0);
  let currentOffset = $state(0);
  let loading = $state(false);
  let error = $state(null);

  const PAGE_SIZE = 100;

  onMount(async () => {
    await loadTables();
  });

  async function loadTables() {
    try {
      const res = await fetch(`/api/sqlite/tables?path=${encodeURIComponent(path)}`);
      if (!res.ok) throw new Error(await res.text());
      tables = await res.json();
      if (tables && tables.length > 0) {
        await selectTable(tables[0]);
      }
    } catch (err) {
      error = err.message;
    }
  }

  async function selectTable(name) {
    activeTable = name;
    currentOffset = 0;
    await loadData();
  }

  async function loadData() {
    if (!activeTable) return;
    loading = true;
    error = null;
    try {
      const res = await fetch(
        `/api/sqlite/query?path=${encodeURIComponent(path)}&table=${encodeURIComponent(activeTable)}&limit=${PAGE_SIZE}&offset=${currentOffset}`
      );
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      columns = data.columns || [];
      rows = data.rows || [];
      totalRows = data.totalRows || 0;
    } catch (err) {
      error = err.message;
    } finally {
      loading = false;
    }
  }

  function prevPage() {
    if (currentOffset > 0) {
      currentOffset = Math.max(0, currentOffset - PAGE_SIZE);
      loadData();
    }
  }

  function nextPage() {
    if (currentOffset + PAGE_SIZE < totalRows) {
      currentOffset += PAGE_SIZE;
      loadData();
    }
  }

  const currentPage = $derived(Math.floor(currentOffset / PAGE_SIZE) + 1);
  const totalPages = $derived(Math.ceil(totalRows / PAGE_SIZE));
</script>

<div class="sqlite-viewer">
  <div class="sqlite-sidebar">
    <div class="sidebar-header">
      <span class="sqlite-badge">SQLite</span>
      <span class="table-count">{tables?.length || 0} tables</span>
    </div>
    <div class="table-list">
      {#each tables || [] as table}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="table-item"
          class:active={activeTable === table}
          onclick={() => selectTable(table)}
        >
          <Icon name="file" size={14} color="#059669" />
          <span>{table}</span>
        </div>
      {/each}
    </div>
  </div>

  <div class="sqlite-main">
    {#if error}
      <div class="sqlite-error">{error}</div>
    {:else if activeTable}
      <div class="data-toolbar">
        <span class="table-name">{activeTable}</span>
        <span class="row-info">{totalRows} rows</span>
        <div class="pagination">
          <button onclick={prevPage} disabled={currentOffset === 0}>← Prev</button>
          <span class="page-info">Page {currentPage} / {totalPages}</span>
          <button onclick={nextPage} disabled={currentOffset + PAGE_SIZE >= totalRows}>Next →</button>
        </div>
      </div>

      <div class="data-table-wrapper">
        {#if loading}
          <div class="loading">Loading...</div>
        {:else}
          <table>
            <thead>
              <tr>
                <th class="row-num">#</th>
                {#each columns as col}
                  <th>{col}</th>
                {/each}
              </tr>
            </thead>
            <tbody>
              {#each rows as row, ri}
                <tr>
                  <td class="row-num">{currentOffset + ri + 1}</td>
                  {#each row as cell}
                    <td>{cell === null ? 'NULL' : cell}</td>
                  {/each}
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    {:else}
      <div class="empty-state">No tables found in this database.</div>
    {/if}
  </div>
</div>

<style>
  .sqlite-viewer {
    width: 100%;
    height: 100%;
    display: flex;
    background-color: #1e1e24;
    overflow: hidden;
  }

  .sqlite-sidebar {
    width: 200px;
    flex-shrink: 0;
    background-color: #16161a;
    border-right: 1px solid #2d2d34;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .sidebar-header {
    padding: 10px 12px;
    border-bottom: 1px solid #2d2d34;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .sqlite-badge {
    background-color: #3b82f6;
    color: #ffffff;
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
  }

  .table-count {
    font-size: 11px;
    color: #8e8e93;
  }

  .table-list {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
  }

  .table-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    font-size: 13px;
    color: #c9d1d9;
    cursor: pointer;
    user-select: none;
  }

  .table-item:hover {
    background-color: rgba(255, 255, 255, 0.04);
  }

  .table-item.active {
    background-color: rgba(99, 102, 241, 0.15);
    color: #ffffff;
  }

  .sqlite-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .data-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 16px;
    background-color: #141416;
    border-bottom: 1px solid #2d2d34;
    flex-shrink: 0;
  }

  .table-name {
    font-size: 13px;
    font-weight: 600;
    color: #e3e3e6;
  }

  .row-info {
    font-size: 12px;
    color: #8e8e93;
  }

  .pagination {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .pagination button {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid #3d3d44;
    color: #e3e3e6;
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
  }

  .pagination button:hover:not(:disabled) {
    background: rgba(99, 102, 241, 0.2);
    border-color: #6366f1;
  }

  .pagination button:disabled {
    opacity: 0.3;
    cursor: default;
  }

  .page-info {
    font-size: 12px;
    color: #8e8e93;
  }

  .data-table-wrapper {
    flex: 1;
    overflow: auto;
  }

  table {
    width: max-content;
    min-width: 100%;
    border-collapse: collapse;
    font-size: 13px;
    font-family: 'Fira Code', monospace;
  }

  thead {
    position: sticky;
    top: 0;
    z-index: 2;
  }

  th {
    background-color: #16161a;
    color: #e3e3e6;
    font-weight: 600;
    text-align: left;
    padding: 8px 16px;
    border-bottom: 2px solid #3b82f6;
    white-space: nowrap;
  }

  td {
    padding: 6px 16px;
    color: #c9d1d9;
    border-bottom: 1px solid #2d2d34;
    white-space: nowrap;
    max-width: 400px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  tr:hover td {
    background-color: rgba(99, 102, 241, 0.08);
  }

  .row-num {
    color: #5d5d66;
    text-align: right;
    font-size: 11px;
    padding-right: 12px;
    min-width: 40px;
    user-select: none;
    background-color: #16161a;
    border-right: 1px solid #2d2d34;
    position: sticky;
    left: 0;
    z-index: 1;
  }

  thead .row-num {
    z-index: 3;
  }

  .sqlite-error {
    padding: 24px;
    color: #ef4444;
    font-size: 14px;
  }

  .loading, .empty-state {
    padding: 24px;
    color: #8e8e93;
    font-size: 14px;
    text-align: center;
  }

  .data-table-wrapper::-webkit-scrollbar {
    width: 10px;
    height: 10px;
  }
  .data-table-wrapper::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 5px;
  }
  .data-table-wrapper::-webkit-scrollbar-corner {
    background: #1e1e24;
  }
</style>
