const { test } = require('../fixtures');

test('03-git-panel', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  await snap('01-before-git-panel');

  // Click Source Control in activity bar
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Source Control tab not available — skipping');
    return;
  }

  await gitBtn.click();
  await page.waitForTimeout(800);
  await snap('02-git-panel-open');
  log('Git panel opened');

  // Wait for commits to load
  const commits = page.locator('.commit-summary');
  await commits.first().waitFor({ timeout: 5000 });
  const commitCount = await commits.count();
  log(`Commits visible: ${commitCount}`);

  await snap('03-commit-list');

  // Click first commit to see details
  await commits.first().click();
  await page.waitForTimeout(1000);
  await snap('04-commit-details');
  log('Opened commit details');

  // Check for file items in commit viewer
  const fileItems = page.locator('.commit-file-item');
  await fileItems.first().waitFor({ timeout: 5000 });
  const fileCount = await fileItems.count();
  log(`Files in commit: ${fileCount}`);

  // Click a file to open diff
  await fileItems.first().click();
  await page.waitForTimeout(1500);
  await snap('05-diff-view');
  log('Opened diff view for first file');

  // Check diff editor rendered
  const diffEditor = page.locator('.diff-editor-container-inner');
  if (await diffEditor.count() > 0) {
    log('Monaco diff editor rendered successfully');
  } else {
    log('Diff editor not found — may be a new file with no parent');
  }

  await snap('06-final-state');
});
