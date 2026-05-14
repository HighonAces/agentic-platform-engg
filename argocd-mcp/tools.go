package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerTools(s *server.MCPServer, c *ArgoCDClient) {
	// 1. list_applications
	s.AddTool(
		mcp.NewTool("list_applications",
			mcp.WithDescription("list_applications returns list of applications"),
			mcp.WithString("search", mcp.Description("Search applications by name. Partial match, no glob patterns.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			return handleResult(c.ListApplications(argStr(args, "search")))
		},
	)

	// 2. list_clusters
	s.AddTool(
		mcp.NewTool("list_clusters",
			mcp.WithDescription("list_clusters returns list of clusters registered with ArgoCD"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleResult(c.ListClusters())
		},
	)

	// 3. get_application
	s.AddTool(
		mcp.NewTool("get_application",
			mcp.WithDescription("get_application returns application by application name. Optionally specify the application namespace."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			return handleResult(c.GetApplication(argStr(args, "applicationName"), argStr(args, "applicationNamespace")))
		},
	)

	// 4. get_application_resource_tree
	s.AddTool(
		mcp.NewTool("get_application_resource_tree",
			mcp.WithDescription("get_application_resource_tree returns resource tree for application by application name."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			return handleResult(c.GetApplicationResourceTree(argStr(args, "applicationName"), argStr(args, "applicationNamespace")))
		},
	)

	// 5. get_application_managed_resources
	s.AddTool(
		mcp.NewTool("get_application_managed_resources",
			mcp.WithDescription("get_application_managed_resources returns managed resources for application by application name with optional filtering."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("kind", mcp.Description("Filter by Kubernetes resource kind")),
			mcp.WithString("namespace", mcp.Description("Filter by Kubernetes namespace")),
			mcp.WithString("name", mcp.Description("Filter by resource name")),
			mcp.WithString("version", mcp.Description("Filter by resource API version")),
			mcp.WithString("group", mcp.Description("Filter by API group")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			filters := make(map[string]string)
			for _, k := range []string{"kind", "namespace", "name", "version", "group"} {
				if v := argStr(args, k); v != "" {
					filters[k] = v
				}
			}
			return handleResult(c.GetApplicationManagedResources(argStr(args, "applicationName"), filters))
		},
	)

	// 6. get_application_workload_logs
	s.AddTool(
		mcp.NewTool("get_application_workload_logs",
			mcp.WithDescription("get_application_workload_logs returns logs for application workload (Deployment, StatefulSet, Pod, etc.)"),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application"), mcp.Required()),
			mcp.WithString("container", mcp.Description("Container name"), mcp.Required()),
			mcp.WithString("resourceName", mcp.Description("Name of the workload resource"), mcp.Required()),
			mcp.WithString("resourceKind", mcp.Description("Kind of the workload resource"), mcp.Required()),
			mcp.WithString("resourceNamespace", mcp.Description("Namespace of the workload resource"), mcp.Required()),
			mcp.WithString("resourceGroup", mcp.Description("Group of the workload resource")),
			mcp.WithString("resourceVersion", mcp.Description("Version of the workload resource")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			query := map[string]string{
				"container":    argStr(args, "container"),
				"resourceName": argStr(args, "resourceName"),
				"kind":         argStr(args, "resourceKind"),
				"namespace":    argStr(args, "resourceNamespace"),
				"group":        argStr(args, "resourceGroup"),
				"version":      argStr(args, "resourceVersion"),
			}
			return handleResult(c.GetLogs(argStr(args, "applicationName"), argStr(args, "applicationNamespace"), query))
		},
	)

	// 7. get_application_events
	s.AddTool(
		mcp.NewTool("get_application_events",
			mcp.WithDescription("get_application_events returns events for application by application name."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			return handleResult(c.GetApplicationEvents(argStr(args, "applicationName"), argStr(args, "applicationNamespace")))
		},
	)

	// 8. sync_application
	s.AddTool(
		mcp.NewTool("sync_application",
			mcp.WithDescription("sync_application syncs application."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application")),
			mcp.WithString("revision", mcp.Description("Sync to a specific revision")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			options := make(map[string]interface{})
			if v := argStr(args, "applicationNamespace"); v != "" {
				options["appNamespace"] = v
			}
			if v := argStr(args, "revision"); v != "" {
				options["revision"] = v
			}
			return handleResult(c.SyncApplication(argStr(args, "applicationName"), options))
		},
	)

	// 9. delete_application
	s.AddTool(
		mcp.NewTool("delete_application",
			mcp.WithDescription("delete_application deletes application."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application")),
			mcp.WithBoolean("cascade", mcp.Description("Whether to cascade deletion")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			query := make(map[string]string)
			if v := argStr(args, "applicationNamespace"); v != "" {
				query["appNamespace"] = v
			}
			if v, ok := args["cascade"].(bool); ok && v {
				query["cascade"] = "true"
			}
			return handleResult(c.DeleteApplication(argStr(args, "applicationName"), query))
		},
	)

	// 10. create_application
	s.AddTool(
		mcp.NewTool("create_application",
			mcp.WithDescription("create_application creates a new ArgoCD application."),
			mcp.WithObject("application", mcp.Description("ArgoCD Application object"), mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			app := args["application"]
			if app == nil {
				return mcp.NewToolResultError("missing application object"), nil
			}
			return handleResult(c.CreateApplication(app))
		},
	)

	// 11. update_application
	s.AddTool(
		mcp.NewTool("update_application",
			mcp.WithDescription("update_application updates application."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithObject("application", mcp.Description("ArgoCD Application object"), mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			app := args["application"]
			if app == nil {
				return mcp.NewToolResultError("missing application object"), nil
			}
			return handleResult(c.UpdateApplication(argStr(args, "applicationName"), app))
		},
	)

	// 12. get_resources
	s.AddTool(
		mcp.NewTool("get_resources",
			mcp.WithDescription("get_resources return manifests for resources managed by the application."),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application"), mcp.Required()),
			mcp.WithString("resourceName", mcp.Description("Name of the resource")),
			mcp.WithString("resourceKind", mcp.Description("Kind of the resource")),
			mcp.WithString("resourceNamespace", mcp.Description("Namespace of the resource")),
			mcp.WithString("resourceGroup", mcp.Description("Group of the resource")),
			mcp.WithString("resourceVersion", mcp.Description("Version of the resource")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			ref := map[string]string{
				"resourceName": argStr(args, "resourceName"),
				"kind":         argStr(args, "resourceKind"),
				"namespace":    argStr(args, "resourceNamespace"),
				"group":        argStr(args, "resourceGroup"),
				"version":      argStr(args, "resourceVersion"),
			}
			return handleResult(c.GetResource(argStr(args, "applicationName"), argStr(args, "applicationNamespace"), ref))
		},
	)

	// 13. get_resource_events
	s.AddTool(
		mcp.NewTool("get_resource_events",
			mcp.WithDescription("get_resource_events returns events for a resource that is managed by an application"),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application"), mcp.Required()),
			mcp.WithString("resourceUID", mcp.Description("UID of the resource"), mcp.Required()),
			mcp.WithString("resourceNamespace", mcp.Description("Namespace of the resource"), mcp.Required()),
			mcp.WithString("resourceName", mcp.Description("Name of the resource"), mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			return handleResult(c.GetResourceEvents(
				argStr(args, "applicationName"),
				argStr(args, "applicationNamespace"),
				argStr(args, "resourceUID"),
				argStr(args, "resourceNamespace"),
				argStr(args, "resourceName"),
			))
		},
	)

	// 14. run_resource_action
	s.AddTool(
		mcp.NewTool("run_resource_action",
			mcp.WithDescription("run_resource_action runs an action on a resource"),
			mcp.WithString("applicationName", mcp.Description("Name of the application"), mcp.Required()),
			mcp.WithString("applicationNamespace", mcp.Description("ArgoCD namespace of the application"), mcp.Required()),
			mcp.WithString("action", mcp.Description("Action name"), mcp.Required()),
			mcp.WithString("resourceName", mcp.Description("Name of the resource"), mcp.Required()),
			mcp.WithString("resourceKind", mcp.Description("Kind of the resource"), mcp.Required()),
			mcp.WithString("resourceNamespace", mcp.Description("Namespace of the resource"), mcp.Required()),
			mcp.WithString("resourceGroup", mcp.Description("Group of the resource")),
			mcp.WithString("resourceVersion", mcp.Description("Version of the resource")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := getArgs(req.Params.Arguments)
			ref := map[string]string{
				"resourceName": argStr(args, "resourceName"),
				"kind":         argStr(args, "resourceKind"),
				"namespace":    argStr(args, "resourceNamespace"),
				"group":        argStr(args, "resourceGroup"),
				"version":      argStr(args, "resourceVersion"),
			}
			return handleResult(c.RunResourceAction(
				argStr(args, "applicationName"),
				argStr(args, "applicationNamespace"),
				ref,
				argStr(args, "action"),
			))
		},
	)
}
