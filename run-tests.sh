#!/bin/bash
set -e

# Clear any previous build artifacts to ensure clean test
echo "=== 1. Building Frontend ==="
cd frontend
npm run build
cd ..

echo "=== 2. Compiling Go Backend ==="
go build -o vidian cmd/vidian/main.go

echo "=== 3. Running End-to-End Workflow Tests ==="
cd frontend
npm run test:e2e
cd ..

echo "=== ALL WORKFLOW TESTS PASSED SUCCESSFULLY ==="
