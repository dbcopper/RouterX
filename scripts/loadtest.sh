#!/usr/bin/env bash
set -euo pipefail

if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 not found. Install from https://k6.io/docs/get-started/installation/"
  exit 1
fi

k6 run loadtest.k6.js
