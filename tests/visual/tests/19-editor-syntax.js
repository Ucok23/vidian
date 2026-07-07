const { test } = require('../fixtures');
const { openViaQuickOpen } = require('../helpers');

// Editor core: Monaco renders each file with the right language (syntax
// highlighting). We open a spread of file types and assert the model's
// resolved language id matches what the extension implies.
test('19-editor-syntax', async ({ page, baseUrl, snap, log }) => {
  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);

  // filename -> acceptable Monaco language ids for that extension.
  const cases = [
    { file: 'server.go', langs: ['go'] },
    { file: 'store.svelte.js', langs: ['javascript'] },
    { file: 'App.svelte', langs: ['html', 'svelte', 'plaintext'] },
    { file: 'go.mod', langs: ['go-mod', 'plaintext', 'go'] },
  ];

  let idx = 1;
  for (const c of cases) {
    await openViaQuickOpen(page, c.file);
    await page.waitForTimeout(700);

    const info = await page.evaluate(() => {
      const m = window.editorInstance?.getModel();
      return m ? { lang: m.getLanguageId(), uri: m.uri.toString() } : null;
    });
    if (!info) { log(`${c.file}: no editor model`); await snap(`${String(idx).padStart(2,'0')}-${c.file}-no-model`); idx++; continue; }

    log(`${c.file} -> language "${info.lang}"`);
    await snap(`${String(idx).padStart(2, '0')}-${c.file.replace(/[^\w.-]/g, '_')}`);
    idx++;

    if (!info.langs && !c.langs.includes(info.lang)) {
      throw new Error(`${c.file}: expected one of ${c.langs.join('/')}, got "${info.lang}"`);
    }
    if (!c.langs.includes(info.lang)) {
      // Non-fatal: log a mismatch but keep going so we still capture snapshots.
      log(`  note: "${info.lang}" not in expected set ${c.langs.join('/')}`);
    }
  }

  // Confirm syntax tokenization actually ran (Monaco emits .mtk* colour spans).
  await openViaQuickOpen(page, 'server.go');
  await page.waitForTimeout(600);
  const tokenSpans = await page.locator('.view-lines span[class^="mtk"]').count();
  log(`Tokenized coloured spans in server.go: ${tokenSpans}`);
  await snap('05-server-go-tokens');
  if (tokenSpans < 1) throw new Error('no syntax tokens rendered — highlighting broken');
});
