const { test } = require('../fixtures');

test('10-gitlens-search-compare', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git panel
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping search/compare tests');
    return;
  }

  await gitBtn.click();
  await page.waitForTimeout(800);
  await snap('01-git-panel-open');
  log('Git panel opened');

  // Search & Compare are grouped behind the MORE divider — reveal it first.
  const moreDivider = page.locator('.more-divider');
  if (await moreDivider.count() === 0) throw new Error('MORE divider not found');
  await moreDivider.click();
  await page.waitForTimeout(400);
  log('MORE group revealed');

  // ── SEARCH COMMITS ────────────────────────────────────────────────────────
  const searchHeader = page.locator('.section-header').filter({ hasText: 'SEARCH COMMITS' });
  if (await searchHeader.count() === 0) {
    log('SEARCH COMMITS section not found');
    await snap('02-no-search-section');
  } else {
    await searchHeader.click();
    await page.waitForTimeout(500);
    await snap('02-search-section-expanded');
    log('SEARCH COMMITS section expanded');

    const searchSection = page.locator('.panel-section').filter({ hasText: 'SEARCH COMMITS' });

    // Fill keyword input
    const keywordInput = searchSection.locator('.search-input').nth(0);
    await keywordInput.waitFor({ state: 'visible', timeout: 3000 });
    await keywordInput.fill('feat');
    log('Typed "feat" in keyword input');

    // Fill author input
    const authorInput = searchSection.locator('.search-input').nth(1);
    if (await authorInput.count() > 0) {
      await authorInput.fill('');
      log('Author input cleared');
    }

    await snap('03-search-inputs-filled');

    // Click Search button
    const searchBtn = searchSection.locator('.search-btn');
    await searchBtn.click();
    await page.waitForTimeout(2000); // wait for results
    log('Search button clicked');
    await snap('04-search-results');

    // Check results
    const results = searchSection.locator('.commit-summary, .commit-row');
    const resultCount = await results.count();
    log(`Search results for "feat": ${resultCount} commits`);

    if (resultCount > 0) {
      // Click first result
      await results.first().click();
      await page.waitForTimeout(1000);
      await snap('05-search-commit-detail');
      log('Opened commit from search result');

      const commitViewer = page.locator('.commit-viewer-container');
      if (await commitViewer.count() > 0) {
        log('Commit viewer rendered from search result');
      }
    } else {
      // Try searching for something more generic
      await keywordInput.fill('fix');
      await searchBtn.click();
      await page.waitForTimeout(2000);
      const altResults = await searchSection.locator('.commit-summary, .commit-row').count();
      log(`Search results for "fix": ${altResults} commits`);
      await snap('05-search-alt-results');
    }

    // Test searching by author only
    await keywordInput.fill('');
    if (await authorInput.count() > 0) {
      await authorInput.fill('Ucok');
      await searchBtn.click();
      await page.waitForTimeout(2000);
      const authorResults = await searchSection.locator('.commit-summary, .commit-row').count();
      log(`Search by author "Ucok": ${authorResults} commits`);
      await snap('06-search-by-author');
    }
  }

  // ── COMPARE REFS ─────────────────────────────────────────────────────────
  const compareHeader = page.locator('.section-header').filter({ hasText: 'COMPARE REFS' });
  if (await compareHeader.count() === 0) {
    log('COMPARE REFS section not found');
    await snap('07-no-compare-section');
  } else {
    await compareHeader.click();
    await page.waitForTimeout(500);
    await snap('07-compare-section-expanded');
    log('COMPARE REFS section expanded');

    const compareSection = page.locator('.panel-section').filter({ hasText: 'COMPARE REFS' });

    // Fill base ref (HEAD~1) and compare ref (HEAD)
    const baseInput = compareSection.locator('.search-input').nth(0);
    const compareInput = compareSection.locator('.search-input').nth(1);

    await baseInput.waitFor({ state: 'visible', timeout: 3000 });
    await baseInput.fill('HEAD~1');
    log('Filled base ref: HEAD~1');

    await compareInput.fill('HEAD');
    log('Filled compare ref: HEAD');

    await snap('08-compare-inputs-filled');

    // Click Compare button
    const compareBtn = compareSection.locator('.search-btn');
    await compareBtn.click();
    await page.waitForTimeout(2500); // git diff can be slow
    log('Compare button clicked');
    await snap('09-compare-results');

    // Check diff stat
    const diffStat = compareSection.locator('.compare-stat, pre');
    if (await diffStat.count() > 0) {
      const statText = await diffStat.first().textContent().catch(() => '');
      log(`Diff stat: ${statText.trim().slice(0, 100)}`);
    }

    // Check file list
    const compareFiles = compareSection.locator('.compare-file, .commit-file-item');
    const fileCount = await compareFiles.count();
    log(`Changed files in compare: ${fileCount}`);

    if (fileCount > 0) {
      log('Compare results show file list');
      await snap('10-compare-file-list');

      // Click a file to open diff
      await compareFiles.first().click();
      await page.waitForTimeout(1500);
      await snap('11-compare-diff-opened');
      log('Opened diff from compare result');
    }

    // Test with named branches
    await baseInput.fill('main');
    await compareInput.fill('HEAD');
    await compareBtn.click();
    await page.waitForTimeout(2500);
    await snap('12-compare-main-to-head');
    log('Compared main..HEAD');
  }

  await snap('13-final-state');
});
