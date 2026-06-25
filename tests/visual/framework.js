const { chromium } = require('playwright');
const fs = require('fs');
const path = require('path');

const BASE_URL = 'http://localhost:3009';
const RESULTS_DIR = process.env.RESULTS_DIR || '/output';
const TESTS_DIR = path.join(__dirname, 'tests');

async function run() {
  const testFiles = fs.readdirSync(TESTS_DIR)
    .filter(f => f.endsWith('.js'))
    .sort();

  console.log(`Found ${testFiles.length} visual tests\n`);

  const summary = [];
  const browser = await chromium.launch({ headless: true });

  for (const file of testFiles) {
    const testName = file.replace('.js', '');
    const testOutputDir = path.join(RESULTS_DIR, testName);
    fs.mkdirSync(testOutputDir, { recursive: true });

    const videoDir = path.join(testOutputDir, 'video-tmp');
    fs.mkdirSync(videoDir, { recursive: true });

    console.log(`━━━ ${testName} ━━━`);
    const logs = [];

    const context = await browser.newContext({
      viewport: { width: 1280, height: 720 },
      recordVideo: { dir: videoDir, size: { width: 1280, height: 720 } }
    });

    const page = await context.newPage();

    const helpers = {
      page,
      baseUrl: BASE_URL,
      screenshot: async (name) => {
        const filepath = path.join(testOutputDir, `${name}.png`);
        await page.screenshot({ path: filepath });
        console.log(`  screenshot: ${name}.png`);
      },
      log: (msg) => {
        console.log(`  ${msg}`);
        logs.push(msg);
      }
    };

    let passed = true;
    let error = null;

    try {
      const testFn = require(path.join(TESTS_DIR, file));
      await testFn(helpers);
    } catch (err) {
      passed = false;
      error = err;
      console.error(`  FAILED: ${err.message}`);
    }

    await page.close();
    await context.close();

    // Move video file to test output dir
    const videoFiles = fs.readdirSync(videoDir);
    if (videoFiles.length > 0) {
      const src = path.join(videoDir, videoFiles[0]);
      const dest = path.join(testOutputDir, 'video.webm');
      fs.renameSync(src, dest);
      console.log(`  video: video.webm`);
    }
    fs.rmSync(videoDir, { recursive: true, force: true });

    // Write result file
    const resultLines = [
      `Test: ${testName}`,
      `Status: ${passed ? 'PASS' : 'FAIL'}`,
      '',
      '--- Log ---',
      ...logs
    ];
    if (error) {
      resultLines.push('', '--- Error ---', error.stack || error.message);
    }
    fs.writeFileSync(path.join(testOutputDir, 'result.txt'), resultLines.join('\n'));

    summary.push({ name: testName, passed, error: error?.message });
    console.log(`  result: ${passed ? 'PASS' : 'FAIL'}\n`);
  }

  await browser.close();

  // Write summary
  const summaryLines = [
    `Visual Test Run — ${new Date().toISOString()}`,
    `Total: ${summary.length}  Passed: ${summary.filter(s => s.passed).length}  Failed: ${summary.filter(s => !s.passed).length}`,
    '',
    ...summary.map(s => `  ${s.passed ? 'PASS' : 'FAIL'}  ${s.name}${s.error ? ` — ${s.error}` : ''}`)
  ];
  fs.writeFileSync(path.join(RESULTS_DIR, 'summary.txt'), summaryLines.join('\n'));
  console.log('━━━ Summary ━━━');
  summaryLines.forEach(l => console.log(l));

  const failed = summary.filter(s => !s.passed).length;
  process.exit(failed > 0 ? 1 : 0);
}

run().catch(err => {
  console.error('Framework error:', err);
  process.exit(1);
});
