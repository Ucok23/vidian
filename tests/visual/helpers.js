// Shared helpers for the visual test suite. These target the running Vidian
// app (Svelte + Monaco) and its LSP bridge.

// openViaQuickOpen opens a file by name using the Ctrl+P palette and waits for
// the Monaco editor model to be ready.
async function openViaQuickOpen(page, filename) {
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(400);
  await page.keyboard.type(filename);
  await page.waitForTimeout(900);
  await page.keyboard.press('Enter');
  await page.waitForFunction(
    () => window.editorInstance && window.editorInstance.getModel(),
    { timeout: 15000 }
  );
}

// waitForLspReady polls the readiness flag the LSP client sets once the
// language server has initialized. Returns true if ready within the timeout,
// false otherwise (so callers can skip LSP-only assertions gracefully when the
// server binary — e.g. gopls — isn't installed on the host).
async function waitForLspReady(page, timeoutMs = 25000) {
  try {
    await page.waitForFunction(() => window._vidianLspReady === true, { timeout: timeoutMs });
    // Give the server a beat to finish indexing the workspace.
    await page.waitForTimeout(1500);
    return true;
  } catch {
    return false;
  }
}

// locateWord returns the 1-based {line, column} of the first occurrence of
// `needle` in the active editor model, or null. Column points just inside the
// match so it lands on the symbol.
async function locateWord(page, needle) {
  return page.evaluate((n) => {
    const model = window.editorInstance?.getModel();
    if (!model) return null;
    const lines = model.getValue().split('\n');
    for (let i = 0; i < lines.length; i++) {
      const idx = lines[i].indexOf(n);
      if (idx !== -1) return { line: i + 1, column: idx + 2 };
    }
    return null;
  }, needle);
}

// locateFuncDecl returns the {line, column} of the name in a `func <name>`
// declaration, landing the column inside the identifier (so LSP requests
// resolve to the symbol rather than the `func` keyword or a comment).
async function locateFuncDecl(page, name) {
  return page.evaluate((n) => {
    const model = window.editorInstance?.getModel();
    if (!model) return null;
    const lines = model.getValue().split('\n');
    const decl = 'func ' + n;
    for (let i = 0; i < lines.length; i++) {
      const idx = lines[i].indexOf(decl);
      if (idx !== -1) return { line: i + 1, column: idx + 'func '.length + 2 };
    }
    return null;
  }, name);
}

// placeCursor moves the editor cursor to a position and focuses the editor.
async function placeCursor(page, line, column) {
  await page.evaluate(({ line, column }) => {
    const ed = window.editorInstance;
    ed.setPosition({ lineNumber: line, column });
    ed.revealLineInCenter(line);
    ed.focus();
  }, { line, column });
}

module.exports = { openViaQuickOpen, waitForLspReady, locateWord, locateFuncDecl, placeCursor };
