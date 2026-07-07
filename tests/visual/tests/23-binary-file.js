const { test } = require('../fixtures');

// Binary file screen: opening a non-image binary (a font ships in the built
// frontend) shows the "cannot be displayed" card instead of Monaco.
test('23-binary-file', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await page.keyboard.press('Control+p');
  await page.waitForTimeout(400);
  await page.keyboard.type('codicon');
  await page.waitForTimeout(700);
  const rows = await page.locator('.file-row .file-name').allTextContents();
  log(`Palette rows for "codicon": ${rows.join(', ') || '(none)'}`);
  if (rows.length === 0) {
    log('no codicon font found in workspace — skipping binary screen test');
    await snap('01-no-binary-file');
    return;
  }
  await page.keyboard.press('Enter');

  const screen = page.locator('.binary-screen');
  try {
    await screen.waitFor({ state: 'visible', timeout: 8000 });
  } catch {
    log('binary screen did not appear');
    await snap('01-no-binary-screen');
    throw new Error('binary screen did not render for the font file');
  }
  await page.waitForTimeout(400);
  await snap('01-binary-screen');

  const heading = await page.locator('.binary-card h3').textContent().catch(() => '');
  const meta = await page.locator('.binary-card .meta-row strong').allTextContents();
  log(`Binary card heading: "${heading}", meta: ${meta.join(' | ')}`);
  if (!/binary/i.test(heading)) throw new Error(`unexpected binary card heading: "${heading}"`);
});
