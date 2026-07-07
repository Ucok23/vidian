#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
RESULTS_DIR="$SCRIPT_DIR/results/$TIMESTAMP"
PORT=3009
DOCKER_IMAGE="mcr.microsoft.com/playwright:v1.53.0-noble"
# Keep this in lock-step with the Docker image tag above: the test runner we
# mount must match the browser build baked into the image.
PLAYWRIGHT_VERSION=1.53.0

mkdir -p "$RESULTS_DIR"

echo "━━━ Vidian Visual Tests ━━━"
echo "  Results: $RESULTS_DIR"
echo ""

# 1. Build
echo "→ Building frontend..."
cd "$PROJECT_DIR/frontend" && npm run build --silent
cd "$PROJECT_DIR"

echo "→ Compiling Go binary..."
go build -o vidian ./cmd/vidian

# 2. Start server
echo "→ Starting server on port $PORT..."
./vidian -dir "$PROJECT_DIR" -port "$PORT" -dev &
SERVER_PID=$!

cleanup() {
  echo "→ Stopping server (pid $SERVER_PID)..."
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
  # Clean up test data created during tests
  rm -rf "$SCRIPT_DIR/test-data" 2>/dev/null || true
}
trap cleanup EXIT

# Wait for server
for i in $(seq 1 20); do
  if curl -s -o /dev/null http://localhost:$PORT/ 2>/dev/null; then
    break
  fi
  sleep 0.5
done

if ! curl -s -o /dev/null http://localhost:$PORT/ 2>/dev/null; then
  echo "ERROR: Server failed to start on port $PORT"
  exit 1
fi
echo "  Server ready."
echo ""

# 3. Ensure the Playwright test runner is installed locally (once). We mount this
# into the container and run it directly. The browser binaries come from the
# Docker image; the test runner comes from here. Installing a *second* copy
# inside the container makes Playwright see two @playwright/test instances and
# silently collect zero tests, so we deliberately do NOT do that.
if [ ! -x "$SCRIPT_DIR/node_modules/.bin/playwright" ]; then
  echo "→ Installing Playwright test runner (@playwright/test@$PLAYWRIGHT_VERSION)..."
  ( cd "$SCRIPT_DIR" \
      && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 npm install --silent --no-save "@playwright/test@$PLAYWRIGHT_VERSION" )
fi

# 4. Run tests in Docker using the mounted runner + the image's browsers.
echo "→ Running visual tests in Docker..."
docker run --rm \
  --network host \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp \
  -v "$SCRIPT_DIR:/tests" \
  -v "$RESULTS_DIR:/output" \
  -w /tests \
  "$DOCKER_IMAGE" \
  bash -c "RESULTS_DIR=/output ./node_modules/.bin/playwright test --config /tests/playwright.config.js"

EXIT_CODE=$?

echo ""
echo "━━━ Done ━━━"
echo "  Results: $RESULTS_DIR"
echo ""

# List output
find "$RESULTS_DIR" -type f | sort | while read f; do
  echo "  ${f#$RESULTS_DIR/}"
done

exit $EXIT_CODE
