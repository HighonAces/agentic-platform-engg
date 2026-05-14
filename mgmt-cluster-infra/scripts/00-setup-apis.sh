#!/usr/bin/env bash
# scripts/00-setup-apis.sh
# Enable all required GCP APIs for the agentic platform
set -euo pipefail

source "$(dirname "$0")/_env.sh"

echo "🔧 Enabling GCP APIs for project: ${PROJECT_ID}"

gcloud services enable \
  container.googleapis.com \
  compute.googleapis.com \
  iam.googleapis.com \
  cloudresourcemanager.googleapis.com \
  sqladmin.googleapis.com \
  storage.googleapis.com \
  artifactregistry.googleapis.com \
  servicenetworking.googleapis.com \
  dns.googleapis.com \
  secretmanager.googleapis.com \
  --project="${PROJECT_ID}"

echo "✅ APIs enabled"
