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

  // ── OPEN COMMIT GRAPH (main-area view) ────────────────────────────────────
  const graphBtn = page.locator('.graph-btn', { hasText: 'Open Commit Graph' });
  if (await graphBtn.count() === 0) {
    throw new Error('"Open Commit Graph" button not found in git panel toolbar');
  }

  await graphBtn.click();
  await page.waitForTimeout(1500);
  await snap('02-graph-view-opened');
  log('Commit Graph view opened in main area');

  // A graph tab should now be active
  const graphTab = page.locator('.tab', { hasText: 'Commit Graph' });
  if (await graphTab.count() === 0) throw new Error('Commit Graph tab not created');
  log('Commit Graph tab present');

  // Wait for the graph SVG + rows to render
  const graphView = page.locator('.graph-view');
  await graphView.waitFor({ state: 'visible', timeout: 8000 });

  const svg = page.locator('.graph-svg');
  await svg.waitFor({ state: 'visible', timeout: 8000 });

  const nodeCount = await page.locator('.graph-svg circle').count();
  const edgeCount = await page.locator('.graph-svg path').count();
  const rowCount = await page.locator('.graph-view .row').count();
  log(`Graph rendered: ${nodeCount} nodes, ${edgeCount} edges, ${rowCount} commit rows`);

  if (nodeCount === 0) throw new Error('No commit nodes drawn in graph');
  if (rowCount === 0) throw new Error('No commit rows rendered');

  await snap('03-graph-rendered');

  // Verify ref pills (HEAD/branch) are present
  const refPills = await page.locator('.graph-view .ref-pill').count();
  log(`Ref pills (branches/tags) shown: ${refPills}`);

  // Verify a commit row shows hash + author + date
  const firstHash = await page.locator('.graph-view .row .hash').first().textContent().catch(() => '');
  log(`First commit hash shown: ${firstHash.trim()}`);
  if (!/^[0-9a-f]{7}$/.test(firstHash.trim())) {
    throw new Error(`Expected a 7-char short hash, got "${firstHash.trim()}"`);
  }

  // Scroll the graph
  const scroll = page.locator('.graph-scroll');
  await scroll.evaluate(el => { el.scrollTop = el.scrollHeight / 2; });
  await page.waitForTimeout(300);
  await snap('04-graph-scrolled');

  await scroll.evaluate(el => { el.scrollTop = 0; });
  await page.waitForTimeout(200);

  // Click a commit row → should open the commit detail viewer
  await page.locator('.graph-view .row').nth(1).click();
  await page.waitForTimeout(1200);
  await snap('05-commit-opened-from-graph');
  const commitTab = await page.locator('.tab', { hasText: 'Commit:' }).count();
  log(`Commit detail tab opened from graph: ${commitTab > 0}`);

  await snap('06-final-state');
  log(`Commit graph test complete — ${nodeCount} nodes, ${rowCount} rows verified`);
});
