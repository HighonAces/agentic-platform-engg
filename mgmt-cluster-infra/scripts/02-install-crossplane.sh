#!/usr/bin/env bash
# scripts/02-install-crossplane.sh
# Installs Crossplane, the GCP provider, and core XRDs/Compositions
set -euo pipefail

source "$(dirname "$0")/_env.sh"

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "📦 Adding Crossplane Helm repo..."
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update

echo "🔧 Installing Crossplane ..."
kubectl create namespace crossplane-system --dry-run=client -o yaml | kubectl apply -f -

helm upgrade --install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system \
  --version "${CROSSPLANE_VERSION}" \
  --values "${SCRIPT_DIR}/crossplane/values.yaml" \
  --wait --timeout=5m

echo "📡 Installing GCP Storage Provider..."
sed "s/<YOUR_GCP_PROJECT_ID>/${PROJECT_ID}/g" \
  "${SCRIPT_DIR}/crossplane/provider-gcp.yaml" | kubectl apply -f -

echo "⏳ Waiting for providers to become Healthy..."
kubectl wait provider.pkg.crossplane.io/upbound-provider-family-gcp \
  --for=condition=Healthy \
  --timeout=300s
kubectl wait provider.pkg.crossplane.io/provider-gcp-storage \
  --for=condition=Healthy \
  --timeout=300s

echo "🔗 Binding Workload Identity: provider KSAs → GSA..."
# Provider KSAs are only created once provider pods start up (after `wait --for=condition=Healthy`).
# Upbound generates unique hashed names like upbound-provider-family-gcp-<hash>.
# We bind every KSA matching 'provider|upbound' so future providers are also covered.
GSA_EMAIL="${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
for KSA in $(kubectl get serviceaccounts -n crossplane-system --no-headers \
               -o custom-columns='NAME:.metadata.name' \
             | grep -E 'provider|upbound'); do
  echo "   → crossplane-system/${KSA}"
  gcloud iam service-accounts add-iam-policy-binding "${GSA_EMAIL}" \
    --project="${PROJECT_ID}" \
    --role="roles/iam.workloadIdentityUser" \
    --member="serviceAccount:${PROJECT_ID}.svc.id.goog[crossplane-system/${KSA}]" \
    --quiet
done

echo "🔗 Applying ProviderConfig (Workload Identity)..."
# Replace placeholder before applying
sed "s/<YOUR_GCP_PROJECT_ID>/${PROJECT_ID}/g" \
  "${SCRIPT_DIR}/crossplane/provider-config.yaml" | kubectl apply -f -

echo "⚙️  Installing Pipeline Functions (Go Templating & Auto Ready)..."
kubectl apply -f "${SCRIPT_DIR}/crossplane/functions/function-go-templating.yaml"
kubectl apply -f "${SCRIPT_DIR}/crossplane/functions/function-auto-ready.yaml"

echo "⏳ Waiting for functions to become Healthy..."
kubectl wait function.pkg.crossplane.io/function-go-templating \
  --for=condition=Healthy \
  --timeout=180s
kubectl wait function.pkg.crossplane.io/function-auto-ready \
  --for=condition=Healthy \
  --timeout=180s

echo "📋 Applying XRD and Composition..."
kubectl apply -f "${SCRIPT_DIR}/crossplane/xrd/"
kubectl apply -f "${SCRIPT_DIR}/crossplane/compositions/"

echo "✅ Crossplane installation complete"
kubectl get providers
kubectl get functions
kubectl get xrd
echo ""
echo "To create a test bucket, submit a claim:"
echo "  kubectl apply -f crossplane/examples/test-bucket-claim.yaml"
