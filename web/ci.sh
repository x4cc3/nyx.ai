#!/bin/bash
set -e

echo "Starting CI checks..."

echo "----------------------------------------"
echo "1. Linting..."
npm run lint

echo "----------------------------------------"
echo "2. Testing..."
npm run test -- --runInBand --passWithNoTests

echo "----------------------------------------"
echo "3. Building..."
npm run build

echo "----------------------------------------"
echo "CI checks passed successfully."
