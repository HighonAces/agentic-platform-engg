#!/usr/bin/env bash
# scripts/04-verify.sh
# Verifies that Crossplane and ArgoCD are healthy and properly configured
set -euo pipefail

source "$(dirname "$0")/_env.sh"

PASS=0
FAIL=0

check() {
  local label="$1"
  shift
  if "$@" &>/dev/null; then
    echo "  ✅ ${label}"
    ((PASS++)) || true
  else
    echo "  ❌ ${label}"
    ((FAIL++)) || true
  fi
}

echo ""
echo "════════════════════════════════════════"
echo "  Cluster Verification"
echo "════════════════════════════════════════"

echo ""
echo "📍 Nodes:"
kubectl get nodes --no-headers | awk '{print "  " $1 "\t" $2}'

echo ""
echo "📦 Crossplane:"
check "crossplane-system namespace exists" kubectl get namespace crossplane-system
check "Crossplane pods running" kubectl get pods -n crossplane-system --field-selector=status.phase=Running --no-headers
check "GCP provider installed" kubectl get provider provider-gcp
check "GCP provider healthy" kubectl get provider provider-gcp -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}' | grep -q True
check "ProviderConfig exists" kubectl get providerconfig default
check "XRDs applied" kubectl get xrd xgkeclusters.platform.example.io

echo ""
echo "🔄 ArgoCD:"
check "argocd namespace exists" kubectl get namespace argocd
check "ArgoCD pods running" kubectl get pods -n argocd --field-selector=status.phase=Running --no-headers
check "app-of-apps Application exists" kubectl get application app-of-apps -n argocd
check "GitHub repo registered" kubectl get secret platform-gitops-repo -n argocd

echo ""
echo "════════════════════════════════════════"
echo "  Results: ${PASS} passed, ${FAIL} failed"
echo "════════════════════════════════════════"

if [[ ${FAIL} -gt 0 ]]; then
  exit 1
fi
