#!/usr/bin/env bash
# scripts/03-install-argocd.sh
# Installs ArgoCD and bootstraps the App-of-Apps root Application
set -euo pipefail

source "$(dirname "$0")/_env.sh"

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

if [[ -z "${GITHUB_TOKEN}" ]]; then
  echo "❌ GITHUB_TOKEN environment variable is not set. Export it before running this script."
  exit 1
fi

echo "📦 Adding Argo Helm repo..."
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

echo "🔧 Creating ArgoCD namespace..."
kubectl create namespace "${ARGOCD_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

echo "🚀 Installing ArgoCD ${ARGOCD_CHART_VERSION}..."
helm upgrade --install argocd argo/argo-cd \
  --namespace "${ARGOCD_NAMESPACE}" \
  --version "${ARGOCD_CHART_VERSION}" \
  --values "${SCRIPT_DIR}/argocd/values.yaml" \
  --wait --timeout=10m

echo "🔑 Applying GitHub repository secret..."
# Replace placeholders with real values
sed \
  -e "s|https://github.com/<YOUR_ORG>/<YOUR_REPO>|${GITHUB_REPO_URL}|g" \
  -e "s|<YOUR_GITHUB_PAT>|${GITHUB_TOKEN}|g" \
  "${SCRIPT_DIR}/argocd/repositories.yaml" | kubectl apply -f -



echo ""
echo "✅ ArgoCD installation complete"
echo ""
echo "Admin password:"
kubectl get secret argocd-initial-admin-secret \
  -n "${ARGOCD_NAMESPACE}" \
  -o jsonpath="{.data.password}" | base64 --decode
echo ""
echo ""
echo "To access the UI, run:"
echo "  kubectl port-forward svc/argocd-server -n ${ARGOCD_NAMESPACE} 8080:443"
echo "  Then open: https://localhost:8080"
