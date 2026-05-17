// main.go — Platform Agent
//
// A Google ADK-based AI agent that connects to three MCP servers:
//   - argocd-mcp  (SSE at MCP_ARGOCD_URL, default http://localhost:8081/sse)
//   - crossplane-mcp (SSE at MCP_CROSSPLANE_URL, default http://localhost:8080/sse)
//   - github-mcp-server (remote Streamable-HTTP at MCP_GITHUB_URL)
//
// The agent uses Gemini as its LLM and exposes a web UI + REST API through
// the standard ADK launcher.
//
// Usage:
//
//	export GOOGLE_API_KEY=...
//	export GITHUB_PAT=...        # GitHub Personal Access Token (for github-mcp)
//	export MCP_ARGOCD_URL=http://localhost:8081/sse
//	export MCP_CROSSPLANE_URL=http://localhost:8080/sse
//	go run . web                  # Launch ADK web UI
//	go run . run "list all ArgoCD applications"
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/mcptoolset"
	"google.golang.org/genai"
)

// ───────────────────────────────────────────────────────────────────────────────
// helpers
// ───────────────────────────────────────────────────────────────────────────────

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// sseTransport builds an SSE MCP transport for a locally-running SSE server.
func sseTransport(sseURL string) mcpsdk.Transport {
	return &mcpsdk.SSEClientTransport{Endpoint: sseURL}
}

// streamableTransport builds a Streamable-HTTP MCP transport (for remote MCPs
// like the official github-mcp-server).
func streamableTransport(ctx context.Context, endpointURL, pat string) mcpsdk.Transport {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: pat})
	return &mcpsdk.StreamableClientTransport{
		Endpoint:   endpointURL,
		HTTPClient: oauth2.NewClient(ctx, ts),
	}
}

// newMCPToolset creates an MCPToolset from the provided transport.
// The agent runner will connect/disconnect automatically.
func newMCPToolset(transport mcpsdk.Transport) tool.Toolset {
	ts, err := mcptoolset.New(mcptoolset.Config{
		Transport: transport,
	})
	if err != nil {
		log.Fatalf("failed to create MCP toolset: %v", err)
	}
	return ts
}

// ───────────────────────────────────────────────────────────────────────────────
// agent system prompt
// ───────────────────────────────────────────────────────────────────────────────

const systemInstruction = `You are the **Platform Agent**, an expert Site Reliability Engineer
and Platform Engineer responsible for the Agentic Platform Engineering stack.

You have access to three sets of tools that cover the full lifecycle of GCP
infrastructure managed via GitOps:

## 1. ArgoCD tools  (prefix: argocd_mcp)
Use these to inspect, sync, and manage ArgoCD applications and their resources.
- list_applications, get_application, get_application_resource_tree
- get_application_managed_resources, get_application_events
- get_application_workload_logs, sync_application, create_application
- update_application, delete_application, get_resources, get_resource_events
- run_resource_action, list_clusters

## 2. Crossplane tools  (prefix: crossplane_mcp)
Use these to understand and troubleshoot Crossplane-managed GCP resources.
- list_xrds, list_compositions, list_providers
- list_claims, list_composite_resources, list_managed_resources
- get_resource_status, get_resource_events

## 3. GitHub tools  (prefix: github_mcp)
Use these to push new infrastructure manifests to the GitOps repository, read
files, manage issues, and create pull requests.
- get_file_contents, create_or_update_file, push_files
- create_pull_request, list_pull_requests, get_commit, list_commits
- list_issues, create_issue, add_issue_comment

## Workflow guidelines

### Provisioning new infrastructure
1. Identify the correct Crossplane XRD and Composition with list_xrds /
   list_compositions.
2. Draft the BucketClaim (or other Claim) YAML that matches the XRD schema.
3. Push the file to the GitOps repo under infrastructure/<team>/ using the
   GitHub push_files tool.
4. Wait and then sync the matching ArgoCD application with sync_application.
5. Poll get_resource_status and list_claims to watch progress.

### Troubleshooting
1. Start with list_applications / get_application to see health and sync status.
2. Get deeper detail from get_application_resource_tree and
   get_application_events.
3. Check the Crossplane layer with list_claims, list_composite_resources, and
   list_managed_resources.
4. Use get_resource_events and get_resource_status for root-cause analysis.
5. If a provider is unhealthy, list_providers will surface the error.

### General rules
- Always confirm destructive actions (delete, prune) with the user before executing.
- When creating GitHub files, use meaningful commit messages following conventional commits.
- Never expose secrets; remind the user to store tokens in environment variables.
- Format responses clearly with headings, bullet points, and code blocks.`

// ───────────────────────────────────────────────────────────────────────────────
// main
// ───────────────────────────────────────────────────────────────────────────────

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// ── LLM model ─────────────────────────────────────────────────────────────────
	modelName := envOrDefault("GEMINI_MODEL", "gemini-2.0-flash")
	model, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("failed to create Gemini model: %v", err)
	}

	// ── MCP toolsets ──────────────────────────────────────────────────────────────
	argoCDURL := envOrDefault("MCP_ARGOCD_URL", "http://localhost:8081/sse")
	crossplaneURL := envOrDefault("MCP_CROSSPLANE_URL", "http://localhost:8080/sse")

	argoCDToolset := newMCPToolset(sseTransport(argoCDURL))
	crossplaneToolset := newMCPToolset(sseTransport(crossplaneURL))

	toolsets := []tool.Toolset{crossplaneToolset, argoCDToolset}

	// GitHub MCP is optional — only added when GITHUB_PAT is set.
	if githubPAT := os.Getenv("GITHUB_PAT"); githubPAT != "" {
		githubURL := envOrDefault("MCP_GITHUB_URL", "https://api.githubcopilot.com/mcp/")
		githubToolset := newMCPToolset(streamableTransport(ctx, githubURL, githubPAT))
		toolsets = append(toolsets, githubToolset)
		log.Println("platform-agent: GitHub MCP toolset enabled")
	} else {
		log.Println("platform-agent: GITHUB_PAT not set — GitHub MCP toolset disabled")
	}

	// ── LLM Agent ──────────────────────────────────────────────────────────────
	a, err := llmagent.New(llmagent.Config{
		Name:        "platform-agent",
		Model:       model,
		Description: "Agentic Platform Engineer — manages GCP infrastructure via ArgoCD, Crossplane, and GitHub.",
		Instruction: systemInstruction,
		Toolsets:    toolsets,
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	log.Printf("platform-agent: starting with model=%s argocd=%s crossplane=%s",
		modelName, argoCDURL, crossplaneURL)

	// ── ADK Launcher (web UI + CLI) ─────────────────────────────────────────────
	cfg := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(a),
	}
	l := full.NewLauncher()
	if err = l.Execute(ctx, cfg, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
