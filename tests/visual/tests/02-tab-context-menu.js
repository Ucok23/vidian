module.exports = async function({ page, baseUrl, screenshot, log }) {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Open 4 files to have multiple tabs
  const fileRows = await page.locator('.node-row').all();
  let opened = 0;
  for (const row of fileRows) {
    if (opened >= 4) break;
    const hidden = await row.locator('.chevron-wrapper.hidden').count();
    if (hidden > 0) {
      await row.click();
      await page.waitForTimeout(400);
      opened++;
    }
  }

  await screenshot('01-before-4-tabs-open');
  log(`Opened ${opened} tabs`);

  // Right-click the second tab to show context menu
  const tabs = await page.locator('.tabs-bar .tab').all();
  if (tabs.length < 2) throw new Error('Need at least 2 tabs for this test');

  await tabs[1].click({ button: 'right' });
  await page.waitForTimeout(400);
  await screenshot('02-context-menu-visible');

  // Verify menu items
  const items = await page.locator('.context-menu-item').allTextContents();
  log(`Menu items: ${items.join(', ')}`);
  const expected = ['Close', 'Close Others', 'Close to the Right', 'Close All', 'Copy Path'];
  for (const e of expected) {
    if (!items.includes(e)) throw new Error(`Missing menu item: ${e}`);
  }

  // Dismiss and reopen on second tab, then click "Close to the Right"
  await page.click('.editor-container');
  await page.waitForTimeout(300);

  await tabs[1].click({ button: 'right' });
  await page.waitForTimeout(300);
  await page.locator('.context-menu-item', { hasText: 'Close to the Right' }).click();
  await page.waitForTimeout(400);
  await screenshot('03-after-close-to-right');

  const remaining = await page.locator('.tabs-bar .tab').count();
  log(`Tabs after "Close to the Right": ${remaining} (expected 2)`);
  if (remaining !== 2) throw new Error(`Expected 2 tabs after close-right, got ${remaining}`);

  // Now right-click first tab → Close Others
  const freshTabs = await page.locator('.tabs-bar .tab').all();
  await freshTabs[0].click({ button: 'right' });
  await page.waitForTimeout(300);
  await page.locator('.context-menu-item', { hasText: 'Close Others' }).click();
  await page.waitForTimeout(400);
  await screenshot('04-after-close-others');

  const afterOthers = await page.locator('.tabs-bar .tab').count();
  log(`Tabs after "Close Others": ${afterOthers} (expected 1)`);
  if (afterOthers !== 1) throw new Error(`Expected 1 tab after close-others, got ${afterOthers}`);

  // Finally, close all
  const lastTab = page.locator('.tabs-bar .tab').first();
  await lastTab.click({ button: 'right' });
  await page.waitForTimeout(300);
  await page.locator('.context-menu-item', { hasText: 'Close All' }).click();
  await page.waitForTimeout(400);
  await screenshot('05-after-close-all');

  const afterAll = await page.locator('.tabs-bar .tab').count();
  log(`Tabs after "Close All": ${afterAll} (expected 0)`);
  if (afterAll !== 0) throw new Error(`Expected 0 tabs after close-all, got ${afterAll}`);
};
