const { test } = require('../fixtures');

test('01-file-explorer', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  await snap('01-initial-workspace');
  log('Workspace loaded with explorer visible');

  // Expand a directory
  const dirs = await page.locator('.node-row .chevron-wrapper:not(.hidden)').all();
  if (dirs.length > 0) {
    await dirs[0].click();
    await page.waitForTimeout(500);
    await snap('02-after-expand-dir');
    log(`Expanded first directory (${dirs.length} dirs available)`);
  }

  // Open a few files
  const fileRows = await page.locator('.node-row').all();
  let opened = 0;
  for (const row of fileRows) {
    if (opened >= 3) break;
    const hidden = await row.locator('.chevron-wrapper.hidden').count();
    if (hidden > 0) {
      const name = await row.locator('.node-name').textContent();
      await row.click();
      await page.waitForTimeout(600);
      log(`Opened file: ${name}`);
      opened++;
    }
  }

  await snap('03-after-open-files');
  log(`Opened ${opened} files — tabs should be visible`);

  // Verify tabs exist
  const tabCount = await page.locator('.tabs-bar .tab').count();
  log(`Tab count: ${tabCount}`);
  if (tabCount !== opened) {
    throw new Error(`Expected ${opened} tabs, got ${tabCount}`);
  }

  // Click between tabs to show switching
  const tabs = await page.locator('.tabs-bar .tab').all();
  if (tabs.length >= 2) {
    await tabs[0].click();
    await page.waitForTimeout(400);
    await snap('04-switched-to-first-tab');

    await tabs[tabs.length - 1].click();
    await page.waitForTimeout(400);
    await snap('05-switched-to-last-tab');
    log('Tab switching works');
  }
});
