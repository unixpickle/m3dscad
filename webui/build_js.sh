#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${ROOT}"
npm install
npm run build

echo "Built webui/main.bundle.js"
