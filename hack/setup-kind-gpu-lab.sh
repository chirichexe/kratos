#!/usr/bin/env bash
set -Eeuo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-kratos-gpu}"
GPU_REPLICAS="${GPU_REPLICAS:-4}"
LOG_DIR="${LOG_DIR:-./tmp}"
LOG_FILE="${LOG_FILE:-${LOG_DIR}/kind-gpu-lab-$(date +%Y%m%d-%H%M%S).log}"
KIND_CONFIG="${KIND_CONFIG:-${LOG_DIR}/kind-gpu.yaml}"
TIME_SLICING_CONFIG="${TIME_SLICING_CONFIG:-${LOG_DIR}/time-slicing-config.yaml}"
BIN_DIR="${BIN_DIR:-${LOG_DIR}/bin}"
HELM_VERSION="${HELM_VERSION:-v3.15.4}"
INSTALL_HELM_IF_MISSING="${INSTALL_HELM_IF_MISSING:-true}"
KIND_CREATE_TIMEOUT="${KIND_CREATE_TIMEOUT:-15m}"

mkdir -p "$LOG_DIR"
exec > >(tee -a "$LOG_FILE") 2>&1
export PS4='+ $(date "+%Y-%m-%dT%H:%M:%S%z") ${BASH_SOURCE##*/}:${LINENO}: '
set -x

mkdir -p "$BIN_DIR"
export PATH="${BIN_DIR}:${PATH}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1
}

install_helm() {
  local arch os archive url
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"

  case "$arch" in
    x86_64 | amd64) arch="amd64" ;;
    aarch64 | arm64) arch="arm64" ;;
    *)
      printf 'Unsupported architecture for Helm install: %s\n' "$arch" >&2
      return 1
      ;;
  esac

  archive="${LOG_DIR}/helm-${HELM_VERSION}-${os}-${arch}.tar.gz"
  url="https://get.helm.sh/helm-${HELM_VERSION}-${os}-${arch}.tar.gz"

  curl -fsSL "$url" -o "$archive"
  tar -xzf "$archive" -C "$LOG_DIR"
  install -m 0755 "${LOG_DIR}/${os}-${arch}/helm" "${BIN_DIR}/helm"
  helm version
}

if ! require_cmd helm && [[ "$INSTALL_HELM_IF_MISSING" == "true" ]]; then
  install_helm
fi

missing=()
for cmd in docker kind kubectl helm nvidia-smi timeout; do
  if ! require_cmd "$cmd"; then
    missing+=("$cmd")
  fi
done

if ((${#missing[@]} > 0)); then
  printf 'Missing required commands: %s\n' "${missing[*]}" >&2
  exit 1
fi

docker info >/dev/null
nvidia-smi

cat >"$KIND_CONFIG" <<YAML
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
  - role: control-plane
  - role: worker
    labels:
      kratos.io/gpu-simulated-node: "true"
  - role: worker
    labels:
      kratos.io/gpu-simulated-node: "true"
  - role: worker
    labels:
      kratos.io/gpu-simulated-node: "true"
YAML

if ! kind get clusters | grep -Fxq "$CLUSTER_NAME"; then
  timeout "$KIND_CREATE_TIMEOUT" kind create cluster --config "$KIND_CONFIG"
fi

kubectl cluster-info --context "kind-${CLUSTER_NAME}"
kubectl get nodes

helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update
kubectl create namespace gpu-operator --dry-run=client -o yaml | kubectl apply -f -
helm upgrade --install gpu-operator nvidia/gpu-operator -n gpu-operator

kubectl rollout status -n gpu-operator daemonset/nvidia-device-plugin-daemonset --timeout=5m

cat >"$TIME_SLICING_CONFIG" <<YAML
apiVersion: v1
kind: ConfigMap
metadata:
  name: time-slicing-config
  namespace: gpu-operator
data:
  any: |-
    version: v1
    flags:
      migStrategy: none
    sharing:
      timeSlicing:
        renameByDefault: false
        failRequestsGreaterThanOne: false
        resources:
          - name: nvidia.com/gpu
            replicas: ${GPU_REPLICAS}
YAML

kubectl apply -f "$TIME_SLICING_CONFIG"
kubectl patch clusterpolicies.nvidia.com/cluster-policy \
  -n gpu-operator --type merge \
  -p "{\"spec\":{\"devicePlugin\":{\"config\":{\"name\":\"time-slicing-config\",\"default\":\"any\"}}}}"

kubectl rollout restart -n gpu-operator daemonset/nvidia-device-plugin-daemonset
kubectl rollout status -n gpu-operator daemonset/nvidia-device-plugin-daemonset --timeout=5m

kubectl label node "${CLUSTER_NAME}-worker" kratos.io/gpu-model=t4 kratos.io/gpu-memory=16Gi --overwrite
kubectl label node "${CLUSTER_NAME}-worker2" kratos.io/gpu-model=a100 kratos.io/gpu-memory=40Gi --overwrite
kubectl label node "${CLUSTER_NAME}-worker3" kratos.io/gpu-model=l4 kratos.io/gpu-memory=24Gi --overwrite

kubectl get nodes --show-labels
kubectl describe node "${CLUSTER_NAME}-worker" | grep -E "nvidia.com/gpu|replicas|SHARED" || true
kubectl get pods -n gpu-operator

set +x
printf 'kind GPU lab setup completed. Command trace: %s\n' "$LOG_FILE"
