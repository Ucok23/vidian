const { test } = require('../fixtures');

test('11-gitlens-stashes-tags', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git panel
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping stashes/tags tests');
    return;
  }

  await gitBtn.click();
  await page.waitForTimeout(800);
  await snap('01-git-panel-open');
  log('Git panel opened');

  // Contributors now lives in the Repo → Insights document, not Source Control.
  const contribHeader = page.locator('.section-header').filter({ hasText: 'CONTRIBUTORS' });
  if (await contribHeader.count() !== 0) {
    throw new Error('CONTRIBUTORS should no longer appear in Source Control');
  }
  log('Contributors correctly absent from Source Control');

  // Stashes/Tags/Branches/Search/Compare are grouped behind the MORE divider —
  // reveal it so the secondary sections mount.
  const moreDivider = page.locator('.more-divider');
  if (await moreDivider.count() === 0) throw new Error('MORE divider not found');
  await moreDivider.click();
  await page.waitForTimeout(400);
  await snap('02-more-revealed');
  log('MORE group revealed');

  // ── STASHES ───────────────────────────────────────────────────────────────
  const stashHeader = page.locator('.section-header').filter({ hasText: 'STASHES' });
  if (await stashHeader.count() === 0) {
    log('STASHES section not found');
    await snap('05-no-stashes');
  } else {
    await stashHeader.click();
    await page.waitForTimeout(1500); // triggers API load
    await snap('05-stashes-expanded');
    log('STASHES section clicked');

    const stashSection = page.locator('.panel-section').filter({ hasText: 'STASHES' });

    // Wait briefly for stash items
    await page.waitForTimeout(1000);

    // Check for stash entries (the project may have no stashes, that's OK)
    const stashItems = stashSection.locator('.stash-item, .commit-row, li');
    const stashCount = await stashItems.count();
    log(`Stash entries: ${stashCount}`);

    // Check for empty state message
    if (stashCount === 0) {
      const emptyText = await stashSection.textContent().catch(() => '');
      if (emptyText.includes('No stashes') || emptyText.includes('no stash') || emptyText.includes('empty')) {
        log('No stashes message shown (expected for a clean repo)');
      } else {
        log('Stash section shows content but no individual items recognized');
      }
    } else {
      // Log stash info
      for (let i = 0; i < Math.min(stashCount, 3); i++) {
        const item = stashItems.nth(i);
        const text = await item.textContent().catch(() => '');
        log(`  Stash ${i}: ${text.trim().slice(0, 60)}`);
      }
    }

    await snap('06-stashes-content');
  }

  await snap('07-after-stashes');

  // ── TAGS ──────────────────────────────────────────────────────────────────
  const tagHeader = page.locator('.section-header').filter({ hasText: 'TAGS' });
  if (await tagHeader.count() === 0) {
    log('TAGS section not found');
    await snap('08-no-tags');
  } else {
    await tagHeader.click();
    await page.waitForTimeout(1500); // triggers API load
    await snap('08-tags-expanded');
    log('TAGS section clicked');

    const tagSection = page.locator('.panel-section').filter({ hasText: 'TAGS' });

    await page.waitForTimeout(1000);

    const tagItems = tagSection.locator('.tag-item');
    const tagCount = await tagItems.count();
    log(`Tag entries: ${tagCount}`);

    if (tagCount === 0) {
      const sectionText = await tagSection.textContent().catch(() => '');
      const hasEmpty = sectionText.toLowerCase().includes('no tag') || sectionText.includes('empty');
      log(hasEmpty ? 'No tags — empty state shown' : 'No tag items found (selector may differ)');
      await snap('09-tags-empty');
    } else {
      // Log tag names
      for (let i = 0; i < Math.min(tagCount, 5); i++) {
        const tag = tagItems.nth(i);
        const name = await tag.locator('.tag-name, .node-name, span').first().textContent().catch(() => '?');
        log(`  Tag: ${name.trim()}`);
      }
      await snap('09-tags-loaded');

      // Click a tag to open its commit
      await tagItems.first().click();
      await page.waitForTimeout(1200);
      await snap('10-tag-commit-opened');
      log('Opened commit from tag click');

      const commitViewer = page.locator('.commit-viewer-container');
      if (await commitViewer.count() > 0) {
        log('Commit viewer opened from tag');
      }
    }
  }

  await snap('11-final-state');
});
