#!/usr/bin/env bash
# scripts/01-create-cluster.sh
# Creates the GKE management cluster with Workload Identity enabled
set -euo pipefail

source "$(dirname "$0")/_env.sh"

echo "🚀 Creating GKE cluster: ${CLUSTER_NAME} in zone ${ZONE}"
echo "   vCPU usage: ${NUM_NODES} nodes × 4 vCPU = $((NUM_NODES * 4)) vCPU (regional quota cap: 32)"

gcloud container clusters create "${CLUSTER_NAME}" \
  --project="${PROJECT_ID}" \
  --zone="${ZONE}" \
  --machine-type="${MACHINE_TYPE}" \
  --num-nodes="${NUM_NODES}" \
  --workload-pool="${PROJECT_ID}.svc.id.goog" \
  --enable-ip-alias \
  --enable-autoscaling \
  --min-nodes="${NUM_NODES}" \
  --max-nodes=6 \
  --enable-autorepair \
  --enable-autoupgrade \
  --release-channel=regular \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver

echo "📋 Fetching kubeconfig..."
gcloud container clusters get-credentials "${CLUSTER_NAME}" \
  --zone="${ZONE}" \
  --project="${PROJECT_ID}"

echo "👤 Creating Google Service Account for Crossplane..."
GSA_EMAIL="${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

gcloud iam service-accounts create "${GSA_NAME}" \
  --project="${PROJECT_ID}" \
  --display-name="Crossplane GCP Provider SA" 2>/dev/null || true

# Wait for IAM propagation — GCP creates the SA asynchronously;
# policy binding fails if it fires before the SA is globally visible.
echo "⏳ Waiting for GSA to propagate across GCP IAM..."
for i in $(seq 1 20); do
  if gcloud iam service-accounts describe "${GSA_EMAIL}" \
       --project="${PROJECT_ID}" &>/dev/null; then
    echo "   ✅ GSA is visible (attempt ${i})"
    break
  fi
  echo "   Attempt ${i}/20 — not yet visible, retrying in 5s..."
  sleep 5
done

echo "🔐 Granting roles/editor to GSA..."
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${GSA_EMAIL}" \
  --role="roles/editor"

# NOTE: Workload Identity bindings for Crossplane provider KSAs are done in
# 02-install-crossplane.sh — provider KSAs don't exist until after Helm install.

echo "✅ Cluster ready:"
kubectl get nodes -o wide
