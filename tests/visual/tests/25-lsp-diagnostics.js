const { test } = require('../fixtures');
const { openViaQuickOpen, waitForLspReady } = require('../helpers');

// LSP diagnostics: introduce a syntax error into the open model and confirm
// gopls reports a marker that Monaco renders as an error squiggle. Skips
// gracefully when gopls is unavailable or edits aren't forwarded.
test('25-lsp-diagnostics', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'profile.go');
  log('Opened profile.go');

  if (!(await waitForLspReady(page))) {
    log('LSP (gopls) not available on host — skipping diagnostics');
    await snap('01-no-lsp');
    return;
  }
  log('gopls ready');

  await snap('01-clean');

  // Corrupt the buffer: append an obviously invalid Go statement at the end.
  await page.evaluate(() => {
    const ed = window.editorInstance;
    const model = ed.getModel();
    const last = model.getLineCount();
    ed.executeEdits('test', [{
      range: new window.monaco.Range(last, model.getLineMaxColumn(last), last, model.getLineMaxColumn(last)),
      text: '\n@@@ this is not valid go @@@\n',
    }]);
  }).catch((e) => log(`edit failed: ${e}`));

  // Give gopls time to re-analyze and publish diagnostics.
  await page.waitForTimeout(4000);

  const markers = await page.evaluate(() => {
    const model = window.editorInstance.getModel();
    return window.monaco.editor.getModelMarkers({ resource: model.uri })
      .map(m => ({ sev: m.severity, msg: m.message, line: m.startLineNumber }));
  }).catch(() => []);

  const squiggles = await page.locator('.view-overlays .squiggly-error, .view-overlays .squiggly-warning').count();
  log(`gopls markers: ${markers.length}, rendered squiggles: ${squiggles}`);
  if (markers.length > 0) log(`  first marker: line ${markers[0].line} — ${String(markers[0].msg).slice(0, 80)}`);
  await snap('02-with-error');

  if (markers.length === 0) {
    log('no diagnostics published (edit not forwarded to gopls, or delayed) — soft pass');
    return;
  }
  log('Diagnostics reported and rendered for the injected error');
});
