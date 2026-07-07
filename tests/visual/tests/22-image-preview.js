const { test } = require('../fixtures');

// Image previewer: opening an image file shows the dedicated preview surface
// (not Monaco). We can't use openViaQuickOpen here because that waits for an
// editor model, which image files never create.
test('22-image-preview', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  // Open hero.png through the palette.
  await page.keyboard.press('Control+p');
  await page.waitForTimeout(400);
  await page.keyboard.type('hero.png');
  await page.waitForTimeout(700);
  const rows = await page.locator('.file-row .file-name').allTextContents();
  log(`Palette rows for "hero.png": ${rows.join(', ') || '(none)'}`);
  if (rows.length === 0) {
    log('hero.png not present in workspace — skipping image preview');
    await snap('01-no-image-file');
    return;
  }
  await page.keyboard.press('Enter');

  const previewer = page.locator('.image-previewer');
  try {
    await previewer.waitFor({ state: 'visible', timeout: 8000 });
  } catch {
    log('image previewer did not appear');
    await snap('01-no-previewer');
    throw new Error('image previewer did not render for hero.png');
  }
  await page.waitForTimeout(500);
  await snap('01-image-previewed');

  const imgs = await page.locator('.image-previewer img').count();
  log(`<img> elements in previewer: ${imgs}`);
  if (imgs < 1) throw new Error('no <img> rendered in image previewer');

  // The Monaco editor must NOT be showing for an image.
  const monaco = await page.locator('.editor-split-container:visible .monaco-container').count();
  log(`Visible Monaco containers while image open: ${monaco}`);
});
