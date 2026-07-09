const { test } = require('../fixtures');

// Onboarding — deterministic repo profile (NOT the AI tour). Opens the panel
// and verifies the non-AI sections render: Stack, Entry Points, Key Files and
// the Activity stats strip.
test('26-onboarding-profile', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  const repoBtn = page.locator('.activity-btn[title="Repo (Overview & Insights)"]');
  if (await repoBtn.count() === 0) {
    log('Not a git repo — skipping onboarding profile test');
    return;
  }
  await repoBtn.click();
  await page.waitForTimeout(500);
  // The Repo document opens on the Overview (onboarding) segment by default.
  await snap('01-onboarding-opened');

  const panel = page.locator('.onboarding');
  await panel.waitFor({ state: 'visible', timeout: 8000 });

  // Wait for analysis to finish (loading -> profile).
  try {
    await page.locator('.onboarding .loading').first().waitFor({ state: 'hidden', timeout: 15000 });
  } catch {
    log('profile still analyzing after 15s');
  }
  await page.waitForTimeout(500);
  await snap('02-profile-loaded');

  const cardTitles = await page.locator('.onboarding .card-title').allTextContents();
  const norm = cardTitles.map(t => t.replace(/\s+/g, ' ').trim());
  log(`Cards: ${norm.join(' | ')}`);

  const has = (label) => norm.some(t => t.startsWith(label));
  for (const section of ['Stack', 'Entry Points', 'Key Files']) {
    if (!has(section)) throw new Error(`onboarding profile missing "${section}" section`);
  }

  // Detail assertions on the deterministic content.
  const stackChips = await page.locator('.onboarding .chip .chip-name').allTextContents();
  const keyFiles = await page.locator('.onboarding .file-row .file-path').allTextContents();
  log(`Stack chips: ${stackChips.join(', ') || '(none)'}`);
  log(`Listed files (entry + key): ${keyFiles.length}`);

  // Repo stats strip (deterministic git/file counts) — now shared above both
  // Repo segments rather than inside the onboarding panel.
  const stats = await page.locator('.repo .stats-strip .stat-value').allTextContents();
  if (stats.length > 0) {
    log(`Repo stats: ${stats.join(' / ')}`);
    await snap('03-activity-stats');
  } else {
    log('no stats block (no git stats available)');
  }

  if (stackChips.length === 0 && keyFiles.length === 0) {
    throw new Error('profile rendered but both Stack and file lists are empty');
  }
});
