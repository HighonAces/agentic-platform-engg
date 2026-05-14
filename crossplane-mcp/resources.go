// resources.go — Tools: list_claims, list_composite_resources,
//                        list_managed_resources, get_resource_status,
//                        get_resource_events
//
// These tools let an AI agent inspect every layer of the Crossplane resource
// hierarchy: Claims → XRs → Managed Resources (actual cloud objects).
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRD GVR — used to discover managed resource types.
var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

func registerResourceTools(s *server.MCPServer, c *client) {
	// list_claims
	s.AddTool(
		mcp.NewTool("list_claims",
			mcp.WithDescription(
				"List Crossplane Claims (namespace-scoped) across all or one namespace. "+
					"Returns claim name, namespace, kind, SYNCED/READY status, and conditions. "+
					"Claims are the user-facing API; they bind to a cluster-scoped XR.",
			),
			mcp.WithString("namespace",
				mcp.Description("Filter by namespace. Empty string = all namespaces."),
				mcp.DefaultString(""),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			ns := argStr(args, "namespace")
			return listClaims(ctx, c, ns)
		},
	)

	// list_composite_resources
	s.AddTool(
		mcp.NewTool("list_composite_resources",
			mcp.WithDescription(
				"List cluster-scoped Composite Resources (XRs) and the cloud resources they own. "+
					"Returns name, kind, SYNCED/READY, and a list of composed resource refs. "+
					"XRs are the cluster-scoped counterpart of Claims.",
			),
			mcp.WithString("kind",
				mcp.Description("Filter by XR kind, e.g. 'XBucket'. Empty = all kinds."),
				mcp.DefaultString(""),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			kind := argStr(args, "kind")
			return listCompositeResources(ctx, c, kind)
		},
	)

	// list_managed_resources
	s.AddTool(
		mcp.NewTool("list_managed_resources",
			mcp.WithDescription(
				"List raw Managed Resources — the actual cloud objects Crossplane provisions. "+
					"Filter by provider group prefix to limit results. "+
					"Returns name, kind, SYNCED, READY, external-name, and any error message. "+
					"Useful for diagnosing provisioning failures at the cloud API level.",
			),
			mcp.WithString("provider_filter",
				mcp.Description(
					"Filter managed resource types by API group substring. "+
						"Examples: 'storage.gcp', 'gke.gcp', 's3.aws'. Empty = all providers.",
				),
				mcp.DefaultString(""),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			filter := argStr(args, "provider_filter")
			return listManagedResources(ctx, c, filter)
		},
	)

	// get_resource_status
	s.AddTool(
		mcp.NewTool("get_resource_status",
			mcp.WithDescription(
				"Get the full status, conditions, and spec of any Crossplane or Kubernetes resource. "+
					"Provide the API group, version, plural resource name, object name, and (for namespaced resources) namespace. "+
					"Common examples:\n"+
					"  XBucket XR:  group=platform.example.io  version=v1alpha1  plural=xbuckets\n"+
					"  GCS Bucket:  group=storage.gcp.upbound.io  version=v1beta2  plural=buckets\n"+
					"  Claim:       group=platform.example.io  version=v1alpha1  plural=buckets  namespace=default",
			),
			mcp.WithString("group",
				mcp.Description("API group, e.g. 'storage.gcp.upbound.io'"),
				mcp.Required(),
			),
			mcp.WithString("version",
				mcp.Description("API version, e.g. 'v1beta2'"),
				mcp.Required(),
			),
			mcp.WithString("plural",
				mcp.Description("Plural resource name, e.g. 'buckets'"),
				mcp.Required(),
			),
			mcp.WithString("name",
				mcp.Description("Resource name"),
				mcp.Required(),
			),
			mcp.WithString("namespace",
				mcp.Description("Namespace for namespaced resources. Empty for cluster-scoped."),
				mcp.DefaultString(""),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args    := getArgs(req.Params.Arguments)
			group   := argStr(args, "group")
			version := argStr(args, "version")
			plural  := argStr(args, "plural")
			name    := argStr(args, "name")
			ns      := argStr(args, "namespace")
			if group == "" || version == "" || plural == "" || name == "" {
				return mcp.NewToolResultError("group, version, plural, and name are required"), nil
			}
			return getResourceStatus(ctx, c, group, version, plural, name, ns)
		},
	)

	// get_resource_events
	s.AddTool(
		mcp.NewTool("get_resource_events",
			mcp.WithDescription(
				"Get recent Kubernetes Events for a resource. "+
					"Events often surface the root-cause error before it appears in .status.conditions. "+
					"For managed resources and XRs, check 'crossplane-system' namespace. "+
					"For claims, check the claim's own namespace.",
			),
			mcp.WithString("name",
				mcp.Description("Resource name to find events for"),
				mcp.Required(),
			),
			mcp.WithString("namespace",
				mcp.Description("Namespace to search. Use 'crossplane-system' for MRs/XRs."),
				mcp.DefaultString("crossplane-system"),
			),
			mcp.WithString("kind",
				mcp.Description("Optional: filter events by involved object kind, e.g. 'Bucket'."),
				mcp.DefaultString(""),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			name := argStr(args, "name")
			ns   := argStr(args, "namespace")
			kind := argStr(args, "kind")
			if name == "" {
				return mcp.NewToolResultError("name is required"), nil
			}
			if ns == "" {
				ns = "crossplane-system"
			}
			return getResourceEvents(ctx, c, name, ns, kind)
		},
	)
}

// ──────────────────────────────────────────────────────────────────────────────
// list_claims
// ──────────────────────────────────────────────────────────────────────────────

type claimSummary struct {
	Name       string      `json:"name"`
	Namespace  string      `json:"namespace"`
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Synced     string      `json:"synced"`
	Ready      string      `json:"ready"`
	BoundXR    string      `json:"boundXR,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

func listClaims(ctx context.Context, c *client, namespace string) (*mcp.CallToolResult, error) {
	// Discover claim GVRs from XRDs
	xrdList, err := c.dynamic.Resource(xrdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list XRDs: %v", err)), nil
	}

	var results []claimSummary

	for _, xrd := range xrdList.Items {
		obj  := xrd.Object
		spec := mapField(obj, "spec")
		if spec == nil {
			continue
		}
		claimNames := mapField(spec, "claimNames")
		if claimNames == nil {
			continue // XRD doesn't define a claim kind
		}
		group  := strField(spec, "group")
		plural := strField(claimNames, "plural")
		kind   := strField(claimNames, "kind")

		versions := sliceField(spec, "versions")
		version  := servedStorageVersion(versions)
		if version == "" || plural == "" || group == "" {
			continue
		}

		gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: plural}

		var items []interface{}
		if namespace != "" {
			list, err := c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}
			for _, it := range list.Items {
				items = append(items, it.Object)
			}
		} else {
			list, err := c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}
			for _, it := range list.Items {
				items = append(items, it.Object)
			}
		}

		for _, raw := range items {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			status     := mapField(item, "status")
			if status == nil {
				status = map[string]interface{}{}
			}
			conditions := summariseConditions(sliceField(status, "conditions"))
			// Bound XR name lives at status.boundCompositeResource.name in older
			// Crossplane versions, or inferred from status.atProvider in newer ones.
			boundXR := ""
			if bcr := mapField(status, "boundCompositeResource"); bcr != nil {
				boundXR = strField(bcr, "name")
			}

			results = append(results, claimSummary{
				Name:       strField(item, "metadata", "name"),
				Namespace:  strField(item, "metadata", "namespace"),
				Kind:       kind,
				APIVersion: group + "/" + version,
				Synced:     conditionStatus(conditions, "Synced"),
				Ready:      conditionStatus(conditions, "Ready"),
				BoundXR:    boundXR,
				Conditions: conditions,
			})
		}
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(`{"message": "no claims found"}`), nil
	}
	return mcp.NewToolResultText(toJSON(results)), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// list_composite_resources
// ──────────────────────────────────────────────────────────────────────────────

type composedRef struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

type xrSummary struct {
	Name             string        `json:"name"`
	Kind             string        `json:"kind"`
	APIVersion       string        `json:"apiVersion"`
	Synced           string        `json:"synced"`
	Ready            string        `json:"ready"`
	ComposedResources []composedRef `json:"composedResources,omitempty"`
	Conditions       []Condition   `json:"conditions,omitempty"`
}

func listCompositeResources(ctx context.Context, c *client, kindFilter string) (*mcp.CallToolResult, error) {
	xrdList, err := c.dynamic.Resource(xrdGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list XRDs: %v", err)), nil
	}

	var results []xrSummary

	for _, xrd := range xrdList.Items {
		obj  := xrd.Object
		spec := mapField(obj, "spec")
		if spec == nil {
			continue
		}
		kind   := strField(spec, "names", "kind")
		group  := strField(spec, "group")
		plural := strField(spec, "names", "plural")
		if kindFilter != "" && kind != kindFilter {
			continue
		}
		versions := sliceField(spec, "versions")
		version  := servedStorageVersion(versions)
		if version == "" || plural == "" {
			continue
		}

		gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: plural}
		list, err := c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, item := range list.Items {
			obj        := item.Object
			status     := mapField(obj, "status")
			if status == nil {
				status = map[string]interface{}{}
			}
			conditions := summariseConditions(sliceField(status, "conditions"))

			var composed []composedRef
			for _, r := range sliceField(status, "resources") {
				rm, ok := r.(map[string]interface{})
				if !ok {
					continue
				}
				composed = append(composed, composedRef{
					Name:       strField(rm, "name"),
					Kind:       strField(rm, "kind"),
					APIVersion: strField(rm, "apiVersion"),
				})
			}

			results = append(results, xrSummary{
				Name:              strField(obj, "metadata", "name"),
				Kind:              kind,
				APIVersion:        group + "/" + version,
				Synced:            conditionStatus(conditions, "Synced"),
				Ready:             conditionStatus(conditions, "Ready"),
				ComposedResources: composed,
				Conditions:        conditions,
			})
		}
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(`{"message": "no composite resources found"}`), nil
	}
	return mcp.NewToolResultText(toJSON(results)), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// list_managed_resources
// ──────────────────────────────────────────────────────────────────────────────

type mrSummary struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Group        string `json:"group"`
	ExternalName string `json:"externalName,omitempty"`
	Synced       string `json:"synced"`
	Ready        string `json:"ready"`
	Error        string `json:"error,omitempty"`
}

func listManagedResources(ctx context.Context, c *client, providerFilter string) (*mcp.CallToolResult, error) {
	// List CRDs labelled crossplane.io/managed=true
	crdList, err := c.dynamic.Resource(crdGVR).List(ctx, metav1.ListOptions{
		LabelSelector: "crossplane.io/managed=true",
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list CRDs: %v", err)), nil
	}

	var results []mrSummary

	for _, crd := range crdList.Items {
		obj   := crd.Object
		group := strField(obj, "spec", "group")
		if providerFilter != "" && !strings.Contains(group, providerFilter) {
			continue
		}
		kind   := strField(obj, "spec", "names", "kind")
		plural := strField(obj, "spec", "names", "plural")

		versions := sliceField(obj, "spec", "versions")
		version  := servedStorageVersion(versions)
		if version == "" || plural == "" {
			continue
		}

		gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: plural}
		list, err := c.dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, item := range list.Items {
			iobj       := item.Object
			status     := mapField(iobj, "status")
			if status == nil {
				status = map[string]interface{}{}
			}
			conditions := summariseConditions(sliceField(status, "conditions"))
			externalName := ""
			if ann := mapField(iobj, "metadata", "annotations"); ann != nil {
				if v, ok := ann["crossplane.io/external-name"].(string); ok {
					externalName = v
				}
			}
			results = append(results, mrSummary{
				Name:         strField(iobj, "metadata", "name"),
				Kind:         kind,
				Group:        group,
				ExternalName: externalName,
				Synced:       conditionStatus(conditions, "Synced"),
				Ready:        conditionStatus(conditions, "Ready"),
				Error:        firstError(conditions),
			})
		}
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(`{"message": "no managed resources found"}`), nil
	}
	return mcp.NewToolResultText(toJSON(results)), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// get_resource_status
// ──────────────────────────────────────────────────────────────────────────────

type resourceStatus struct {
	Name               string      `json:"name"`
	Namespace          string      `json:"namespace,omitempty"`
	UID                string      `json:"uid"`
	CreationTimestamp  string      `json:"creationTimestamp"`
	ExternalName       string      `json:"externalName,omitempty"`
	Synced             string      `json:"synced"`
	Ready              string      `json:"ready"`
	Conditions         []Condition `json:"conditions,omitempty"`
	ForProvider        interface{} `json:"forProvider,omitempty"`
	AtProvider         interface{} `json:"atProvider,omitempty"`
	Parameters         interface{} `json:"parameters,omitempty"`
	CompositionRef     interface{} `json:"compositionRef,omitempty"`
	ProviderConfigRef  interface{} `json:"providerConfigRef,omitempty"`
}

func getResourceStatus(ctx context.Context, c *client, group, version, plural, name, namespace string) (*mcp.CallToolResult, error) {
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: plural}

	var obj map[string]interface{}
	if namespace != "" {
		res, err := c.dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get %s/%s in %s: %v", plural, name, namespace, err)), nil
		}
		obj = res.Object
	} else {
		res, err := c.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get %s/%s: %v", plural, name, err)), nil
		}
		obj = res.Object
	}

	meta   := mapField(obj, "metadata")
	spec   := mapField(obj, "spec")
	status := mapField(obj, "status")
	if meta == nil {
		meta = map[string]interface{}{}
	}
	if spec == nil {
		spec = map[string]interface{}{}
	}
	if status == nil {
		status = map[string]interface{}{}
	}

	conditions := summariseConditions(sliceField(status, "conditions"))

	externalName := ""
	if ann := mapField(meta, "annotations"); ann != nil {
		if v, ok := ann["crossplane.io/external-name"].(string); ok {
			externalName = v
		}
	}

	result := resourceStatus{
		Name:              strField(meta, "name"),
		Namespace:         strField(meta, "namespace"),
		UID:               strField(meta, "uid"),
		CreationTimestamp: strField(meta, "creationTimestamp"),
		ExternalName:      externalName,
		Synced:            conditionStatus(conditions, "Synced"),
		Ready:             conditionStatus(conditions, "Ready"),
		Conditions:        conditions,
		ForProvider:       spec["forProvider"],
		AtProvider:        status["atProvider"],
		Parameters:        spec["parameters"],
		CompositionRef:    spec["compositionRef"],
		ProviderConfigRef: spec["providerConfigRef"],
	}
	return mcp.NewToolResultText(toJSON(result)), nil
}

// ──────────────────────────────────────────────────────────────────────────────
// get_resource_events
// ──────────────────────────────────────────────────────────────────────────────

type eventSummary struct {
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	Count     int32  `json:"count"`
	Age       string `json:"age"`
	Source    string `json:"source,omitempty"`
}

func getResourceEvents(ctx context.Context, c *client, name, namespace, kind string) (*mcp.CallToolResult, error) {
	// Check in both the given namespace and crossplane-system (MRs/XRs emit events there)
	namespaces := []string{namespace}
	if namespace != "crossplane-system" {
		namespaces = append(namespaces, "crossplane-system")
	}

	var results []eventSummary

	for _, ns := range namespaces {
		evList, err := c.kubernetes.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", name),
		})
		if err != nil {
			continue
		}
		for _, ev := range evList.Items {
			if kind != "" && ev.InvolvedObject.Kind != kind {
				continue
			}
			age := ""
			if !ev.LastTimestamp.IsZero() {
				age = ageFromTimestamp(ev.LastTimestamp.UTC().Format("2006-01-02T15:04:05Z"))
			}
			source := ""
			if ev.Source.Component != "" {
				source = ev.Source.Component
			}
			results = append(results, eventSummary{
				Namespace: ns,
				Kind:      ev.InvolvedObject.Kind,
				Reason:    ev.Reason,
				Type:      ev.Type,
				Message:   ev.Message,
				Count:     ev.Count,
				Age:       age,
				Source:    source,
			})
		}
	}

	if len(results) == 0 {
		return mcp.NewToolResultText(`{"message": "no events found"}`), nil
	}
	return mcp.NewToolResultText(toJSON(results)), nil
}
