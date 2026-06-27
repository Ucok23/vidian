const fs = require('fs');
const path = require('path');
const { test } = require('../fixtures');

test('05-csv-viewer', async ({ page, baseUrl, snap, log }) => {
  // Create a test CSV file in the workspace
  const csvContent = 'Name,Age,City,Role,Salary\nAlice,30,New York,Engineer,120000\nBob,25,San Francisco,Designer,95000\nCharlie,35,"Los Angeles",Manager,140000\nDiana,28,"Chicago, IL",Analyst,85000\nEve,32,Seattle,"Senior Engineer",150000\nFrank,29,Austin,DevOps,110000\nGrace,31,Boston,"Data Scientist",130000';

  // Write CSV via the filesystem (mounted in docker)
  const csvPath = '/tests/test-data/test.csv';
  fs.mkdirSync(path.dirname(csvPath), { recursive: true });
  fs.writeFileSync(csvPath, csvContent);

  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  await snap('01-before-csv');

  // Navigate to tests > visual > test-data > test.csv
  const folders = ['tests', 'visual', 'test-data'];
  for (const folder of folders) {
    const rows = await page.locator('.node-row').all();
    for (const row of rows) {
      const name = await row.locator('.node-name').textContent();
      if (name.trim() === folder) {
        await row.click();
        await page.waitForTimeout(400);
        break;
      }
    }
  }

  // Click test.csv
  const rows = await page.locator('.node-row').all();
  for (const row of rows) {
    const name = await row.locator('.node-name').textContent();
    if (name.trim() === 'test.csv') {
      await row.click();
      await page.waitForTimeout(1000);
      break;
    }
  }

  await snap('02-csv-table-view');
  log('CSV file opened in table view');

  // Verify table rendered
  const headerCount = await page.locator('.csv-viewer th').count();
  const rowCount = await page.locator('.csv-viewer tbody tr').count();
  log(`CSV table: ${headerCount - 1} columns, ${rowCount} rows`);

  if (headerCount < 2) throw new Error('CSV table headers not rendered');
  if (rowCount < 1) throw new Error('CSV table rows not rendered');

  // Verify toolbar shows stats
  const info = await page.locator('.csv-info').textContent();
  log(`CSV toolbar info: ${info}`);

  await snap('03-csv-final');
});
