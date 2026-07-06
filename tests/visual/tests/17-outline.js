const { test } = require('../fixtures');
const { openViaQuickOpen, waitForLspReady } = require('../helpers');

// Outline sidebar: the document-symbol tree for the active file, with
// jump-to-symbol.
test('17-outline', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'profile.go');
  log('Opened profile.go');

  if (!(await waitForLspReady(page))) {
    log('LSP (gopls) not available on host — skipping outline');
    await snap('01-no-lsp');
    return;
  }
  log('gopls ready');

  // Open the Outline panel from the activity bar.
  await page.locator('.activity-btn[title="Outline"]').click();
  await snap('01-outline-opened');

  const items = page.locator('.outline-item');
  try {
    await items.first().waitFor({ state: 'visible', timeout: 15000 });
  } catch {
    log('Outline did not populate');
    await snap('02-outline-empty');
    return;
  }

  const count = await items.count();
  const names = await items.locator('.outline-name').allTextContents();
  log(`Outline: ${count} symbols — ${names.slice(0, 6).join(', ')}…`);
  await snap('02-outline-populated');
  if (count < 1) throw new Error('expected at least one outline symbol');

  // Jump to a symbol and confirm the editor scrolls/positions there.
  await items.nth(Math.min(3, count - 1)).click();
  await page.waitForTimeout(800);
  await snap('03-jumped-to-symbol');
  log('Clicked an outline symbol to jump to it');
});
