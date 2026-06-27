const { test } = require('../fixtures');

test('08-gitlens-file-history', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git repo
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping file history test');
    return;
  }

  // Open a tracked file first so file history loads for it
  const allRows = await page.locator('.node-row').all();
  let targetFile = '';
  for (const row of allRows) {
    const name = await row.locator('.node-name').textContent().catch(() => '');
    if (name.trim() === 'go.mod') {
      await row.click();
      await page.waitForTimeout(1000);
      targetFile = 'go.mod';
      log('Opened go.mod to load its file history');
      break;
    }
  }

  if (!targetFile) {
    // Fallback: open any tracked file
    for (const row of allRows) {
      const hidden = await row.locator('.chevron-wrapper.hidden').count();
      if (hidden > 0) {
        const name = await row.locator('.node-name').textContent().catch(() => '');
        await row.click();
        await page.waitForTimeout(1000);
        targetFile = name.trim();
        log(`Opened ${targetFile}`);
        break;
      }
    }
  }

  // Open git panel
  await gitBtn.click();
  await page.waitForTimeout(800);
  await snap('01-git-panel-open');
  log('Git panel opened');

  // Find and click FILE HISTORY section header to expand it
  const fileHistoryHeader = page.locator('.section-header').filter({ hasText: 'FILE HISTORY' });
  if (await fileHistoryHeader.count() === 0) {
    throw new Error('FILE HISTORY section not found in git panel');
  }

  // Check if already expanded; if collapsed, click to expand
  const fileHistorySection = page.locator('.panel-section').filter({ hasText: 'FILE HISTORY' });
  await fileHistoryHeader.click();
  await page.waitForTimeout(1500); // wait for history to load
  await snap('02-file-history-expanded');
  log('FILE HISTORY section expanded');

  // Look for commit summaries within the file history section
  // File history commits are .commit-summary elements inside the section
  const commitRows = fileHistorySection.locator('.commit-summary');
  const commitCount = await commitRows.count();
  log(`File history commits visible: ${commitCount}`);

  if (commitCount === 0) {
    // Check if there are file history commit rows of another class
    const anyCommits = await fileHistorySection.locator('.commit-row, .history-commit, li').count();
    log(`Alternative commit elements: ${anyCommits}`);
    await snap('02b-file-history-empty');
    log('No commits found in file history — file may be new or selector mismatch');
  } else {
    await snap('03-file-history-commits-visible');
    log(`Found ${commitCount} commits in file history`);

    // Click the first commit to open commit viewer
    await commitRows.first().click();
    await page.waitForTimeout(1200);
    await snap('04-commit-viewer-from-file-history');
    log('Opened commit viewer from file history');

    // Check commit viewer rendered
    const commitViewer = page.locator('.commit-viewer-container');
    if (await commitViewer.count() > 0) {
      log('Commit viewer rendered successfully');

      // Check for parent commit button
      const parentBtn = page.locator('.commit-action-btn').filter({ hasText: 'Parent Commit' });
      if (await parentBtn.count() > 0) {
        log('Parent commit button visible');
        await snap('05-parent-commit-button');

        // Click parent commit
        await parentBtn.click();
        await page.waitForTimeout(1500);
        await snap('06-parent-commit-opened');
        log('Parent commit opened');
      } else {
        log('Parent commit button not found');
        await snap('05-no-parent-btn');
      }

      // Check files changed section
      const fileItems = page.locator('.commit-file-item');
      const fileCount = await fileItems.count();
      log(`Files in commit: ${fileCount}`);

      if (fileCount > 0) {
        // Hover over first file to show action buttons
        await fileItems.first().hover();
        await page.waitForTimeout(300);
        await snap('07-file-actions-hover');
        log('Hovered over file to reveal Open/Diff buttons');

        // Click Diff button
        const diffBtn = fileItems.first().locator('.file-action-btn.primary');
        if (await diffBtn.count() > 0) {
          await diffBtn.click();
          await page.waitForTimeout(1500);
          await snap('08-diff-from-commit');
          log('Opened diff from commit viewer');
        }
      }
    } else {
      log('Commit viewer not found in main content area');
      await snap('05-no-commit-viewer');
    }
  }

  await snap('09-final-state');
});
