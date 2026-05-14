package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ArgoCDClient struct {
	BaseURL  string
	Token    string
	HTTPClient *http.Client
}

func NewArgoCDClient(baseURL, token string) *ArgoCDClient {
	// Respect NODE_TLS_REJECT_UNAUTHORIZED if set, or default to skipping TLS for local/port-forwarded instances
	skipTLS := os.Getenv("NODE_TLS_REJECT_UNAUTHORIZED") == "0" || os.Getenv("ARGOCD_INSECURE") == "true"

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLS},
	}

	return &ArgoCDClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (c *ArgoCDClient) request(method, path string, query map[string]string, body interface{}) ([]byte, error) {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, err
	}

	if query != nil {
		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("argocd api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Tool Implementation Methods

func (c *ArgoCDClient) ListApplications(search string) (interface{}, error) {
	query := make(map[string]string)
	if search != "" {
		query["search"] = search
	}
	data, err := c.request("GET", "/api/v1/applications", query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) ListClusters() (interface{}, error) {
	data, err := c.request("GET", "/api/v1/clusters", nil, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetApplication(name, namespace string) (interface{}, error) {
	query := make(map[string]string)
	if namespace != "" {
		query["appNamespace"] = namespace
	}
	data, err := c.request("GET", "/api/v1/applications/"+name, query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetApplicationResourceTree(name, namespace string) (interface{}, error) {
	query := make(map[string]string)
	if namespace != "" {
		query["appNamespace"] = namespace
	}
	data, err := c.request("GET", "/api/v1/applications/"+name+"/resource-tree", query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetApplicationManagedResources(name string, filters map[string]string) (interface{}, error) {
	data, err := c.request("GET", "/api/v1/applications/"+name+"/managed-resources", filters, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetApplicationEvents(name, namespace string) (interface{}, error) {
	query := make(map[string]string)
	if namespace != "" {
		query["appNamespace"] = namespace
	}
	data, err := c.request("GET", "/api/v1/applications/"+name+"/events", query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) SyncApplication(name string, options map[string]interface{}) (interface{}, error) {
	data, err := c.request("POST", "/api/v1/applications/"+name+"/sync", nil, options)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) DeleteApplication(name string, query map[string]string) (interface{}, error) {
	data, err := c.request("DELETE", "/api/v1/applications/"+name, query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) CreateApplication(app interface{}) (interface{}, error) {
	data, err := c.request("POST", "/api/v1/applications", nil, app)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) UpdateApplication(name string, app interface{}) (interface{}, error) {
	data, err := c.request("PUT", "/api/v1/applications/"+name, nil, app)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetResource(appName, appNs string, resourceRef map[string]string) (interface{}, error) {
	query := make(map[string]string)
	for k, v := range resourceRef {
		query[k] = v
	}
	if appNs != "" {
		query["appNamespace"] = appNs
	}
	data, err := c.request("GET", "/api/v1/applications/"+appName+"/resource", query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetResourceEvents(appName, appNs, uid, ns, name string) (interface{}, error) {
	query := map[string]string{
		"appNamespace":      appNs,
		"resourceNamespace": ns,
		"resourceUID":       uid,
		"resourceName":      name,
	}
	data, err := c.request("GET", "/api/v1/applications/"+appName+"/events", query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetResourceActions(appName, appNs string, resourceRef map[string]string) (interface{}, error) {
	query := make(map[string]string)
	for k, v := range resourceRef {
		query[k] = v
	}
	if appNs != "" {
		query["appNamespace"] = appNs
	}
	data, err := c.request("GET", "/api/v1/applications/"+appName+"/resource/actions", query, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) RunResourceAction(appName, appNs string, resourceRef map[string]string, action string) (interface{}, error) {
	query := make(map[string]string)
	for k, v := range resourceRef {
		query[k] = v
	}
	if appNs != "" {
		query["appNamespace"] = appNs
	}
	data, err := c.request("POST", "/api/v1/applications/"+appName+"/resource/actions", query, action)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func (c *ArgoCDClient) GetLogs(appName, appNs string, query map[string]string) (interface{}, error) {
	if appNs != "" {
		query["appNamespace"] = appNs
	}
	// Note: For simplicity in Go, we'll fetch a static chunk of logs. 
	// ArgoCD API logs are usually streamed (SSE), but we'll try a regular GET first with tailLines.
	if _, ok := query["tailLines"]; !ok {
		query["tailLines"] = "100"
	}
	data, err := c.request("GET", "/api/v1/applications/"+appName+"/logs", query, nil)
	if err != nil {
		return nil, err
	}
	
	// Logs are returned as newline-separated JSON objects
	lines := strings.Split(string(data), "\n")
	var logEntries []map[string]interface{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			logEntries = append(logEntries, entry)
		}
	}
	return logEntries, nil
}
