const { test: base } = require('@playwright/test');

// Runs in the page on every navigation. Draws a visible cursor that follows the
// real mouse and pulses on click, so the recorded video/screenshots show *where*
// each interaction happens instead of elements "magically" reacting.
function mouseHelper() {
  const attach = () => {
    if (document.getElementById('pw-cursor')) return;
    const style = document.createElement('style');
    style.textContent = `
      #pw-cursor {
        position: fixed; top: 0; left: 0; width: 20px; height: 20px;
        margin: -10px 0 0 -10px; border-radius: 50%;
        background: rgba(99,102,241,.35); border: 2px solid #6366f1;
        box-shadow: 0 0 8px rgba(99,102,241,.6);
        pointer-events: none; z-index: 2147483647;
        transition: width .1s, height .1s, margin .1s, background .1s;
      }
      #pw-cursor.click {
        width: 34px; height: 34px; margin: -17px 0 0 -17px;
        background: rgba(99,102,241,.15);
      }`;
    const dot = document.createElement('div');
    dot.id = 'pw-cursor';
    document.head.appendChild(style);
    document.body.appendChild(dot);
    document.addEventListener('mousemove', (e) => {
      dot.style.left = e.clientX + 'px';
      dot.style.top = e.clientY + 'px';
    }, true);
    document.addEventListener('mousedown', () => dot.classList.add('click'), true);
    document.addEventListener('mouseup', () => dot.classList.remove('click'), true);
  };
  if (document.body) attach();
  else window.addEventListener('DOMContentLoaded', attach);
}

exports.test = base.extend({
  baseUrl: [async ({}, use) => {
    await use('http://localhost:3009');
  }, { scope: 'test' }],

  // Override the built-in page to inject the visible cursor on every load.
  page: async ({ page }, use) => {
    await page.addInitScript(mouseHelper);
    await use(page);
  },

  snap: [async ({ page }, use, testInfo) => {
    await use(async (name) => {
      // Write to the test's output dir (under /output/test-results/<test>/) so the
      // file persists and the attachment carries a real `path` for the reporter.
      const file = testInfo.outputPath(`${name}.png`);
      await page.screenshot({ path: file });
      await testInfo.attach(name, { path: file, contentType: 'image/png' });
    });
  }, { scope: 'test' }],

  log: [async ({}, use, testInfo) => {
    const entries = [];
    await use((msg) => {
      entries.push(msg);
      console.log(`  ${msg}`);
    });
    if (entries.length > 0) {
      await testInfo.attach('logs', {
        body: Buffer.from(entries.join('\n')),
        contentType: 'text/plain'
      });
    }
  }, { scope: 'test' }]
});
