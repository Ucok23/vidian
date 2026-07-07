const { test } = require('../fixtures');
const { openViaQuickOpen } = require('../helpers');

// Markdown preview: opening a .md file auto-renders the preview pane; the
// floating toggle switches to the raw Monaco source and back.
test('21-markdown-preview', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  await openViaQuickOpen(page, 'README.md');
  await page.waitForTimeout(800);

  // Preview pane should be visible with rendered HTML (markdown-body).
  const pane = page.locator('.preview-pane-full');
  await pane.waitFor({ state: 'visible', timeout: 8000 });
  const body = page.locator('.preview-pane-full .markdown-body');
  await body.waitFor({ state: 'visible', timeout: 8000 });
  await snap('01-preview-rendered');

  const headings = await page.locator('.preview-pane-full .markdown-body h1, .preview-pane-full .markdown-body h2').count();
  log(`Rendered markdown headings: ${headings}`);
  if (headings < 1) throw new Error('markdown preview rendered no headings');

  // Toggle to raw source: Monaco split container becomes visible.
  const toggle = page.locator('.preview-toggle-btn');
  await toggle.waitFor({ state: 'visible', timeout: 5000 });
  await toggle.click();
  await page.waitForTimeout(700);
  await snap('02-raw-source');

  const monacoVisible = await page.locator('.editor-split-container .monaco-container .view-lines').count();
  const paneGone = await page.locator('.preview-pane-full').count();
  log(`After toggle -> monaco visible spans: ${monacoVisible}, preview panes: ${paneGone}`);
  if (paneGone > 0) throw new Error('preview pane still shown after switching to raw');

  // Toggle back to preview.
  await toggle.click();
  await page.waitForTimeout(700);
  await snap('03-back-to-preview');
  const paneAgain = await page.locator('.preview-pane-full .markdown-body').count();
  log(`Back to preview -> markdown-body present: ${paneAgain}`);
  if (paneAgain < 1) throw new Error('preview did not restore after toggling back');
});
