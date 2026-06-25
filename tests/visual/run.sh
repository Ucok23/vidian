#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
RESULTS_DIR="$SCRIPT_DIR/results/$TIMESTAMP"
PORT=3009
DOCKER_IMAGE="mcr.microsoft.com/playwright:v1.53.0-noble"

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

# 3. Run tests in Docker
echo "→ Running visual tests in Docker..."
docker run --rm \
  --network host \
  -v "$SCRIPT_DIR:/tests" \
  -v "$RESULTS_DIR:/output" \
  -w /tmp/pw \
  "$DOCKER_IMAGE" \
  bash -c "
    npm init -y --silent 2>/dev/null
    npm install playwright@1.53.0 --silent 2>/dev/null
    NODE_PATH=/tmp/pw/node_modules node /tests/framework.js
  "

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
