const { test } = require('../fixtures');
const { openViaQuickOpen, waitForLspReady, locateWord, placeCursor } = require('../helpers');

// Go-to-definition: from a cross-file usage of git.GetRepoProfile in server.go,
// jump to its declaration in internal/git/profile.go (a new tab).
test('14-goto-definition', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'server.go');
  await snap('01-server-go-open');
  log('Opened server.go');

  if (!(await waitForLspReady(page))) {
    log('LSP (gopls) not available on host — skipping go-to-definition');
    await snap('02-no-lsp');
    return;
  }
  log('gopls ready');

  const pos = await locateWord(page, 'GetRepoProfile');
  if (!pos) { log('GetRepoProfile usage not found'); await snap('02-symbol-missing'); return; }
  await placeCursor(page, pos.line, pos.column);
  await snap('03-cursor-on-usage');
  log(`Cursor on GetRepoProfile usage at ${pos.line}:${pos.column}`);

  await page.evaluate(() => window.editorInstance.trigger('test', 'editor.action.revealDefinition', {}));
  await page.waitForTimeout(1800);
  await snap('04-after-goto-definition');

  const uri = await page.evaluate(() => window.editorInstance.getModel().uri.toString());
  const activeTab = await page.locator('.tabs-bar .tab.active .tab-name').textContent().catch(() => '');
  log(`Active tab: ${activeTab} | model: ${uri}`);
  if (!/profile\.go$/.test(uri)) {
    throw new Error(`Go-to-definition did not navigate to profile.go (got ${uri})`);
  }
  log('Navigated to the definition in profile.go');
});
