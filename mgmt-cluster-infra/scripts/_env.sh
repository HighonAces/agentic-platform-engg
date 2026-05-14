#!/usr/bin/env bash
# scripts/_env.sh
# Shared environment variables — source this from every script.
# Usage: source ./scripts/_env.sh   (NOT ./_env.sh — that runs a subshell)
#
# Values here are ALWAYS applied (direct assignment, not conditional defaults).
# Override any variable by exporting it AFTER sourcing this file.

export PROJECT_ID="gen-lang-client-0714624790"
export REGION="us-central1"
export ZONE="us-central1-a"          # used for ZONAL cluster (see 01-create-cluster.sh)
export CLUSTER_NAME="mgmt-cluster"
export MACHINE_TYPE="e2-standard-4"
export NUM_NODES="3"                  # 3 nodes × 4 vCPU = 12 vCPU (fits in 32 quota)
export GSA_NAME="crossplane-sa"
export GITHUB_REPO_URL="https://github.com/HighonAces/agentic-platform-engg"
export ARGOCD_NAMESPACE="argocd"
export CROSSPLANE_VERSION="1.20.0"
export ARGOCD_CHART_VERSION="7.7.11"
