#!/usr/bin/env bash
# scripts/00-delete-cluster.sh
# Deletes the GKE management cluster and cleans up associated resources.
# WARNING: This is destructive. All cluster workloads will be lost.
set -euo pipefail

source "$(dirname "$0")/_env.sh"

echo "⚠️  About to delete cluster: ${CLUSTER_NAME} in zone ${ZONE}"
echo "   Project: ${PROJECT_ID}"
echo "   Press Ctrl+C within 5 seconds to cancel..."
sleep 5

echo "🗑️  Deleting GKE cluster..."
gcloud container clusters delete "${CLUSTER_NAME}" \
  --zone="${ZONE}" \
  --project="${PROJECT_ID}" \
  --quiet

echo "🗑️  Deleting Google Service Account..."
gcloud iam service-accounts delete \
  "${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --project="${PROJECT_ID}" \
  --quiet || true

echo "✅ Cluster deleted. Run 01-create-cluster.sh to recreate."
