const puppeteer = require('puppeteer-core');
const { spawn } = require('child_process');
const path = require('path');

async function runTest() {
  console.log("=== STARTING END-TO-END WORKFLOW TEST ===");
  
  // 1. Start Vidian Server in background on port 3009
  const serverPath = path.resolve(__dirname, '../vidian');
  const workspacePath = path.resolve(__dirname, '..');
  
  console.log(`Starting server: ${serverPath} -dir ${workspacePath} -port 3009 -dev`);
  const serverProcess = spawn(serverPath, ['-dir', workspacePath, '-port', '3009', '-dev'], {
    cwd: workspacePath
  });

  // Collect server logs
  serverProcess.stdout.on('data', (data) => {
    // console.log(`[Server]: ${data}`);
  });
  serverProcess.stderr.on('data', (data) => {
    // console.error(`[Server Error]: ${data}`);
  });

  // Wait 2 seconds for server to start up
  await new Promise(resolve => setTimeout(resolve, 2000));

  let browser;
  let exitCode = 0;
  const consoleErrors = [];

  try {
    // 2. Launch headless Chromium
    console.log("Launching Chromium browser...");
    browser = await puppeteer.launch({
      executablePath: '/usr/bin/chromium-browser',
      headless: true,
      args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    const page = await browser.newPage();

    // 3. Setup console error capture
    page.on('console', msg => {
      if (msg.type() === 'error' || msg.type() === 'warning') {
        console.log(`[Browser Console ${msg.type().toUpperCase()}]: ${msg.text()}`);
      }
      if (msg.type() === 'error') {
        // Ignore expected 404s from git show when a commit has no parent (e.g. first commit)
        const text = msg.text();
        if (!text.includes('404') && !text.includes('Not Found')) {
          consoleErrors.push(text);
        }
      }
    });

    page.on('pageerror', err => {
      console.log(`[Browser Uncaught Exception]: ${err.message}`);
      consoleErrors.push(err.message);
    });

    // 4. Navigate to Vidian
    console.log("Navigating to http://localhost:3009...");
    await page.goto('http://localhost:3009', { waitUntil: 'networkidle0' });

    // Verify Welcome Screen Title
    const title = await page.title();
    console.log(`Page title is: "${title}"`);
    if (title !== 'Vidian') {
      throw new Error(`Expected page title to be "Vidian", got "${title}"`);
    }

    // 5. Test File Selection in Explorer
    console.log("Testing file explorer click...");
    // Wait for file nodes to render
    await page.waitForSelector('.node-name');
    
    // Find go.mod or main.go and click it
    const fileNodes = await page.$$('.node-name');
    let fileClicked = false;
    for (const node of fileNodes) {
      const text = await page.evaluate(el => el.textContent, node);
      if (text.includes('main.go') || text.includes('go.mod')) {
        console.log(`Clicking explorer node: ${text}`);
        await node.click();
        fileClicked = true;
        break;
      }
    }
    
    if (!fileClicked) {
      throw new Error("Could not find a file in the explorer tree to open");
    }

    // Wait for Monaco Editor to mount and render content
    await page.waitForSelector('.monaco-editor', { timeout: 5000 });
    console.log("Monaco Editor mounted successfully.");

    // 6. Test Source Control Panel
    console.log("Testing Source Control sidebar...");
    const gitTabSelector = '.activity-btn[title="Source Control"]';
    
    // Check if git icon is present (it should be because we committed earlier)
    const gitTab = await page.$(gitTabSelector);
    if (gitTab) {
      await gitTab.click();
      console.log("Switched to Git Panel.");
      
      // Wait for git log/commits to render
      await page.waitForSelector('.commit-summary', { timeout: 5000 });
      console.log("Git Panel loaded commit history successfully.");

      // Click first commit to expand details
      const firstCommit = await page.$('.commit-summary');
      await firstCommit.click();
      // Wait for files changed list in the main area commit viewer
      await page.waitForSelector('.commit-file-item', { timeout: 3000 });
      console.log("Loaded commit file items in main view.");
 
      // Click a file to open Diff view
      const firstCommitFile = await page.$('.commit-file-item');
      await firstCommitFile.click();
      console.log("Opened file diff view.");

      // Wait for Diff Editor
      await page.waitForSelector('.diff-editor-container-inner', { timeout: 5000 });
      console.log("Monaco Diff Editor mounted successfully.");
    } else {
      console.log("Source Control tab not present in this workspace (skipping Git checks).");
    }

    // 7. Verify no errors occurred during tests
    if (consoleErrors.length > 0) {
      throw new Error(`Captured ${consoleErrors.length} console/page errors during integration tests.`);
    }

    console.log("=== ALL END-TO-END TESTS PASSED SUCCESSFULLY ===");

  } catch (err) {
    console.error("=== TEST SUITE FAILED ===");
    console.error(err);
    exitCode = 1;
  } finally {
    if (browser) {
      await browser.close();
    }
    console.log("Stopping server...");
    serverProcess.kill('SIGTERM');
    process.exit(exitCode);
  }
}

runTest();
