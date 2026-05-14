// cmd/test-client/main.go
// Local smoke test for the MCP server using the official mcp-go client.
//
// Usage:
//   Terminal 1:  MCP_TRANSPORT=sse ./crossplane-mcp
//   Terminal 2:  go run test-local.go
//
// This is much more reliable than curl because it handles the SSE session
// lifecycle exactly as an AI agent would.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	fmt.Println("═══════════════════════════════════════")
	fmt.Println("  Crossplane MCP — local smoke test")
	fmt.Println("  Endpoint: http://localhost:8080/sse")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	// 1. Create SSE client
	c, err := mcpclient.NewSSEMCPClient("http://localhost:8080/sse")
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 2. Start the client (establishes SSE stream)
	if err := c.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start client (is the server running?): %v", err)
	}

	// 3. Initialize handshake
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = "2024-11-05"
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "smoke-test",
		Version: "1.0",
	}

	initRes, err := c.Initialize(ctx, initReq)
	if err != nil {
		log.Fatalf("❌ Failed to initialize: %v", err)
	}
	fmt.Printf("── Init ────────────────────────────────\n")
	fmt.Printf("  Server: %s %s\n\n", initRes.ServerInfo.Name, initRes.ServerInfo.Version)

	if err := c.Ping(ctx); err != nil {
		log.Fatalf("❌ Ping failed: %v", err)
	}

	// 4. List Tools
	fmt.Printf("── Tool discovery ──────────────────────\n")
	toolsReq := mcp.ListToolsRequest{}
	toolsRes, err := c.ListTools(ctx, toolsReq)
	if err != nil {
		log.Fatalf("❌ Failed to list tools: %v", err)
	}
	fmt.Printf("  Found %d tools:\n", len(toolsRes.Tools))
	for _, t := range toolsRes.Tools {
		fmt.Printf("    • %s\n", t.Name)
	}
	fmt.Println()

	// 5. Call Tools
	fmt.Printf("── Tool calls ──────────────────────────\n")
	pass := 0
	fail := 0

	calls := []struct {
		name string
		args map[string]interface{}
	}{
		{"list_xrds", nil},
		{"list_compositions", nil},
		{"list_providers", nil},
		{"list_claims", map[string]interface{}{"namespace": ""}},
		{"list_composite_resources", map[string]interface{}{"kind": ""}},
		{"list_managed_resources", map[string]interface{}{"provider_filter": "storage.gcp"}},
	}

	for _, call := range calls {
		req := mcp.CallToolRequest{}
		req.Params.Name = call.name
		if call.args != nil {
			req.Params.Arguments = call.args
		}

		res, err := c.CallTool(ctx, req)
		if err != nil {
			fmt.Printf("  ❌  %s: %v\n", call.name, err)
			fail++
			continue
		}

		if res.IsError {
			// Extract error message if possible
			errMsg := "unknown error"
			if len(res.Content) > 0 {
				if txt, ok := res.Content[0].(mcp.TextContent); ok {
					errMsg = txt.Text
				}
			}
			fmt.Printf("  ❌  %s: %s\n", call.name, errMsg)
			fail++
		} else {
			fmt.Printf("  ✅  %s\n", call.name)
			pass++
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════")
	if fail == 0 {
		fmt.Printf("  ✅  All %d tool calls succeeded\n", pass)
	} else {
		fmt.Printf("  Passed: %d  Failed: %d\n", pass, fail)
	}
	fmt.Println("═══════════════════════════════════════")
}
