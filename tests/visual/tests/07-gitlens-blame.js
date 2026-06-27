const { test } = require('../fixtures');

test('07-gitlens-blame', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git repo
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping blame tests');
    return;
  }

  // Open go.mod (small, tracked file at root level)
  const allRows = await page.locator('.node-row').all();
  let fileOpened = false;
  for (const row of allRows) {
    const name = await row.locator('.node-name').textContent().catch(() => '');
    if (name.trim() === 'go.mod') {
      await row.click();
      await page.waitForTimeout(1200);
      log('Opened go.mod');
      fileOpened = true;
      break;
    }
  }

  if (!fileOpened) {
    // Fallback: open any visible text file
    for (const row of allRows) {
      const hidden = await row.locator('.chevron-wrapper.hidden').count();
      if (hidden > 0) {
        const name = await row.locator('.node-name').textContent().catch(() => '');
        const ext = name.trim().split('.').pop();
        if (['go', 'js', 'md', 'mod', 'sum', 'svelte'].includes(ext)) {
          await row.click();
          await page.waitForTimeout(1200);
          log(`Opened fallback file: ${name.trim()}`);
          fileOpened = true;
          break;
        }
      }
    }
  }

  if (!fileOpened) {
    throw new Error('Could not open any text file for blame testing');
  }

  // Wait for editor-actions toolbar (only appears for tracked text files in git repos)
  const editorActions = page.locator('.editor-actions');
  try {
    await editorActions.waitFor({ state: 'visible', timeout: 6000 });
    log('Editor actions toolbar visible');
  } catch {
    log('Editor actions toolbar not visible — file may not be git-tracked or editor not mounted');
    await snap('00-no-toolbar');
    return;
  }

  await snap('01-file-open-with-toolbar');

  // Click in editor content to position cursor on line 1 → triggers status bar blame
  const editorContent = page.locator('.monaco-editor .view-lines');
  if (await editorContent.count() > 0) {
    await editorContent.click({ position: { x: 10, y: 10 } });
    await page.waitForTimeout(1500);
  }

  // Check status bar for blame info
  const statusBlame = page.locator('.status-blame');
  if (await statusBlame.count() > 0) {
    const blameText = await statusBlame.textContent().catch(() => '');
    log(`Status bar blame: "${blameText.trim().slice(0, 70)}"`);
  } else {
    log('Status bar blame not visible (may appear after blame loads)');
  }
  await snap('02-status-bar-blame');

  // ── Blame toggle ──────────────────────────────────────────────────────────
  const blameBtn = page.locator('.editor-actions button').filter({ hasText: 'Blame' });
  await blameBtn.click();
  await page.waitForTimeout(2000); // blame fetch + decorations
  log('All-line blame enabled');
  await snap('03-all-lines-blame');

  // Check that blame class appears in DOM
  const blameInline = page.locator('.monaco-git-blame-inline');
  const blameLineCount = await blameInline.count();
  log(`Inline blame decoration elements: ${blameLineCount}`);

  // Hover over a Monaco view-line to trigger blame popup
  const viewLines = page.locator('.monaco-editor .view-line');
  const lineCount = await viewLines.count();
  if (lineCount >= 3) {
    const targetLine = viewLines.nth(2);
    await targetLine.hover({ force: true });
    await page.waitForTimeout(1000);
    const popup = page.locator('.blame-popup');
    if (await popup.count() > 0) {
      log('Blame popup appeared on hover');
      await snap('04-blame-hover-popup');
    } else {
      log('Blame popup not triggered (may need cursor on blame text)');
      await snap('04-blame-hover-no-popup');
    }
  } else {
    log('Not enough visible lines for hover test');
    await snap('04-blame-too-few-lines');
  }

  // ── Heatmap toggle ───────────────────────────────────────────────────────
  const heatmapBtn = page.locator('.editor-actions button').filter({ hasText: 'Heatmap' });
  await heatmapBtn.click();
  await page.waitForTimeout(1000);
  log('Heatmap enabled');
  await snap('05-heatmap-active');

  // Report which heat classes are present
  for (const cls of ['heat-hot', 'heat-warm', 'heat-mild', 'heat-cool', 'heat-cold']) {
    const n = await page.locator(`.${cls}`).count();
    if (n > 0) { log(`Heatmap class .${cls}: ${n} lines`); break; }
  }

  // ── Code Lens toggle ─────────────────────────────────────────────────────
  const lensBtn = page.locator('.editor-actions button').filter({ hasText: 'Lens' });
  await lensBtn.click();
  await page.waitForTimeout(2000); // lens provider needs time
  log('Code lens enabled');
  await snap('06-code-lens-active');

  // Check for Monaco code lens DOM elements
  const codeLensElems = page.locator('.codelens-decoration');
  const lensCount = await codeLensElems.count();
  log(`Code lens decoration elements: ${lensCount}`);

  // ── All three active ──────────────────────────────────────────────────────
  await snap('07-blame-heatmap-lens-all-active');
  log('Blame + Heatmap + Code Lens simultaneously active');

  // ── Turn everything off ───────────────────────────────────────────────────
  await blameBtn.click();
  await page.waitForTimeout(300);
  await heatmapBtn.click();
  await page.waitForTimeout(300);
  await lensBtn.click();
  await page.waitForTimeout(500);
  await snap('08-all-features-disabled');
  log('All blame features disabled — editor back to normal');
});
