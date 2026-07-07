const { test } = require('../fixtures');

// App shell: sidebar collapse/expand (Ctrl+B), activity-bar view switching,
// and dragging the resize handle to change sidebar width.
test('27-app-shell', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  const sidebar = page.locator('.sidebar');
  await sidebar.waitFor({ state: 'visible', timeout: 8000 });
  await snap('01-sidebar-visible');

  // --- Ctrl+B collapses the sidebar ---
  await page.keyboard.press('Control+b');
  await page.waitForTimeout(400);
  const collapsed = await sidebar.count();
  log(`After Ctrl+B, .sidebar elements: ${collapsed}`);
  await snap('02-sidebar-collapsed');
  if (collapsed > 0) throw new Error('sidebar did not collapse on Ctrl+B');

  // --- Ctrl+B again restores it ---
  await page.keyboard.press('Control+b');
  await page.waitForTimeout(400);
  await sidebar.waitFor({ state: 'visible', timeout: 5000 });
  await snap('03-sidebar-restored');
  log('Sidebar restored');

  // --- Activity-bar view switching ---
  const views = [
    { title: 'Search (Ctrl+Shift+F)', probe: '.search-box, .search-panel, .sidebar' },
    { title: 'Source Control', probe: '.sidebar' },
    { title: 'Explorer (Ctrl+B)', probe: '.node-row' },
  ];
  let i = 4;
  for (const v of views) {
    const btn = page.locator(`.activity-btn[title="${v.title}"]`);
    if (await btn.count() === 0) { log(`activity button "${v.title}" not present, skipping`); continue; }
    await btn.click();
    await page.waitForTimeout(500);
    const active = await page.locator('.activity-btn.active').count();
    log(`Clicked "${v.title}" -> active buttons: ${active}`);
    await snap(`${String(i).padStart(2, '0')}-view-${v.title.split(' ')[0].toLowerCase()}`);
    i++;
  }

  // --- Resize handle drag ---
  const widthBefore = await page.evaluate(() => {
    const el = document.querySelector('.sidebar');
    return el ? el.getBoundingClientRect().width : 0;
  });
  const handle = page.locator('.resize-handle');
  if (await handle.count() > 0) {
    const box = await handle.boundingBox();
    if (box) {
      await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
      await page.mouse.down();
      await page.mouse.move(box.x + 120, box.y + box.height / 2, { steps: 8 });
      await page.mouse.up();
      await page.waitForTimeout(400);
    }
  }
  const widthAfter = await page.evaluate(() => {
    const el = document.querySelector('.sidebar');
    return el ? el.getBoundingClientRect().width : 0;
  });
  log(`Sidebar width: ${Math.round(widthBefore)}px -> ${Math.round(widthAfter)}px`);
  await snap(`${String(i).padStart(2, '0')}-after-resize`);
  if (Math.abs(widthAfter - widthBefore) < 10) {
    throw new Error(`resize handle drag did not change width (${widthBefore} -> ${widthAfter})`);
  }
  log('Resize handle works');
});
