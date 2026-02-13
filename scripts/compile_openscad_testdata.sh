#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IN_DIR="${ROOT}/testdata/openscad_scad"
OUT_DIR="${ROOT}/testdata/openscad_stl"

mkdir -p "${OUT_DIR}"

shopt -s nullglob
for scad_file in "${IN_DIR}"/*.scad; do
  base="$(basename "${scad_file}" .scad)"
  out_path="${OUT_DIR}/${base}.stl"
  openscad -o "${out_path}" "${scad_file}"
done
