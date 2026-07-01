#!/bin/sh
set -eu

if [ "$#" -lt 2 ]; then
  echo "usage: profile.sh <workload> <report-path-without-extension> [workload-args...]" >&2
  exit 2
fi

workload="$1"
report_path="$2"
shift 2

experiment_name="${KRATOS_EXPERIMENT_NAME:-$(basename "$report_path")}"
namespace="${KRATOS_EXPERIMENT_NAMESPACE:-${POD_NAMESPACE:-}}"
if [ -z "$namespace" ] && [ -r /var/run/secrets/kubernetes.io/serviceaccount/namespace ]; then
  namespace="$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace)"
fi
namespace="${namespace:-default}"
summary_configmap="${KRATOS_PROFILE_SUMMARY_CONFIGMAP:-${experiment_name}-profile-summary}"
summary_key="${KRATOS_PROFILE_SUMMARY_KEY:-summary.json}"
ncu_csv="${KRATOS_NCU_CSV_PATH:-${report_path}.csv}"
ncu_raw="${KRATOS_NCU_RAW_PATH:-${report_path}.raw.txt}"
summary_path="${KRATOS_PROFILE_SUMMARY_PATH:-/tmp/summary.json}"

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
  --print-units base \
  > "$ncu_raw"
cat "$ncu_raw"

echo "== ncu export csv =="
ncu --import "$report_path.ncu-rep" \
  --page raw \
  --csv \
  > "$ncu_csv"

echo "== parse ncu summary =="
python3 /scripts/parse_ncu.py \
  --input "$ncu_raw" \
  --output "$summary_path"
cat "$summary_path"

echo "== publish profile summary configmap =="
kubectl create configmap "$summary_configmap" \
  "--from-file=${summary_key}=${summary_path}" \
  -n "$namespace" \
  --dry-run=client \
  -o yaml | kubectl apply -f -

echo "== generated reports =="
ls -lh "$(dirname "$report_path")"
