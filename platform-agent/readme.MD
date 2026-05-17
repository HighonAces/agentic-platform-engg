# Platform Agent

A **Google ADK**-based AI platform engineer that autonomously manages GCP
infrastructure by orchestrating three MCP servers: **ArgoCD**, **Crossplane**,
and **GitHub**.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Platform Agent                      │
│            (Google ADK · Gemini LLM)                 │
│                                                      │
│  ┌────────────┐  ┌────────────────┐  ┌───────────┐  │
│  │ ArgoCD MCP │  │ Crossplane MCP │  │ GitHub MCP│  │
│  │  (SSE)     │  │   (SSE)        │  │  (HTTP)   │  │
│  └────────────┘  └────────────────┘  └───────────┘  │
└─────────────────────────────────────────────────────┘
         │                  │                 │
   ArgoCD API         Kubernetes API    GitHub API
   (GitOps apps)   (Crossplane CRDs)  (GitOps repo)
```

---

## What the agent can do

| Capability | Tools used |
|---|---|
| **Provision GCP infra** | GitHub push → ArgoCD sync → Crossplane status |
| **Troubleshoot sync failures** | ArgoCD events, resource tree, workload logs |
| **Inspect Crossplane resources** | list_claims, list_managed_resources, get_resource_status |
| **Manage GitOps repo** | push_files, create_pull_request, get_file_contents |
| **Monitor provider health** | list_providers |

---

## Prerequisites

| Tool | Purpose |
|---|---|
| `kubectl port-forward svc/argocd-server -n argocd 8081:443` | Expose ArgoCD MCP |
| `MCP_TRANSPORT=sse go run . (in crossplane-mcp)` | Expose Crossplane MCP |
| `GOOGLE_API_KEY` env var | Gemini API key |
| `GITHUB_PAT` env var | GitHub Personal Access Token |

---

## Quick start

```bash
# 1. Start the MCP servers (in separate terminals)
cd argocd-mcp   && MCP_TRANSPORT=sse MCP_ADDR=:8081 go run .
cd crossplane-mcp && MCP_TRANSPORT=sse MCP_ADDR=:8080 go run .

# 2. Set environment variables
export GOOGLE_API_KEY="your-gemini-api-key"
export GITHUB_PAT="ghp_your-github-pat"

# 3. Launch the agent web UI
cd platform-agent && go run . web
# Open http://localhost:8080 in your browser

# 4. Or run a one-shot CLI command
go run . run "List all ArgoCD applications and their health status"
go run . run "Add a new GCS bucket for team-1 with versioning enabled"
go run . run "Why is team-2-infrastructure out of sync?"
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `GOOGLE_API_KEY` | — | **Required.** Gemini API key |
| `GITHUB_PAT` | — | GitHub PAT. If unset, GitHub MCP is disabled |
| `GEMINI_MODEL` | `gemini-2.0-flash` | Gemini model name |
| `MCP_ARGOCD_URL` | `http://localhost:8081/sse` | ArgoCD MCP SSE endpoint |
| `MCP_CROSSPLANE_URL` | `http://localhost:8080/sse` | Crossplane MCP SSE endpoint |
| `MCP_GITHUB_URL` | `https://api.githubcopilot.com/mcp/` | GitHub MCP endpoint |

---

## Example conversations

### Provision a new GCS bucket
```
You: "Create a new STANDARD GCS bucket called team-1-logs in the US region with 
      versioning enabled. Push it to the repo and monitor until it's ready."

Agent: 
1. Checks list_xrds to confirm BucketClaim schema
2. Drafts infrastructure/team-1/team-1-logs-bucket.yaml
3. Pushes via GitHub push_files
4. Syncs team-1-infrastructure ArgoCD app
5. Polls list_claims until Ready=True
```

### Troubleshoot a failing application
```
You: "Why is team-2-infrastructure showing as degraded?"

Agent:
1. get_application → shows sync error
2. get_application_resource_tree → identifies failing BucketClaim
3. get_resource_status → finds "quota exceeded" error
4. get_resource_events → confirms GCP quota error message
5. Suggests resolution steps
```
