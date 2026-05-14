// main.go — Crossplane MCP Server entry point.
//
// Exposes the following tools to AI agents:
//   list_xrds                 – available infrastructure types (XRDs)
//   list_compositions         – compositions + which XRD each implements
//   list_providers            – provider health
//   list_claims               – namespace-scoped claims
//   list_composite_resources  – cluster-scoped XRs + their composed resources
//   list_managed_resources    – raw cloud objects (SYNCED/READY/error)
//   get_resource_status       – deep status for any k8s object
//   get_resource_events       – recent k8s Events for any object
//
// Transport (set MCP_TRANSPORT env var):
//   stdio  (default) – for Claude Desktop / ADK stdio clients
//   http             – StreamableHTTP (MCP 2025-03-26 spec) on MCP_ADDR (default :8080)
//                      Plain POST to /mcp — no session handshake, curl-friendly.
package main

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Build Kubernetes clients (in-cluster → kubeconfig fallback)
	c, err := newClient()
	if err != nil {
		log.Fatalf("failed to build kubernetes client: %v", err)
	}

	s := server.NewMCPServer(
		"crossplane-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	registerXRDTools(s, c)
	registerProviderTools(s, c)
	registerResourceTools(s, c)

	switch os.Getenv("MCP_TRANSPORT") {
	case "sse":
		addr := os.Getenv("MCP_ADDR")
		if addr == "" {
			addr = ":8080"
		}
		log.Printf("crossplane-mcp: SSE transport on http://localhost%s/sse", addr)
		// SSEServer sets up /sse (GET) and /message (POST) automatically
		sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost"+addr))
		if err := sseServer.Start(addr); err != nil {
			log.Fatal(err)
		}
	default:
		log.Print("crossplane-mcp: stdio transport")
		if err := server.ServeStdio(s); err != nil {
			log.Fatal(err)
		}
	}
}
