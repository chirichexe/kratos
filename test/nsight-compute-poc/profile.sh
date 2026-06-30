#!/bin/sh
set -eu

if [ "$#" -lt 2 ]; then
  echo "usage: profile.sh <workload> <report-path-without-extension> [workload-args...]" >&2
  exit 2
fi

workload="$1"
report_path="$2"
shift 2

mkdir -p "$(dirname "$report_path")"

echo "== nvidia-smi =="
nvidia-smi

echo "== ncu --version =="
ncu --version

echo "== ncu profiling runner =="
if [ -n "${NCU_KERNEL_NAME:-}" ]; then
  ncu --target-processes all \
    --set "${NCU_SET:-basic}" \
    --kernel-name "$NCU_KERNEL_NAME" \
    --launch-count "${NCU_LAUNCH_COUNT:-1}" \
    --force-overwrite \
    --export "$report_path" \
    "$workload" "$@"
else
  ncu --target-processes all \
    --set "${NCU_SET:-basic}" \
    --launch-count "${NCU_LAUNCH_COUNT:-1}" \
    --force-overwrite \
    --export "$report_path" \
    "$workload" "$@"
fi

echo "== ncu imported raw metrics =="
ncu --import "$report_path.ncu-rep" \
  --page raw \
  --print-units base

echo "== generated reports =="
ls -lh "$(dirname "$report_path")"
