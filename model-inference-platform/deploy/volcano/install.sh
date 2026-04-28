#!/bin/bash
# Volcano Installation Script for Model Inference Platform
# Installs Volcano scheduler + integrates with HAMi GPU virtualization

set -e

echo "========================================="
echo " Volcano Installation for Inference Platform"
echo "========================================="

# ─── Configuration ─────────────────────────────────────────────────────────────
VOLCANO_VERSION="${VOLCANO_VERSION:-v1.9.0}"
NAMESPACE="${NAMESPACE:-volcano-system}"
RELEASE_NAME="${RELEASE_NAME:-volcano}"
HELM_REPO="https://volcano-sh.github.io/helm-charts"

# ─── Step 1: Check Prerequisites ──────────────────────────────────────────────
echo ""
echo "Step 1: Checking prerequisites..."

if ! command -v helm &> /dev/null; then
    echo "❌ helm is not installed. Please install helm first."
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed. Please install kubectl first."
    exit 1
fi

echo "✅ Prerequisites met (helm, kubectl)"

# ─── Step 2: Add Volcano Helm Repository ──────────────────────────────────────
echo ""
echo "Step 2: Adding Volcano Helm repository..."

helm repo add volcano-sh "$HELM_REPO" || true
helm repo update

echo "✅ Volcano Helm repository added"

# ─── Step 3: Install Volcano ──────────────────────────────────────────────────
echo ""
echo "Step 3: Installing Volcano $VOLCANO_VERSION..."

helm upgrade --install "$RELEASE_NAME" volcano-sh/volcano \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --version "$VOLCANO_VERSION" \
  --values deploy/volcano/values.yaml \
  --wait \
  --timeout 300s

echo "✅ Volcano installed successfully"

# ─── Step 4: Verify Installation ──────────────────────────────────────────────
echo ""
echo "Step 4: Verifying Volcano installation..."

echo "Checking Volcano pods..."
kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=volcano

echo ""
echo "Checking Volcano CRDs..."
kubectl get crd | grep volcano

echo ""
echo "Checking Volcano scheduler..."
kubectl get svc -n "$NAMESPACE" | grep volcano-scheduler

echo "✅ Volcano verification complete"

# ─── Step 5: Create Default Queue ─────────────────────────────────────────────
echo ""
echo "Step 5: Creating default queue for inference platform..."

cat <<EOF | kubectl apply -f -
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: inference-platform-queue
spec:
  weight: 1
  reclaimable: true
  capability:
    nvidia.com/gpu: 32
  guarantee:
    resource:
      nvidia.com/gpu: 8
EOF

echo "✅ Default queue created"

# ─── Step 6: Configure HAMi Integration ───────────────────────────────────────
echo ""
echo "Step 6: Configuring HAMi integration..."

# Check if HAMi is installed
if kubectl get namespace hami-system &> /dev/null; then
    echo "✅ HAMi is already installed"
    echo "   Volcano will work with HAMi for GPU resource allocation:"
    echo "   - HAMi: GPU memory isolation, core quota"
    echo "   - Volcano: Gang scheduling, queue management, preemption"
else
    echo "⚠️  HAMi is not installed. Volcano will still work, but without GPU virtualization."
    echo "   To install HAMi, run: bash deploy/hami/install.sh"
fi

# ─── Step 7: Display Summary ──────────────────────────────────────────────────
echo ""
echo "========================================="
echo " Installation Complete!"
echo "========================================="
echo ""
echo "Volcano Components:"
echo "  - Scheduler:    $NAMESPACE/volcano-scheduler"
echo "  - Controller:   $NAMESPACE/volcano-controller-manager"
echo "  - Admission:    $NAMESPACE/volcano-admission"
echo ""
echo "Default Queue:"
echo "  - Name:         inference-platform-queue"
echo "  - GPU Quota:    32 GPUs"
echo "  - Guaranteed:   8 GPUs"
echo ""
echo "Next Steps:"
echo "  1. Update fine-tuning deployments to use VolcanoJob"
echo "  2. Configure queue assignment in Job annotations"
echo "  3. Monitor queue usage: kubectl get queue inference-platform-queue"
echo ""
