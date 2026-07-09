const { test } = require('../fixtures');

test('13-insights-panel', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Skip if not a git repository (no Repo button will be shown).
  const repoBtn = page.locator('.activity-btn[title="Repo (Overview & Insights)"]');
  if (await repoBtn.count() === 0) {
    log('Not a git repo — skipping insights panel test');
    return;
  }

  await snap('01-initial-state');
  log('App loaded');

  // Open the merged Repo document via the activity bar icon.
  await repoBtn.click();
  await page.waitForTimeout(800);

  // A single tab labelled "Repo" must be active.
  const insightsTab = page.locator('.tab', { hasText: 'Repo' });
  if (await insightsTab.count() === 0) {
    throw new Error('"Repo" tab not created in tab bar');
  }
  const isActive = await insightsTab.evaluate(el => el.classList.contains('active'));
  if (!isActive) throw new Error('"Repo" tab is not the active tab');
  log('Repo tab is active');

  // Switch to the Insights segment.
  await page.locator('.segment', { hasText: 'Insights' }).click();
  await page.waitForTimeout(2500); // allow the insights API calls to complete
  await snap('02-insights-opened');
  log('Insights segment opened');

  // The insights container must be visible.
  const insights = page.locator('.insights');
  await insights.waitFor({ state: 'visible', timeout: 5000 });

  // Heatmap: at least one non-empty cell must be rendered.
  const filledCells = page.locator('.heatmap-grid .cell:not(.future)');
  const cellCount = await filledCells.count();
  if (cellCount === 0) throw new Error('Heatmap rendered no cells');
  log(`Heatmap: ${cellCount} cells rendered`);

  await snap('03-heatmap-visible');

  // Hot files: at least one bar row should be present.
  const barRows = page.locator('.bar-row');
  const barCount = await barRows.count();
  if (barCount === 0) throw new Error('No hot-file bar rows rendered');
  log(`Hot files: ${barCount} entries`);

  // Author rows should also appear.
  const authorRows = page.locator('.author-row');
  const authorCount = await authorRows.count();
  if (authorCount === 0) throw new Error('No author rows rendered');
  log(`Authors: ${authorCount} entries`);

  await snap('04-full-panel');

  // Clicking a hot-file bar should open that file as a new tab.
  const firstBar = barRows.first();
  const fileName = await firstBar.locator('.bar-label').textContent();
  await firstBar.click();
  await page.waitForTimeout(800);

  const openedTab = page.locator('.tab', { hasText: fileName.trim() });
  const opened = await openedTab.count();
  log(`Clicked hot file "${fileName.trim()}" — tab opened: ${opened > 0}`);
  if (opened === 0) throw new Error(`Expected a tab for "${fileName.trim()}" to open`);

  await snap('05-file-opened-from-hot-files');

  // Switch back to the insights tab.
  await insightsTab.click();
  await page.waitForTimeout(400);
  await snap('06-back-to-insights');
  log('Returned to Insights tab — test complete');
});
