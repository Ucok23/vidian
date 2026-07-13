import monaco from './monaco.js';
import { pathToFileUri, workspaceFileUri } from './paths.js';
import { store } from './store.svelte.js';

let ws = null;
let activeLang = null; // 'go', 'python', 'typescript', 'rust'
let activeWs = null;   // workspace ID the socket is scoped to
let requestId = 0;
const pendingRequests = new Map();
let isInitialized = false;
let lspStatusWs = null; // workspace id the cached status in store.lspStatus is for

// ensureLspStatus fetches per-language server availability once per workspace
// and caches it on the store. Used to skip connecting to servers that aren't
// installed and to show install hints in the UI.
async function ensureLspStatus() {
  const wsId = store.currentWorkspaceId;
  if (lspStatusWs === wsId && Object.keys(store.lspStatus || {}).length) {
    return store.lspStatus;
  }
  try {
    const url = `/api/lsp/status${wsId ? `?ws=${encodeURIComponent(wsId)}` : ''}`;
    const res = await fetch(url);
    store.lspStatus = await res.json();
    lspStatusWs = wsId;
  } catch (err) {
    console.error('Failed to fetch LSP status', err);
  }
  return store.lspStatus;
}

// installLsp asks the backend to install the language server for `lang`,
// streaming its output to onChunk as it runs. On success it clears the missing
// -server notice and reconnects LSP for the file currently in focus. The
// command is chosen server-side; we only name the language. Returns true if the
// server is detected afterwards.
export async function installLsp(lang, onChunk) {
  const wsId = store.currentWorkspaceId;
  const url = `/api/lsp/install${wsId ? `?ws=${encodeURIComponent(wsId)}` : ''}`;
  let res;
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ lang })
    });
  } catch (err) {
    onChunk(`Failed to start install: ${err}\n`);
    return false;
  }
  if (!res.body) {
    onChunk(`Install failed: ${res.status} ${res.statusText}\n`);
    return false;
  }
  const reader = res.body.getReader();
  const dec = new TextDecoder();
  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    onChunk(dec.decode(value, { stream: true }));
  }

  // The status cache is now stale — force a refresh and see if the server showed up.
  lspStatusWs = null;
  const status = await ensureLspStatus();
  const available = status?.[lang]?.available === true;
  if (available) {
    store.lspIssue = null;
    const fname = (store.activePath || '').split('/').pop();
    if (fname && store.workspace?.path) {
      await initLsp(store.workspace.path, fname);
    }
  }
  return available;
}

// List of file extension to language ID mapping
const extMapping = {
  go: 'go',
  py: 'python',
  ts: 'typescript',
  js: 'javascript',
  jsx: 'javascript',
  tsx: 'typescript',
  rs: 'rust',
  c: 'c',
  h: 'c',
  cpp: 'cpp',
  cc: 'cpp',
  cxx: 'cpp',
  hpp: 'cpp',
  lua: 'lua',
  rb: 'ruby'
};

// Languages that have a backend language server wired up in internal/lsp.
const supportedLangs = ['go', 'python', 'typescript', 'javascript', 'rust', 'c', 'cpp', 'lua', 'ruby'];

export async function initLsp(workspacePath, filename) {
  if (!filename) return;
  const ext = filename.split('.').pop().toLowerCase();
  const lspLang = extMapping[ext];

  // Only connect for supported languages. A file that needs no server clears
  // any lingering "server missing" notice.
  if (!lspLang || !supportedLangs.includes(lspLang)) {
    if (ws) {
      ws.close();
      ws = null;
      activeLang = null;
      isInitialized = false;
    }
    store.lspIssue = null;
    return;
  }

  // Gate on availability: if the server for this language isn't installed,
  // surface the install hint and skip the doomed connection. Code intelligence
  // is an enhancement — the editor works fine without it.
  const status = await ensureLspStatus();
  const langStatus = status?.[lspLang];
  if (langStatus && langStatus.available === false) {
    if (ws) {
      ws.close();
      ws = null;
      activeLang = null;
      isInitialized = false;
    }
    store.lspIssue = { lang: lspLang, install: langStatus.install || '', canInstall: !!langStatus.canInstall };
    return;
  }
  store.lspIssue = null;

  // If already connected for the same language AND workspace, do nothing.
  // A workspace switch must force a reconnect so the server scopes the language
  // server to the new directory.
  if (ws && activeLang === lspLang && activeWs === store.currentWorkspaceId && ws.readyState === WebSocket.OPEN) {
    return;
  }

  // Close existing connection
  if (ws) {
    ws.close();
    isInitialized = false;
    if (typeof window !== 'undefined') window._vidianLspReady = false;
    pendingRequests.clear();
  }

  activeLang = lspLang;
  activeWs = store.currentWorkspaceId;
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsParam = store.currentWorkspaceId ? `&ws=${encodeURIComponent(store.currentWorkspaceId)}` : '';
  const url = `${protocol}//${window.location.host}/api/lsp?lang=${lspLang}${wsParam}`;
  ws = new WebSocket(url);

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      if (msg.id !== undefined) {
        const pending = pendingRequests.get(msg.id);
        if (pending) {
          pendingRequests.delete(msg.id);
          if (msg.error) {
            pending.reject(msg.error);
          } else {
            pending.resolve(msg.result);
          }
        }
      } else {
        // Notifications (like diagnostics)
        if (msg.method === 'textDocument/publishDiagnostics') {
          handleDiagnostics(msg.params);
        } else if (msg.method === 'vidian/serverUnavailable') {
          // Backstop for the availability gate (e.g. the binary vanished after
          // the status check): surface the install hint instead of dying quietly.
          store.lspIssue = { lang: msg.params?.lang || activeLang, install: msg.params?.install || '' };
        } else if (msg.method === 'window/showMessage' && msg.params?.type === 1) {
          store.lspIssue = { lang: activeLang, install: '', message: msg.params?.message };
        }
      }
    } catch (err) {
      console.error('Failed to parse LSP message', err);
    }
  };

  ws.onopen = async () => {
    try {
      // Send initialize request to language server
      const initResult = await sendRequest('initialize', {
        processId: null,
        rootUri: pathToFileUri(workspacePath),
        capabilities: {
          textDocument: {
            hover: { contentFormat: ['markdown', 'plaintext'] },
            definition: { dynamicRegistration: true },
            references: { dynamicRegistration: true },
            documentSymbol: { hierarchicalDocumentSymbolSupport: true },
            callHierarchy: { dynamicRegistration: true }
          }
        }
      });

      sendNotification('initialized', {});
      isInitialized = true;
      store.lspIssue = null; // server is up — clear any stale "missing" notice
      // Readiness flag for tests/tooling to await before exercising LSP features.
      if (typeof window !== 'undefined') window._vidianLspReady = true;
      console.log(`LSP (${lspLang}) initialized successfully`, initResult);

      // Notify listeners (e.g. the references CodeLens provider) that symbol
      // and reference queries will now succeed, so they can refresh.
      if (typeof window !== 'undefined' && Array.isArray(window._vidianLspReadyListeners)) {
        window._vidianLspReadyListeners.forEach(fn => { try { fn(); } catch { /* ignore */ } });
      }

      // Trigger didOpen for the currently active file
      if (store.activeFile) {
        sendDidOpen(store.activeFile.path, store.activeFile.content, store.activeFile.name);
      }
    } catch (err) {
      console.error(`LSP (${lspLang}) initialization failed`, err);
    }
  };

  ws.onerror = (err) => {
    console.error(`LSP (${lspLang}) WebSocket error`, err);
  };
}

export function sendRequest(method, params) {
  return new Promise((resolve, reject) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      reject(new Error('LSP Connection is not open'));
      return;
    }
    const id = ++requestId;
    pendingRequests.set(id, { resolve, reject });
    ws.send(JSON.stringify({
      jsonrpc: '2.0',
      id,
      method,
      params
    }));
  });
}

export function sendNotification(method, params) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify({
    jsonrpc: '2.0',
    method,
    params
  }));
}

export function sendDidOpen(path, content, filename) {
  if (!isInitialized || !filename) return;
  const ext = filename.split('.').pop().toLowerCase();
  const languageId = extMapping[ext] || 'plaintext';

  sendNotification('textDocument/didOpen', {
    textDocument: {
      uri: workspaceFileUri(store.workspace.path, path),
      languageId: languageId,
      version: 1,
      text: content
    }
  });
}

function handleDiagnostics(params) {
  const { uri, diagnostics } = params;
  
  const model = monaco.editor.getModel(monaco.Uri.parse(uri));
  if (model) {
    const markers = diagnostics.map(d => ({
      severity: d.severity === 1 ? monaco.MarkerSeverity.Error : monaco.MarkerSeverity.Warning,
      message: d.message,
      startLineNumber: d.range.start.line + 1,
      startColumn: d.range.start.character + 1,
      endLineNumber: d.range.end.line + 1,
      endColumn: d.range.end.character + 1
    }));
    monaco.editor.setModelMarkers(model, 'lsp', markers);
  }
}

// findReferences asks the language server for every use of the symbol under
// `position` (declaration included). Returns raw LSP Location objects
// ({ uri, range }), or [] if the server can't answer. The caller is
// responsible for turning URIs into workspace-relative paths and previews.
export async function findReferences(model, position) {
  if (!isInitialized) return [];
  try {
    const res = await sendRequest('textDocument/references', {
      textDocument: { uri: model.uri.toString() },
      position: { line: position.lineNumber - 1, character: position.column - 1 },
      context: { includeDeclaration: true }
    });
    if (!res) return [];
    return Array.isArray(res) ? res : [res];
  } catch (err) {
    console.error('LSP references failed', err);
    return [];
  }
}

// documentSymbols asks the language server for the symbol tree (functions,
// types, methods, …) of the file backing `model`. Returns [] if unavailable.
// The result may be either a flat SymbolInformation[] or a hierarchical
// DocumentSymbol[]; the caller normalizes both shapes.
export async function documentSymbols(model) {
  if (!isInitialized) return [];
  try {
    const res = await sendRequest('textDocument/documentSymbol', {
      textDocument: { uri: model.uri.toString() }
    });
    return Array.isArray(res) ? res : [];
  } catch (err) {
    console.error('LSP documentSymbol failed', err);
    return [];
  }
}

// prepareCallHierarchy resolves the call-hierarchy anchor item(s) for the
// symbol at `position`. These CallHierarchyItem handles are what incoming/
// outgoingCalls operate on. Returns [] if the server can't answer.
export async function prepareCallHierarchy(model, position) {
  if (!isInitialized) return [];
  try {
    const res = await sendRequest('textDocument/prepareCallHierarchy', {
      textDocument: { uri: model.uri.toString() },
      position: { line: position.lineNumber - 1, character: position.column - 1 }
    });
    return Array.isArray(res) ? res : res ? [res] : [];
  } catch (err) {
    console.error('LSP prepareCallHierarchy failed', err);
    return [];
  }
}

// incomingCalls returns the callers of a CallHierarchyItem. Each result is
// { from: CallHierarchyItem, fromRanges: Range[] }: `from` is the calling
// symbol, `fromRanges` the individual call sites within it. Returns [] if
// unavailable.
export async function incomingCalls(item) {
  if (!isInitialized) return [];
  try {
    const res = await sendRequest('callHierarchy/incomingCalls', { item });
    return Array.isArray(res) ? res : [];
  } catch (err) {
    console.error('LSP incomingCalls failed', err);
    return [];
  }
}

// outgoingCalls returns the callees of a CallHierarchyItem. Each result is
// { to: CallHierarchyItem, fromRanges: Range[] }. Returns [] if unavailable.
export async function outgoingCalls(item) {
  if (!isInitialized) return [];
  try {
    const res = await sendRequest('callHierarchy/outgoingCalls', { item });
    return Array.isArray(res) ? res : [];
  } catch (err) {
    console.error('LSP outgoingCalls failed', err);
    return [];
  }
}

// lspReady reports whether an LSP session is initialized for the active
// language, so UI can avoid offering LSP-only affordances otherwise.
export function lspReady() {
  return isInitialized;
}

// Register Monaco providers for all supported languages
export function registerLspProviders() {
  supportedLangs.forEach(lang => {
    monaco.languages.registerHoverProvider(lang, {
      async provideHover(model, position) {
        if (!isInitialized || activeLang !== lang) return null;
        try {
          const res = await sendRequest('textDocument/hover', {
            textDocument: { uri: model.uri.toString() },
            position: { line: position.lineNumber - 1, character: position.column - 1 }
          });
          if (res && res.contents) {
            let value = '';
            if (typeof res.contents === 'string') {
              value = res.contents;
            } else if (res.contents.value) {
              value = res.contents.value;
            } else if (Array.isArray(res.contents)) {
              value = res.contents.map(c => typeof c === 'string' ? c : c.value).join('\n\n');
            }
            return {
              contents: [{ value }]
            };
          }
        } catch (err) {
          console.error(`LSP Hover failed for ${lang}`, err);
        }
        return null;
      }
    });

    monaco.languages.registerDefinitionProvider(lang, {
      async provideDefinition(model, position) {
        if (!isInitialized || activeLang !== lang) return null;
        try {
          const res = await sendRequest('textDocument/definition', {
            textDocument: { uri: model.uri.toString() },
            position: { line: position.lineNumber - 1, character: position.column - 1 }
          });
          if (res) {
            const locations = Array.isArray(res) ? res : [res];
            return locations.map(loc => ({
              uri: monaco.Uri.parse(loc.uri),
              range: {
                startLineNumber: loc.range.start.line + 1,
                startColumn: loc.range.start.character + 1,
                endLineNumber: loc.range.end.line + 1,
                endColumn: loc.range.end.character + 1
              }
            }));
          }
        } catch (err) {
          console.error(`LSP Definition failed for ${lang}`, err);
        }
        return null;
      }
    });
  });
}
