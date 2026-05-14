package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func toJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}

func getArgs(raw any) map[string]interface{} {
	if m, ok := raw.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func argStr(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func argBool(args map[string]interface{}, key string) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return false
}

func argMap(args map[string]interface{}, key string) map[string]string {
	if v, ok := args[key].(map[string]interface{}); ok {
		m := make(map[string]string)
		for k, val := range v {
			m[k] = fmt.Sprintf("%v", val)
		}
		return m
	}
	return nil
}

func argSlice(args map[string]interface{}, key string) []string {
	if v, ok := args[key].([]interface{}); ok {
		s := make([]string, 0, len(v))
		for _, val := range v {
			s = append(s, fmt.Sprintf("%v", val))
		}
		return s
	}
	return nil
}

func argInterface(args map[string]interface{}, key string) interface{} {
	return args[key]
}

func handleResult(res interface{}, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(toJSON(res)), nil
}
