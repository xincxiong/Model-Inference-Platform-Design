#!/usr/bin/env bash
# ==============================================================================
# install.sh - Install HAMi + NVIDIA GPU Operator on Kubernetes
# ==============================================================================
# Prerequisites:
#   - kubectl configured with cluster admin privileges
#   - Helm 3.x installed
#   - Kubernetes 1.25+ cluster with NVIDIA GPU nodes
#
# Usage:
#   ./install.sh [--dry-run] [--skip-gpu-operator] [--namespace hami-system]
# ==============================================================================

set -euo pipefail

# ── Configuration ─────────────────────────────────────────────────────────────
HAMI_VERSION="${HAMI_VERSION:-v2.8.1}"
GPU_OPERATOR_VERSION="${GPU_OPERATOR_VERSION:-v24.9.2}"
HAMI_NAMESPACE="${HAMI_NAMESPACE:-hami-system}"
GPU_OPERATOR_NAMESPACE="${GPU_OPERATOR_NAMESPACE:-gpu-operator}"
INFERENCE_NAMESPACE="inference-platform"
DRY_RUN=false
SKIP_GPU_OPERATOR=false

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --dry-run) DRY_RUN=true; shift ;;
    --skip-gpu-operator) SKIP_GPU_OPERATOR=true; shift ;;
    --namespace) HAMI_NAMESPACE="$2"; shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

HELM_FLAGS=""
if $DRY_RUN; then
  HELM_FLAGS="--dry-run"
  echo "[DRY RUN] No changes will be applied"
fi

echo "================================================================"
echo "  Model Inference Platform - HAMi + GPU Operator Installer"
echo "  HAMi version   : ${HAMI_VERSION}"
echo "  GPU Operator   : ${GPU_OPERATOR_VERSION}"
echo "  HAMi namespace : ${HAMI_NAMESPACE}"
echo "================================================================"

# ── Step 1: Create namespaces ─────────────────────────────────────────────────
echo ""
echo "[1/7] Creating namespaces..."
kubectl apply -f ../kubernetes/namespace.yaml $HELM_FLAGS || true

# ── Step 2: Install cert-manager (required by HAMi webhook) ──────────────────
echo ""
echo "[2/7] Installing cert-manager (HAMi webhook dependency)..."
if ! kubectl get namespace cert-manager &>/dev/null; then
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml
  echo "  Waiting for cert-manager to be ready..."
  kubectl wait --for=condition=available --timeout=120s \
    deployment/cert-manager-webhook -n cert-manager
fi

# ── Step 3: Install NVIDIA GPU Operator ──────────────────────────────────────
if ! $SKIP_GPU_OPERATOR; then
  echo ""
  echo "[3/7] Installing NVIDIA GPU Operator..."
  helm repo add nvidia https://helm.ngc.nvidia.com/nvidia || true
  helm repo update

  kubectl create namespace "${GPU_OPERATOR_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

  helm upgrade --install gpu-operator nvidia/gpu-operator \
    --namespace "${GPU_OPERATOR_NAMESPACE}" \
    --version "${GPU_OPERATOR_VERSION}" \
    --values gpu-operator-values.yaml \
    --wait \
    --timeout 300s \
    ${HELM_FLAGS}

  echo "  GPU Operator installed. Waiting for node-feature-discovery..."
  kubectl wait --for=condition=ready pod \
    -l app=gpu-feature-discovery \
    -n "${GPU_OPERATOR_NAMESPACE}" \
    --timeout=120s || true
else
  echo "[3/7] Skipping GPU Operator installation (--skip-gpu-operator)"
fi

# ── Step 4: Add HAMi Helm repo ───────────────────────────────────────────────
echo ""
echo "[4/7] Adding HAMi Helm repository..."
helm repo add hami-charts https://project-hami.github.io/HAMi/ || true
helm repo update

# ── Step 5: Install HAMi ─────────────────────────────────────────────────────
echo ""
echo "[5/7] Installing HAMi ${HAMI_VERSION}..."
kubectl create namespace "${HAMI_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install hami hami-charts/hami \
  --namespace "${HAMI_NAMESPACE}" \
  --version "${HAMI_VERSION}" \
  --values values.yaml \
  --set scheduler.extenderPort=9090 \
  --set devicePlugin.nvidia.enabled=true \
  --wait \
  --timeout 300s \
  ${HELM_FLAGS}

# ── Step 6: Verify HAMi scheduler is running ─────────────────────────────────
echo ""
echo "[6/7] Verifying HAMi installation..."
if ! $DRY_RUN; then
  echo "  Waiting for HAMi scheduler pod..."
  kubectl wait --for=condition=ready pod \
    -l app.kubernetes.io/name=hami-scheduler \
    -n "${HAMI_NAMESPACE}" \
    --timeout=120s

  echo "  HAMi Scheduler status:"
  kubectl get pods -n "${HAMI_NAMESPACE}" -l app.kubernetes.io/name=hami-scheduler

  echo ""
  echo "  HAMi Device Plugin status:"
  kubectl get daemonset -n "${HAMI_NAMESPACE}"

  echo ""
  echo "  GPU nodes discovered by HAMi:"
  kubectl get nodes -o custom-columns=\
"NAME:.metadata.name,\
GPU:.status.capacity.nvidia\.com/gpu,\
GPU-MEM:.status.capacity.nvidia\.com/gpumem,\
GPU-CORES:.status.capacity.nvidia\.com/gpucores" 2>/dev/null || \
  kubectl get nodes --show-labels | grep -E "gpu|nvidia" || \
  echo "  (No GPU nodes found or labels not yet applied)"
fi

# ── Step 7: Apply platform Kubernetes manifests ───────────────────────────────
echo ""
echo "[7/7] Applying inference-platform Kubernetes manifests..."
if ! $DRY_RUN; then
  kubectl apply -f ../kubernetes/namespace.yaml
  kubectl apply -f ../kubernetes/secrets.yaml
  kubectl apply -f ../kubernetes/configmap.yaml
  kubectl apply -f ../kubernetes/postgres-statefulset.yaml
  kubectl apply -f ../kubernetes/redis-statefulset.yaml
  kubectl apply -f ../kubernetes/backend-deployment.yaml
  kubectl apply -f ../kubernetes/frontend-deployment.yaml
  kubectl apply -f ../kubernetes/ingress.yaml

  echo "  Waiting for PostgreSQL..."
  kubectl wait --for=condition=ready pod \
    -l app=postgres \
    -n "${INFERENCE_NAMESPACE}" \
    --timeout=120s

  echo "  Waiting for Redis..."
  kubectl wait --for=condition=ready pod \
    -l app=redis \
    -n "${INFERENCE_NAMESPACE}" \
    --timeout=60s

  echo "  Waiting for backend..."
  kubectl wait --for=condition=available deployment/backend \
    -n "${INFERENCE_NAMESPACE}" \
    --timeout=120s
fi

echo ""
echo "================================================================"
echo "  Installation complete!"
echo ""
echo "  Access the platform:"
echo "  - Frontend : http://inference.example.com"
echo "  - API      : http://api.inference.example.com"
echo "  - HAMi     : kubectl port-forward -n ${HAMI_NAMESPACE} svc/hami-scheduler 9090:9090"
echo ""
echo "  Verify GPU resources:"
echo "  kubectl get nodes -o json | jq '.items[].status.capacity | to_entries | map(select(.key | contains(\"nvidia\")))'"
echo ""
echo "  Deploy a vLLM worker with HAMi GPU slicing:"
echo "  kubectl apply -f ../kubernetes/vllm-worker.yaml"
echo "================================================================"
