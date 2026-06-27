const { test } = require('../fixtures');

test('11-gitlens-contributors-stashes-tags', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git panel
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping contributors/stashes/tags tests');
    return;
  }

  await gitBtn.click();
  await page.waitForTimeout(800);
  await snap('01-git-panel-open');
  log('Git panel opened');

  // ── CONTRIBUTORS ──────────────────────────────────────────────────────────
  const contribHeader = page.locator('.section-header').filter({ hasText: 'CONTRIBUTORS' });
  if (await contribHeader.count() === 0) {
    log('CONTRIBUTORS section not found');
    await snap('02-no-contributors');
  } else {
    await contribHeader.click();
    await page.waitForTimeout(1500); // triggers API load on first open
    await snap('02-contributors-expanded');
    log('CONTRIBUTORS section clicked');

    const contribSection = page.locator('.panel-section').filter({ hasText: 'CONTRIBUTORS' });

    // Wait for contributor items to render
    const contribItems = contribSection.locator('.contributor-item');
    try {
      await contribItems.first().waitFor({ state: 'visible', timeout: 5000 });
      const count = await contribItems.count();
      log(`Contributors found: ${count}`);
      await snap('03-contributors-loaded');

      // Log contributor names
      for (let i = 0; i < Math.min(count, 5); i++) {
        const item = contribItems.nth(i);
        const name = await item.locator('.contributor-name').textContent().catch(() => '?');
        const commits = await item.locator('.contributor-count').textContent().catch(() => '?');
        log(`  ${name.trim()} — ${commits.trim()} commits`);
      }

      // Check avatar initials
      const avatars = contribSection.locator('.contributor-avatar');
      log(`Avatar elements: ${await avatars.count()}`);
    } catch {
      log('Contributors did not load within timeout');
      await snap('03-contributors-timeout');

      // Check for loading indicator
      const loading = contribSection.locator('.loading-text, .spinner');
      if (await loading.count() > 0) {
        log('Loading indicator present');
      }
    }
  }

  await snap('04-after-contributors');

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
    const emptyMsg = stashSection.locator('.empty-state, .no-items, .dim-text');
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
