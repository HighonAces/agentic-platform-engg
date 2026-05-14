# mgmt-cluster-infra

Bootstrap infrastructure for the **Agentic Platform Engineering** management cluster.

This folder contains everything needed to:
1. Create a GKE cluster on GCP
2. Install **Crossplane** (infrastructure-as-code via Kubernetes)
3. Install **ArgoCD** (GitOps continuous delivery)
4. Bootstrap the GitOps **App-of-Apps** pattern

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| `gcloud` | ≥ 450.0 | [Install](https://cloud.google.com/sdk/docs/install) |
| `kubectl` | ≥ 1.28 | `gcloud components install kubectl` |
| `helm` | ≥ 3.14 | [Install](https://helm.sh/docs/intro/install/) |
| `argocd` CLI | ≥ 2.10 | `brew install argocd` |

## Folder Structure

```
mgmt-cluster-infra/
├── _env.sh (inherited via scripts/_env.sh)
├── gcloud-commands.txt          # Step-by-step gcloud CLI reference
├── crossplane-installation-commands.txt  # Step-by-step Crossplane reference
├── argo-installation-commands.txt        # Step-by-step ArgoCD reference
│
├── crossplane/
│   ├── values.yaml              # Helm values for Crossplane
│   ├── provider-gcp.yaml        # GCP Provider manifest
│   ├── provider-config.yaml     # ProviderConfig (Workload Identity)
│   ├── xrd/
│   │   └── gke-cluster-xrd.yaml # CompositeResourceDefinition for GKE clusters
│   └── compositions/
│       └── gke-cluster-composition.yaml  # Composition implementing the XRD
│
├── argocd/
│   ├── values.yaml              # Helm values for ArgoCD
│   ├── repositories.yaml        # GitHub repo secret
│   └── app-of-apps.yaml         # Root Application (App-of-Apps)
│
└── scripts/                     # Automated bootstrap scripts
    ├── _env.sh                  # Shared variables — EDIT THIS FIRST
    ├── 00-setup-apis.sh
    ├── 01-create-cluster.sh
    ├── 02-install-crossplane.sh
    ├── 03-install-argocd.sh
    └── 04-verify.sh
```

## Quick Start

### 1. Configure variables

Edit `scripts/_env.sh` with your GCP project ID, region, and GitHub repo URL.

### 2. Run scripts in order

```bash
cd mgmt-cluster-infra

# Make scripts executable
chmod +x scripts/*.sh

# Step 0: Enable GCP APIs
./scripts/00-setup-apis.sh

# Step 1: Create GKE cluster + Workload Identity GSA
./scripts/01-create-cluster.sh

# Step 2: Install Crossplane + GCP Provider
./scripts/02-install-crossplane.sh

# Step 3: Install ArgoCD + bootstrap App-of-Apps
#         Requires GITHUB_TOKEN env var
export GITHUB_TOKEN="ghp_xxxx"
./scripts/03-install-argocd.sh

# Step 4: Verify everything is healthy
./scripts/04-verify.sh
```

### 3. Update placeholder values

Before running, replace all `<YOUR_*>` placeholders in:
- `scripts/_env.sh`
- `crossplane/provider-gcp.yaml` (GSA email)
- `crossplane/provider-config.yaml` (project ID)
- `argocd/repositories.yaml` (GitHub repo URL + PAT)
- `argocd/app-of-apps.yaml` (GitHub repo URL)

## Next Steps

After the management cluster is running:

1. **`gitops/`** — Add ArgoCD Application definitions and Crossplane claims
2. **`mcp-servers/`** — Build MCP servers for Crossplane and ArgoCD APIs
3. **`adk-agent/`** — Build the ADK agent that talks to MCP servers
