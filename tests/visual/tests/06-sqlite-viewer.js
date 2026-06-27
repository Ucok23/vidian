const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const { test } = require('../fixtures');

test('06-sqlite-viewer', async ({ page, baseUrl, snap, log }) => {
  // Create a test SQLite database in the workspace
  const dbPath = '/tests/test-data/test.db';
  fs.mkdirSync(path.dirname(dbPath), { recursive: true });

  // Use python3 to create the db (available in playwright image)
  try {
    execSync(`python3 -c "
import sqlite3
conn = sqlite3.connect('${dbPath}')
c = conn.cursor()
c.execute('CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, email TEXT, age INTEGER)')
c.execute('CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY, name TEXT, price REAL, stock INTEGER)')
c.execute('CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY, user_id INTEGER, product_id INTEGER, quantity INTEGER, total REAL)')
c.execute('DELETE FROM users')
c.execute('DELETE FROM products')
c.execute('DELETE FROM orders')
users = [(1,'Alice','alice@example.com',30),(2,'Bob','bob@example.com',25),(3,'Charlie','charlie@example.com',35),(4,'Diana','diana@example.com',28),(5,'Eve','eve@example.com',32)]
c.executemany('INSERT INTO users VALUES (?,?,?,?)', users)
products = [(1,'Widget',9.99,100),(2,'Gadget',19.99,50),(3,'Doohickey',29.99,25),(4,'Thingamajig',49.99,10)]
c.executemany('INSERT INTO products VALUES (?,?,?,?)', products)
orders = [(1,1,1,2,19.98),(2,1,2,1,19.99),(3,2,1,3,29.97),(4,3,3,1,29.99),(5,4,4,2,99.98),(6,5,2,1,19.99)]
c.executemany('INSERT INTO orders VALUES (?,?,?,?,?)', orders)
conn.commit()
conn.close()
"`, { stdio: 'pipe' });
  } catch (err) {
    log('Failed to create test.db with python3, trying sqlite3 CLI...');
    throw err;
  }

  await page.goto(baseUrl, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  await snap('01-before-sqlite');

  // Navigate to tests > visual > test-data > test.db
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

  // Click test.db
  let rows = await page.locator('.node-row').all();
  for (const row of rows) {
    const name = await row.locator('.node-name').textContent();
    if (name.trim() === 'test.db') {
      await row.click();
      await page.waitForTimeout(1500);
      break;
    }
  }

  await snap('02-sqlite-first-table');
  log('SQLite database opened');

  // Verify table sidebar rendered
  const tableItems = await page.locator('.table-item').all();
  const tableNames = [];
  for (const item of tableItems) {
    tableNames.push((await item.textContent()).trim());
  }
  log(`Tables found: ${tableNames.join(', ')}`);
  if (tableItems.length < 2) throw new Error('Expected at least 2 tables in sidebar');

  // Verify data table rendered
  const colCount = await page.locator('.sqlite-main th').count();
  const rowCount = await page.locator('.sqlite-main tbody tr').count();
  log(`First table: ${colCount - 1} columns, ${rowCount} rows`);

  // Click "products" table
  for (const item of tableItems) {
    const text = await item.textContent();
    if (text.trim() === 'products') {
      await item.click();
      await page.waitForTimeout(800);
      break;
    }
  }

  await snap('03-sqlite-products-table');
  log('Switched to products table');

  // Click "users" table
  for (const item of tableItems) {
    const text = await item.textContent();
    if (text.trim() === 'users') {
      await item.click();
      await page.waitForTimeout(800);
      break;
    }
  }

  await snap('04-sqlite-users-table');
  log('Switched to users table');

  await snap('05-sqlite-final');
});
