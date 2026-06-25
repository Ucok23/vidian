<script>
  let { content = '' } = $props();

  let headers = $state([]);
  let rows = $state([]);

  function parseCSV(text) {
    const result = [];
    let current = '';
    let inQuotes = false;
    let row = [];

    for (let i = 0; i < text.length; i++) {
      const ch = text[i];
      const next = text[i + 1];

      if (inQuotes) {
        if (ch === '"' && next === '"') {
          current += '"';
          i++;
        } else if (ch === '"') {
          inQuotes = false;
        } else {
          current += ch;
        }
      } else {
        if (ch === '"') {
          inQuotes = true;
        } else if (ch === ',') {
          row.push(current);
          current = '';
        } else if (ch === '\n' || (ch === '\r' && next === '\n')) {
          row.push(current);
          current = '';
          if (row.length > 1 || row[0] !== '') result.push(row);
          row = [];
          if (ch === '\r') i++;
        } else {
          current += ch;
        }
      }
    }
    // Last field/row
    row.push(current);
    if (row.length > 1 || row[0] !== '') result.push(row);
    return result;
  }

  $effect(() => {
    const parsed = parseCSV(content || '');
    if (parsed.length > 0) {
      headers = parsed[0];
      rows = parsed.slice(1);
    } else {
      headers = [];
      rows = [];
    }
  });
</script>

<div class="csv-viewer">
  <div class="csv-toolbar">
    <span class="csv-label">CSV</span>
    <span class="csv-info">{rows.length} rows × {headers.length} columns</span>
  </div>
  <div class="csv-table-wrapper">
    <table>
      <thead>
        <tr>
          <th class="row-num">#</th>
          {#each headers as header, i}
            <th>{header}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each rows as row, ri}
          <tr>
            <td class="row-num">{ri + 1}</td>
            {#each headers as _, ci}
              <td>{row[ci] ?? ''}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

<style>
  .csv-viewer {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: #1e1e24;
    overflow: hidden;
  }

  .csv-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 16px;
    background-color: #141416;
    border-bottom: 1px solid #2d2d34;
    flex-shrink: 0;
  }

  .csv-label {
    background-color: #059669;
    color: #ffffff;
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
  }

  .csv-info {
    font-size: 12px;
    color: #8e8e93;
  }

  .csv-table-wrapper {
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
    border-bottom: 2px solid #6366f1;
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

  .csv-table-wrapper::-webkit-scrollbar {
    width: 10px;
    height: 10px;
  }
  .csv-table-wrapper::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 5px;
  }
  .csv-table-wrapper::-webkit-scrollbar-corner {
    background: #1e1e24;
  }
</style>
