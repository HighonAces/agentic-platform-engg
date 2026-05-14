// providers.go — Tool: list_providers
//
// Returns all installed Crossplane Providers with their health status.
// Agents should call this first when diagnosing claim or managed resource failures.
package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	providerGVR = schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1",
		Resource: "providers",
	}
	functionGVR = schema.GroupVersionResource{
		Group:    "pkg.crossplane.io",
		Version:  "v1beta1",
		Resource: "functions",
	}
)

func registerProviderTools(s *server.MCPServer, c *client) {
	s.AddTool(
		mcp.NewTool("list_providers",
			mcp.WithDescription(
				"List all installed Crossplane Providers and their health status. "+
					"Returns name, package image, installed revision, and HEALTHY/INSTALLED conditions. "+
					"Always check this first when diagnosing claim or managed resource failures — "+
					"an unhealthy provider will block all reconciliation.",
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return listProviders(ctx, c)
		},
	)
}

// ──────────────────────────────────────────────────────────────────────────────

type providerSummary struct {
	Name            string      `json:"name"`
	Package         string      `json:"package"`
	CurrentRevision string      `json:"currentRevision,omitempty"`
	Healthy         string      `json:"healthy"`
	Installed       string      `json:"installed"`
	Conditions      []Condition `json:"conditions,omitempty"`
}

func listProviders(ctx context.Context, c *client) (*mcp.CallToolResult, error) {
	// Providers
	list, err := c.dynamic.Resource(providerGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results := make([]providerSummary, 0, len(list.Items))
	for _, item := range list.Items {
		obj    := item.Object
		spec   := mapField(obj, "spec")
		status := mapField(obj, "status")
		if spec == nil {
			spec = map[string]interface{}{}
		}
		if status == nil {
			status = map[string]interface{}{}
		}

		conditions := summariseConditions(sliceField(status, "conditions"))
		results = append(results, providerSummary{
			Name:            strField(obj, "metadata", "name"),
			Package:         strField(spec, "package"),
			CurrentRevision: strField(status, "currentRevision"),
			Healthy:         conditionStatus(conditions, "Healthy"),
			Installed:       conditionStatus(conditions, "Installed"),
			Conditions:      conditions,
		})
	}

	// Also include Functions so agent knows what pipeline functions are available
	fnList, err := c.dynamic.Resource(functionGVR).List(ctx, metav1.ListOptions{})
	if err == nil {
		type fnSummary struct {
			Name      string      `json:"name"`
			Package   string      `json:"package"`
			Kind      string      `json:"kind"`
			Healthy   string      `json:"healthy"`
			Installed string      `json:"installed"`
			Conditions []Condition `json:"conditions,omitempty"`
		}
		type response struct {
			Providers []providerSummary `json:"providers"`
			Functions []fnSummary       `json:"functions"`
		}
		fns := make([]fnSummary, 0, len(fnList.Items))
		for _, item := range fnList.Items {
			obj    := item.Object
			spec   := mapField(obj, "spec")
			status := mapField(obj, "status")
			if spec == nil {
				spec = map[string]interface{}{}
			}
			if status == nil {
				status = map[string]interface{}{}
			}
			conds := summariseConditions(sliceField(status, "conditions"))
			fns = append(fns, fnSummary{
				Name:      strField(obj, "metadata", "name"),
				Package:   strField(spec, "package"),
				Kind:      "Function",
				Healthy:   conditionStatus(conds, "Healthy"),
				Installed: conditionStatus(conds, "Installed"),
				Conditions: conds,
			})
		}
		return mcp.NewToolResultText(toJSON(response{Providers: results, Functions: fns})), nil
	}

	return mcp.NewToolResultText(toJSON(results)), nil
}
