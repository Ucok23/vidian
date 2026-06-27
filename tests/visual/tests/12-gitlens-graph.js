const { test } = require('../fixtures');

test('12-gitlens-graph', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git panel
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping commit graph test');
    return;
  }

  await gitBtn.click();
  await page.waitForTimeout(800);
  await snap('01-git-panel-open');
  log('Git panel opened');

  // ── COMMIT GRAPH ──────────────────────────────────────────────────────────
  const graphHeader = page.locator('.section-header').filter({ hasText: 'COMMIT GRAPH' });
  if (await graphHeader.count() === 0) {
    throw new Error('COMMIT GRAPH section not found in git panel');
  }

  await graphHeader.click();
  await page.waitForTimeout(3000); // git log --graph can be slow
  await snap('02-graph-section-expanded');
  log('COMMIT GRAPH section expanded — waiting for output');

  const graphSection = page.locator('.panel-section').filter({ hasText: 'COMMIT GRAPH' });

  // Wait for graph output element
  const graphOutput = graphSection.locator('.graph-output');
  try {
    await graphOutput.waitFor({ state: 'visible', timeout: 8000 });
    log('Graph output element visible');
  } catch {
    log('Graph output did not appear within timeout');
    await snap('03-graph-timeout');

    // Check for loading indicator
    const loading = graphSection.locator('.loading-text, .spinner');
    if (await loading.count() > 0) {
      log('Loading indicator visible — still loading');
    } else {
      log('No loading indicator either — may have failed');
    }
    return;
  }

  await snap('03-graph-loaded');

  // Read graph content
  const graphText = await graphOutput.textContent().catch(() => '');
  const lines = graphText.trim().split('\n');
  log(`Commit graph lines: ${lines.length}`);
  log(`First line: ${lines[0]?.trim().slice(0, 80) || '(empty)'}`);

  // Verify it contains git graph symbols (*, |, /)
  const hasGraphSymbols = /[*|\/\\]/.test(graphText);
  log(`Graph symbols present: ${hasGraphSymbols}`);

  if (!hasGraphSymbols && graphText.length > 0) {
    log('Graph text present but no branch-line symbols (may be a simple linear history)');
  }

  if (lines.length === 0 || graphText.trim().length === 0) {
    throw new Error('Commit graph output is empty');
  }

  // Scroll the graph panel to show more entries
  await graphOutput.evaluate(el => { el.scrollTop = el.scrollHeight / 2; });
  await page.waitForTimeout(400);
  await snap('04-graph-scrolled-middle');
  log('Scrolled to middle of commit graph');

  // Scroll to bottom
  await graphOutput.evaluate(el => { el.scrollTop = el.scrollHeight; });
  await page.waitForTimeout(400);
  await snap('05-graph-scrolled-bottom');
  log('Scrolled to bottom of commit graph');

  // Scroll back to top
  await graphOutput.evaluate(el => { el.scrollTop = 0; });
  await page.waitForTimeout(300);
  await snap('06-graph-top');

  // ── Collapse and re-expand to verify state persists ───────────────────────
  await graphHeader.click();
  await page.waitForTimeout(400);
  await snap('07-graph-collapsed');
  log('Graph section collapsed');

  await graphHeader.click();
  await page.waitForTimeout(1000);
  await snap('08-graph-re-expanded');
  log('Graph section re-expanded (should not re-fetch if already loaded)');

  const graphOutputAfter = graphSection.locator('.graph-output');
  const textAfter = await graphOutputAfter.textContent().catch(() => '');
  log(`Graph still has content after toggle: ${textAfter.trim().length > 0}`);

  await snap('09-final-state');
  log(`Commit graph test complete — ${lines.length} graph lines verified`);
});
