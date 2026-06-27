const path = require('path');

const OUTPUT_DIR = process.env.RESULTS_DIR || path.join(__dirname, 'results', 'latest');

module.exports = {
  testDir: path.join(__dirname, 'tests'),
  testMatch: '**/*.js',
  outputDir: path.join(OUTPUT_DIR, 'test-results'),
  reporter: [
    ['html', { outputFolder: path.join(OUTPUT_DIR, 'playwright-report'), open: 'never' }],
    [path.join(__dirname, 'custom-reporter.js'), { outputDir: OUTPUT_DIR }]
  ],
  use: {
    baseURL: 'http://localhost:3009',
    trace: 'on',
    video: { mode: 'on', size: { width: 1280, height: 720 } },
    screenshot: 'on',
    viewport: { width: 1280, height: 720 }
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
  workers: 1,
  timeout: 60000,
  retries: 0
};
