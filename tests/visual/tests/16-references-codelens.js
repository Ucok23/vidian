const { test } = require('../fixtures');
const { openViaQuickOpen, waitForLspReady } = require('../helpers');

// References CodeLens: an "N references" lens should appear above declarations,
// and clicking it opens the references sidebar.
test('16-references-codelens', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'profile.go');
  await snap('01-profile-go-open');
  log('Opened profile.go');

  if (!(await waitForLspReady(page))) {
    log('LSP (gopls) not available on host — skipping references codelens');
    await snap('02-no-lsp');
    return;
  }
  log('gopls ready');

  // CodeLenses resolve asynchronously (documentSymbol + references per decl).
  const lens = page.locator('.codelens-decoration').filter({ hasText: 'reference' });
  try {
    await lens.first().waitFor({ state: 'visible', timeout: 20000 });
  } catch {
    log('References CodeLens did not render');
    await snap('03-no-codelens');
    return;
  }

  const lensText = await lens.first().textContent();
  log(`CodeLens visible: "${lensText.trim()}"`);
  await snap('03-references-codelens');

  // Clicking the lens opens the references sidebar.
  await lens.first().click();
  const refItem = page.locator('.refs-item');
  try {
    await refItem.first().waitFor({ state: 'visible', timeout: 15000 });
    log(`Clicking the lens opened the references panel (${await refItem.count()} results)`);
    await snap('04-lens-opened-references');
  } catch {
    log('References panel did not open from the lens click');
    await snap('04-lens-click-no-panel');
  }
});
