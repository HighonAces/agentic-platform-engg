// xrds.go — Tools: list_xrds, list_compositions
//
// These tools give the agent a complete picture of what infrastructure types
// the platform exposes and which compositions implement them.
package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Crossplane API group/version/resources
var (
	xrdGVR = schema.GroupVersionResource{
		Group:    "apiextensions.crossplane.io",
		Version:  "v1",
		Resource: "compositeresourcedefinitions",
	}
	compositionGVR = schema.GroupVersionResource{
		Group:    "apiextensions.crossplane.io",
		Version:  "v1",
		Resource: "compositions",
	}
)

func registerXRDTools(s *server.MCPServer, c *client) {
	s.AddTool(
		mcp.NewTool("list_xrds",
			mcp.WithDescription(
				"List all CompositeResourceDefinitions (XRDs). "+
					"Returns each type's group, kind, claim kind, and ESTABLISHED status. "+
					"Use this first to understand what infrastructure types the platform exposes.",
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return listXRDs(ctx, c)
		},
	)

	s.AddTool(
		mcp.NewTool("list_compositions",
			mcp.WithDescription(
				"List all Compositions and the XRD kind each one implements. "+
					"Shows the pipeline steps (functions) used for resource rendering.",
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return listCompositions(ctx, c)
		},
	)
}

// ──────────────────────────────────────────────────────────────────────────────

type xrdSummary struct {
	Name        string      `json:"name"`
	Group       string      `json:"group"`
	Kind        string      `json:"kind"`
	ClaimKind   string      `json:"claimKind,omitempty"`
	Established string      `json:"established"`
	Conditions  []Condition `json:"conditions,omitempty"`
}

func listXRDs(ctx context.Context, c *client) (*mcp.CallToolResult, error) {
	list, err := c.dynamic.Resource(xrdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results := make([]xrdSummary, 0, len(list.Items))
	for _, item := range list.Items {
		obj := item.Object
		spec := mapField(obj, "spec")
		if spec == nil {
			spec = map[string]interface{}{}
		}
		status := mapField(obj, "status")
		if status == nil {
			status = map[string]interface{}{}
		}

		conditions := summariseConditions(sliceField(status, "conditions"))

		results = append(results, xrdSummary{
			Name:        strField(obj, "metadata", "name"),
			Group:       strField(spec, "group"),
			Kind:        strField(spec, "names", "kind"),
			ClaimKind:   strField(spec, "claimNames", "kind"),
			Established: conditionStatus(conditions, "Established"),
			Conditions:  conditions,
		})
	}
	return mcp.NewToolResultText(toJSON(results)), nil
}

// ──────────────────────────────────────────────────────────────────────────────

type pipelineStep struct {
	Step     string `json:"step"`
	Function string `json:"function"`
}

type compositionSummary struct {
	Name             string         `json:"name"`
	CompositeKind    string         `json:"compositeKind"`
	CompositeAPIVer  string         `json:"compositeApiVersion"`
	PipelineSteps    []pipelineStep `json:"pipelineSteps,omitempty"`
	Labels           map[string]interface{} `json:"labels,omitempty"`
}

func listCompositions(ctx context.Context, c *client) (*mcp.CallToolResult, error) {
	list, err := c.dynamic.Resource(compositionGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	results := make([]compositionSummary, 0, len(list.Items))
	for _, item := range list.Items {
		obj  := item.Object
		spec := mapField(obj, "spec")
		if spec == nil {
			spec = map[string]interface{}{}
		}
		ref := mapField(spec, "compositeTypeRef")
		if ref == nil {
			ref = map[string]interface{}{}
		}

		var steps []pipelineStep
		for _, s := range sliceField(spec, "pipeline") {
			sm, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			fn := mapField(sm, "functionRef")
			fnName := ""
			if fn != nil {
				fnName = strField(fn, "name")
			}
			steps = append(steps, pipelineStep{
				Step:     strField(sm, "step"),
				Function: fnName,
			})
		}

		labels := mapField(obj, "metadata", "labels")

		results = append(results, compositionSummary{
			Name:            strField(obj, "metadata", "name"),
			CompositeKind:   strField(ref, "kind"),
			CompositeAPIVer: strField(ref, "apiVersion"),
			PipelineSteps:   steps,
			Labels:          labels,
		})
	}
	return mcp.NewToolResultText(toJSON(results)), nil
}
