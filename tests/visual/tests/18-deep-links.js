const { test } = require('../fixtures');
const { openViaQuickOpen } = require('../helpers');

// Deep links: opening a file syncs ?file=&line= to the URL, the "Link" toolbar
// button copies a shareable URL, and reloading that URL restores the file.
// (No language server required.)
test('18-deep-links', async ({ page, baseUrl, snap, log }) => {
  await page.context().grantPermissions(['clipboard-read', 'clipboard-write']).catch(() => {});

  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'profile.go');
  await page.waitForTimeout(600);
  await snap('01-file-open');

  const url = page.url();
  log(`URL after open: ${url}`);
  if (!/[?&]file=/.test(url)) throw new Error(`URL not synced with ?file= (got ${url})`);
  log('URL synced with the active file');

  // Copy the shareable link via the toolbar button.
  const linkBtn = page.locator('.editor-actions button').filter({ hasText: /Link|Copied/ });
  if (await linkBtn.count() > 0) {
    await linkBtn.first().click();
    await page.waitForTimeout(300);
    await snap('02-link-copied');
    const label = await linkBtn.first().textContent();
    log(`Link button after click: "${label.trim()}"`);
    if (!/Copied/.test(label)) log('Note: button did not switch to "Copied" (clipboard perms?)');
  } else {
    log('Link button not found in toolbar');
  }

  // Reload the deep link and confirm the file is restored.
  await page.goto(url, { waitUntil: 'networkidle' });
  await page.waitForFunction(
    () => window.editorInstance && window.editorInstance.getModel(),
    { timeout: 15000 }
  );
  await page.waitForTimeout(800);
  await snap('03-restored-after-reload');

  const activeTab = await page.locator('.tabs-bar .tab.active .tab-name').textContent().catch(() => '');
  const uri = await page.evaluate(() => window.editorInstance.getModel().uri.toString());
  log(`After reload — active tab: ${activeTab}, model: ${uri}`);
  if (!/profile\.go$/.test(uri)) throw new Error(`deep link did not restore profile.go (got ${uri})`);
  log('Deep link restored the file after reload');
});
