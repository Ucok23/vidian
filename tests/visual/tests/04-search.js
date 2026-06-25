module.exports = async function({ page, baseUrl, screenshot, log }) {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  await screenshot('01-before-search');

  // Click Search in activity bar
  const searchBtn = page.locator('.activity-btn[title="Search (Ctrl+Shift+F)"]');
  await searchBtn.click();
  await page.waitForTimeout(500);
  await screenshot('02-search-panel-open');
  log('Search panel opened');

  // Type a search query
  const searchInput = page.locator('.search-box input');
  await searchInput.waitFor({ timeout: 5000 });
  await searchInput.fill('func');
  await page.waitForTimeout(2000);
  await screenshot('03-search-results');

  const resultCount = await page.locator('.match-row').count();
  log(`Search results for "func": ${resultCount}`);

  if (resultCount > 0) {
    // Click first result to navigate to file
    await page.locator('.match-row').first().click();
    await page.waitForTimeout(800);
    await screenshot('04-after-click-result');
    log('Navigated to first search result');

    const tabs = await page.locator('.tabs-bar .tab').count();
    log(`Tabs open after search navigation: ${tabs}`);
  }

  // Switch to explorer first, then back to search to ensure sidebar is open on search
  const explorerBtn = page.locator('.activity-btn[title="Explorer (Ctrl+B)"]');
  await explorerBtn.click();
  await page.waitForTimeout(300);
  await searchBtn.click();
  await page.waitForTimeout(500);

  // Clear and try another query
  const input = page.locator('.search-box input');
  await input.waitFor({ timeout: 5000 });
  await input.fill('');
  await page.waitForTimeout(300);
  await input.fill('import');
  await page.waitForTimeout(2000);
  await screenshot('05-second-search');

  const resultCount2 = await page.locator('.match-row').count();
  log(`Search results for "import": ${resultCount2}`);

  await screenshot('06-final-state');
};
