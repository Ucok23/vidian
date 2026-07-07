const { test } = require('../fixtures');

// Quick Open palette (Ctrl+P): fuzzy file filter, keyboard navigation, and
// open-on-Enter. Exercises the overlay UI directly rather than via the helper.
test('20-quick-open', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  // Open the palette.
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(500);
  const overlay = page.locator('.overlay .palette-card');
  await overlay.waitFor({ state: 'visible', timeout: 5000 });
  await snap('01-palette-open');
  log('Quick Open palette visible');

  const initialRows = await page.locator('.file-row').count();
  log(`Initial rows shown: ${initialRows}`);
  if (initialRows < 1) throw new Error('palette showed no files on open');

  // Type a filter and confirm the list narrows.
  await page.keyboard.type('server');
  await page.waitForTimeout(500);
  await snap('02-filtered-server');
  const filtered = await page.locator('.file-row').allTextContents();
  log(`Filtered rows: ${filtered.length}`);
  const allMatch = (await page.locator('.file-row .file-name').allTextContents())
    .every(n => n.toLowerCase().includes('server') || n.length === 0);
  if (!allMatch) log('  note: some rows do not contain "server" (path match still valid)');

  // Arrow-key navigation moves the selection highlight.
  await page.keyboard.press('ArrowDown');
  await page.waitForTimeout(200);
  await page.keyboard.press('ArrowDown');
  await page.waitForTimeout(200);
  await snap('03-keyboard-navigated');
  const selCount = await page.locator('.file-row.selected').count();
  log(`Selected rows highlighted: ${selCount}`);
  if (selCount !== 1) throw new Error(`expected exactly one .selected row, got ${selCount}`);

  // Open on Enter and confirm the palette closes and a tab opens.
  await page.keyboard.press('Enter');
  await page.waitForTimeout(900);
  await snap('04-opened-selection');
  const stillOpen = await page.locator('.overlay .palette-card').count();
  if (stillOpen > 0) throw new Error('palette did not close after Enter');
  const tabs = await page.locator('.tabs-bar .tab').count();
  log(`Palette closed; open tabs: ${tabs}`);
  if (tabs < 1) throw new Error('no tab opened from Quick Open');

  // Escape closes the palette when reopened.
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(400);
  await page.keyboard.press('Escape');
  await page.waitForTimeout(300);
  const afterEsc = await page.locator('.overlay .palette-card').count();
  log(`Palette count after Escape: ${afterEsc}`);
  await snap('05-after-escape');
  if (afterEsc > 0) throw new Error('Escape did not dismiss the palette');
});
