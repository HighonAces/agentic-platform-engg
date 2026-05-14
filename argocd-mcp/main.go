package main

import (
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	baseURL := os.Getenv("ARGOCD_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	token := os.Getenv("ARGOCD_API_TOKEN")

	c := NewArgoCDClient(baseURL, token)

	s := server.NewMCPServer(
		"argocd-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	registerTools(s, c)

	switch os.Getenv("MCP_TRANSPORT") {
	case "sse":
		addr := os.Getenv("MCP_ADDR")
		if addr == "" {
			addr = ":8081" // Use a different default than Crossplane if running both
		}
		log.Printf("argocd-mcp: SSE transport on http://localhost%s/sse", addr)
		sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost"+addr))
		if err := sseServer.Start(addr); err != nil {
			log.Fatal(err)
		}
	default:
		log.Print("argocd-mcp: stdio transport")
		if err := server.ServeStdio(s); err != nil {
			log.Fatal(err)
		}
	}
}
