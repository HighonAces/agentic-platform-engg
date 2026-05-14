// client.go — Kubernetes client initialisation.
//
// Priority: in-cluster config → KUBECONFIG env → ~/.kube/config.
// Set MCP_KUBE_CONTEXT to select a non-default context.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// client wraps both a dynamic client (for CRDs / custom resources) and a
// typed client (for Events, which benefit from the strongly-typed API).
type client struct {
	dynamic    dynamic.Interface
	kubernetes kubernetes.Interface
}

func newClient() (*client, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		overrides := &clientcmd.ConfigOverrides{}
		if ctx := os.Getenv("MCP_KUBE_CONTEXT"); ctx != "" {
			overrides.CurrentContext = ctx
		}
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, overrides,
		).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("cannot build k8s config: %w", err)
		}
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}
	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("typed client: %w", err)
	}
	return &client{dynamic: dyn, kubernetes: kube}, nil
}
