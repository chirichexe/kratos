#!/usr/bin/env bash
set -euo pipefail

REPORT_DIR="${REPORT_DIR:-/tmp/nsight-compute}"
REPORT_BASENAME="${REPORT_BASENAME:-vectoradd}"
REPORT_PATH="${REPORT_DIR}/${REPORT_BASENAME}"

mkdir -p "${REPORT_DIR}"

echo "== nvidia-smi =="
nvidia-smi

echo "== ncu --version =="
ncu --version

echo "== workload smoke test =="
/usr/local/bin/vectoradd

echo "== ncu profile =="
ncu \
  --target-processes all \
  --set basic \
  --kernel-name regex:vectorAdd \
  --launch-count 1 \
  --force-overwrite \
  --export "${REPORT_PATH}" \
  /usr/local/bin/vectoradd

echo "== ncu imported raw metrics =="
ncu \
  --import "${REPORT_PATH}.ncu-rep" \
  --page raw \
  --print-units base

echo "== generated reports =="
ls -lh "${REPORT_DIR}"

echo "Nsight Compute report: ${REPORT_PATH}.ncu-rep"
