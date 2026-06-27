const { test } = require('../fixtures');

test('09-gitlens-editor-actions', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Require git repo
  const gitBtn = page.locator('.activity-btn[title="Source Control"]');
  if (await gitBtn.count() === 0) {
    log('Not a git repo — skipping editor action tests');
    return;
  }

  // Open a tracked file with history (go.mod or any .go file)
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
    for (const row of allRows) {
      const hidden = await row.locator('.chevron-wrapper.hidden').count();
      if (hidden > 0) {
        const name = await row.locator('.node-name').textContent().catch(() => '');
        await row.click();
        await page.waitForTimeout(1200);
        log(`Opened ${name.trim()}`);
        fileOpened = true;
        break;
      }
    }
  }

  // Wait for editor-actions toolbar
  const editorActions = page.locator('.editor-actions');
  try {
    await editorActions.waitFor({ state: 'visible', timeout: 6000 });
    log('Editor actions toolbar ready');
  } catch {
    log('No editor actions toolbar — file may not be git-tracked');
    await snap('00-no-toolbar');
    return;
  }

  await snap('01-file-open-toolbar-ready');

  // ── Prev Rev button ───────────────────────────────────────────────────────
  const prevRevBtn = page.locator('.editor-actions button').filter({ hasText: 'Prev Rev' });
  if (await prevRevBtn.count() === 0) {
    log('Prev Rev button not found in toolbar');
    await snap('02-no-prev-rev-btn');
  } else {
    log('Clicking Prev Rev button');
    await prevRevBtn.click();
    await page.waitForTimeout(2500); // file history fetch + diff open
    await snap('02-prev-rev-clicked');

    // Check for diff editor
    const diffEditor = page.locator('.diff-editor-container-inner, .monaco-diff-editor');
    if (await diffEditor.count() > 0) {
      log('Diff editor opened for previous revision');
      await snap('03-prev-rev-diff-open');
    } else {
      log('Diff editor not found — may be at root commit or history unavailable');
      await snap('03-prev-rev-no-diff');
    }

    // Check for new diff tab in tabs bar
    const diffTab = page.locator('.tabs-bar .tab').filter({ hasText: '(Prev Rev)' });
    if (await diffTab.count() > 0) {
      log('Prev Rev diff tab visible in tab bar');
    }
  }

  // ── Line History ──────────────────────────────────────────────────────────
  // Navigate back to original file tab if we opened a diff
  const tabs = await page.locator('.tabs-bar .tab').all();
  for (const tab of tabs) {
    const tabText = await tab.textContent().catch(() => '');
    if (tabText.includes('go.mod') && !tabText.includes('Prev Rev') && !tabText.includes('↔')) {
      await tab.click();
      await page.waitForTimeout(500);
      log('Switched back to original file tab');
      break;
    }
  }

  await snap('04-back-to-file');

  // Click in editor to focus it
  const editorContent = page.locator('.monaco-editor .view-lines');
  if (await editorContent.count() > 0) {
    await editorContent.click({ position: { x: 50, y: 20 } });
    await page.waitForTimeout(300);

    // Select multiple lines using keyboard (Shift+Down x4)
    await page.keyboard.press('Home'); // go to start of line
    await page.keyboard.press('Shift+End'); // select to end of line
    await page.keyboard.press('Shift+ArrowDown');
    await page.keyboard.press('Shift+ArrowDown');
    await page.keyboard.press('Shift+ArrowDown');
    await page.waitForTimeout(600);
    log('Selected multiple lines in Monaco editor');
    await snap('05-multi-line-selection');

    // Check if Line History button appeared in toolbar
    const lineHistoryBtn = page.locator('.editor-actions button').filter({ hasText: 'Line History' });
    const btnVisible = await lineHistoryBtn.count() > 0;
    log(`Line History button visible: ${btnVisible}`);

    if (btnVisible) {
      await snap('06-line-history-button-visible');

      // Click Line History
      await lineHistoryBtn.click();
      await page.waitForTimeout(1500); // API call + panel open

      await snap('07-line-history-triggered');
      log('Line History triggered');

      // Git panel should now be open showing LINE HISTORY section
      const gitPanel = page.locator('.git-panel, .panel-section').filter({ hasText: 'LINE HISTORY' });
      if (await gitPanel.count() > 0) {
        log('Git panel opened with LINE HISTORY section');
        await snap('08-line-history-panel');

        // Check for commits in line history
        const lineHistorySection = page.locator('.panel-section').filter({ hasText: 'LINE HISTORY' });
        const commitItems = lineHistorySection.locator('.commit-summary, .commit-row');
        const count = await commitItems.count();
        log(`Line history commits: ${count}`);

        if (count > 0) {
          await commitItems.first().click();
          await page.waitForTimeout(1000);
          await snap('09-line-history-commit-opened');
          log('Opened commit from line history');
        }
      } else {
        log('LINE HISTORY section not found after trigger');
        await snap('08-line-history-section-missing');
      }
    } else {
      log('Line History button did not appear — selection may not have registered');
      await snap('06-line-history-btn-missing');
    }
  }

  await snap('10-final-state');
});
