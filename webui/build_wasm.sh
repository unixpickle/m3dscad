#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOROOT="$(go env GOROOT)"

cp "${GOROOT}/misc/wasm/wasm_exec.js" "${ROOT}/webui/wasm_exec.js"
(cd "${ROOT}" && GOOS=js GOARCH=wasm go build -o "${ROOT}/webui/m3dscad.wasm" ./webui/wasm)

echo "Built webui/m3dscad.wasm"
