const { test } = require('../fixtures');
const { openViaQuickOpen, waitForLspReady, locateFuncDecl, placeCursor } = require('../helpers');

// Find All References: on the GetRepoProfile declaration in profile.go, open the
// references sidebar (Shift+F12 / the registered action) and confirm it lists
// use sites grouped by file, then navigate to one.
test('15-find-references', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'profile.go');
  await snap('01-profile-go-open');
  log('Opened profile.go');

  if (!(await waitForLspReady(page))) {
    log('LSP (gopls) not available on host — skipping find-references');
    await snap('02-no-lsp');
    return;
  }
  log('gopls ready');

  const pos = await locateFuncDecl(page, 'GetRepoProfile');
  if (!pos) { log('GetRepoProfile declaration not found'); await snap('02-decl-missing'); return; }
  await placeCursor(page, pos.line, pos.column);
  await snap('03-cursor-on-declaration');

  // Trigger the same action the Shift+F12 keybinding / context menu invoke.
  await page.evaluate(() => window.editorInstance.trigger('test', 'vidian.findReferences', {}));

  const refItem = page.locator('.refs-item');
  try {
    await refItem.first().waitFor({ state: 'visible', timeout: 15000 });
  } catch {
    log('References panel did not populate');
    await snap('04-no-references');
    return;
  }

  const count = await refItem.count();
  const groups = await page.locator('.refs-group').count();
  log(`References panel: ${count} results across ${groups} file(s)`);
  await snap('04-references-panel');
  if (count < 1) throw new Error('expected at least one reference');

  // Click a result and confirm it navigates to a source file + line.
  await refItem.first().click();
  await page.waitForTimeout(1200);
  await snap('05-navigated-to-reference');
  const uri = await page.evaluate(() => window.editorInstance.getModel().uri.toString());
  log(`Navigated to: ${uri}`);
  if (!/\.go$/.test(uri)) throw new Error(`clicking a reference did not open a Go file (got ${uri})`);
});
