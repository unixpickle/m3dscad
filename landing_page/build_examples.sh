#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLES_DIR="${ROOT}/landing_page/examples"
OUT_DIR="${ROOT}/landing_page/assets/examples"

mkdir -p "${OUT_DIR}"

for in_file in "${EXAMPLES_DIR}"/*.scad; do
  base_name="$(basename "${in_file}" .scad)"
  out_file="${OUT_DIR}/${base_name}.stl"
  delta="0.05"
  echo "[build_examples] ${base_name}.scad -> ${base_name}.stl (delta=${delta})"
  (
    cd "${ROOT}"
    go run ./cmd/m3dscad -in "${in_file}" -out "${out_file}" -delta "${delta}"
  )
done

echo "[build_examples] wrote STLs to ${OUT_DIR}"
