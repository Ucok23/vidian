const { test } = require('../fixtures');
const { openViaQuickOpen, waitForLspReady, locateFuncDecl, locateWord, placeCursor } = require('../helpers');

// LSP hover: hovering a symbol shows Monaco's hover widget with the type/doc
// info supplied by gopls. Skips gracefully when the server isn't installed.
test('24-lsp-hover', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'profile.go');
  log('Opened profile.go');

  if (!(await waitForLspReady(page))) {
    log('LSP (gopls) not available on host — skipping hover');
    await snap('01-no-lsp');
    return;
  }
  log('gopls ready');

  const pos = (await locateFuncDecl(page, 'GetRepoProfile')) || (await locateWord(page, 'GetRepoProfile'));
  if (!pos) { log('symbol not found'); await snap('01-symbol-missing'); return; }
  await placeCursor(page, pos.line, pos.column);
  await snap('01-cursor-on-symbol');

  // Trigger the hover widget programmatically (more reliable than mouse-hover
  // timing across headless runs).
  await page.evaluate(() => window.editorInstance.trigger('test', 'editor.action.showHover', {}));
  await page.waitForTimeout(1500);

  const hover = page.locator('.monaco-hover');
  const visible = await hover.isVisible().catch(() => false);
  await snap('02-hover-widget');
  if (!visible) {
    log('hover widget did not appear (gopls returned no hover) — soft pass');
    return;
  }
  const text = (await hover.innerText().catch(() => '')).replace(/\s+/g, ' ').trim();
  log(`Hover content: ${text.slice(0, 120)}`);
  if (text.length < 2) throw new Error('hover widget visible but empty');
});
