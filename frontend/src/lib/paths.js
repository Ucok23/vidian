// Path/URI helpers that work for both POSIX and Windows workspaces.
//
// The backend hands us `workspace.path` as a native absolute path — POSIX
// (`/home/u/proj`) or Windows (`C:\Users\u\proj`) — while file paths *within* a
// workspace always use forward slashes. Language servers speak `file://` URIs.
// These helpers bridge the two consistently so Monaco models, `didOpen`
// notifications, and diagnostics all agree on exactly one URI per file.
//
// The pre-Windows code built `file://${workspacePath}` directly, which produced
// the malformed `file://C:\Users\...` on Windows (backslashes, no leading slash
// before the drive) and broke every LSP feature. These helpers produce the
// standard `file:///C:/Users/...` form instead.

// joinWorkspacePath builds a normalized absolute path (forward slashes, no
// trailing slash) from a native workspace root and an optional
// workspace-relative path.
export function joinWorkspacePath(workspacePath, relPath = '') {
  const base = String(workspacePath || '').replace(/\\/g, '/').replace(/\/+$/, '');
  const rel = String(relPath || '').replace(/\\/g, '/').replace(/^\/+/, '');
  return rel ? `${base}/${rel}` : base;
}

// pathToFileUri converts a native absolute path (POSIX or Windows) into a
// `file://` URI. Windows drive paths (`C:/...`) gain the required leading slash
// (`/C:/...`); each path component is percent-encoded (so the drive colon
// becomes `%3A`, matching what Monaco and language servers emit).
export function pathToFileUri(absPath) {
  let p = String(absPath).replace(/\\/g, '/');
  if (!p.startsWith('/')) p = '/' + p; // C:/... -> /C:/...
  const encoded = p.split('/').map(encodeURIComponent).join('/');
  return 'file://' + encoded;
}

// workspaceFileUri is the common case: the `file://` URI for a
// workspace-relative path.
export function workspaceFileUri(workspacePath, relPath) {
  return pathToFileUri(joinWorkspacePath(workspacePath, relPath));
}

// fileUriToRelPath converts an LSP `file://` URI (from any server, on any OS)
// into a workspace-relative path with forward slashes. It tolerates
// percent-encoding, backslashes, and the `/C:/` drive prefix, and compares the
// drive path case-insensitively (Windows filesystems are case-insensitive and
// servers disagree on drive-letter case).
export function fileUriToRelPath(uri, workspacePath) {
  let p = String(uri).replace(/^file:\/\//, '');
  try { p = decodeURIComponent(p); } catch { /* leave as-is */ }
  p = p.replace(/\\/g, '/');
  // Windows URIs are `/C:/Users/...` — drop the slash before the drive letter.
  if (/^\/[A-Za-z]:\//.test(p)) p = p.slice(1);

  const base = joinWorkspacePath(workspacePath);
  const isWin = /^[A-Za-z]:\//.test(base);
  const a = isWin ? p.toLowerCase() : p;
  const b = isWin ? base.toLowerCase() : base;
  if (base && a.startsWith(b)) p = p.slice(base.length);
  if (p.startsWith('/')) p = p.slice(1);
  return p;
}
