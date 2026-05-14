// helpers.go — Shared types and utility functions.
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ──────────────────────────────────────────────────────────────────────────────
// Shared types
// ──────────────────────────────────────────────────────────────────────────────

// Condition is a trimmed representation of a Kubernetes condition.
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
	Age     string `json:"age,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Unstructured helpers
// ──────────────────────────────────────────────────────────────────────────────

// strField returns a string field from an unstructured map, or "".
func strField(obj map[string]interface{}, fields ...string) string {
	val, found, _ := unstructured.NestedString(obj, fields...)
	if !found {
		return ""
	}
	return val
}

// boolField returns a bool field, or false.
func boolField(obj map[string]interface{}, fields ...string) bool {
	val, found, _ := unstructured.NestedBool(obj, fields...)
	if !found {
		return false
	}
	return val
}

// sliceField returns a []interface{} field, or nil.
func sliceField(obj map[string]interface{}, fields ...string) []interface{} {
	val, found, _ := unstructured.NestedSlice(obj, fields...)
	if !found {
		return nil
	}
	return val
}

// mapField returns a nested map, or nil.
func mapField(obj map[string]interface{}, fields ...string) map[string]interface{} {
	val, found, _ := unstructured.NestedMap(obj, fields...)
	if !found {
		return nil
	}
	return val
}

// ──────────────────────────────────────────────────────────────────────────────
// Condition summarisation
// ──────────────────────────────────────────────────────────────────────────────

// summariseConditions parses .status.conditions from an unstructured object.
func summariseConditions(raw []interface{}) []Condition {
	out := make([]Condition, 0, len(raw))
	for _, item := range raw {
		c, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		age := ageFromTimestamp(strField(c, "lastTransitionTime"))
		out = append(out, Condition{
			Type:    strField(c, "type"),
			Status:  strField(c, "status"),
			Reason:  strField(c, "reason"),
			Message: strField(c, "message"),
			Age:     age,
		})
	}
	return out
}

// conditionStatus returns the Status of the first condition matching cType.
func conditionStatus(conditions []Condition, cType string) string {
	for _, c := range conditions {
		if c.Type == cType {
			return c.Status
		}
	}
	return "Unknown"
}

// firstError returns the message of the first False condition.
func firstError(conditions []Condition) string {
	for _, c := range conditions {
		if c.Status == "False" && c.Message != "" {
			return c.Message
		}
	}
	return ""
}

// ──────────────────────────────────────────────────────────────────────────────
// Formatting helpers
// ──────────────────────────────────────────────────────────────────────────────

// ageFromTimestamp converts an RFC3339 timestamp to a human-readable age string.
func ageFromTimestamp(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	d := time.Since(t)
	mins := int(d.Minutes())
	switch {
	case mins < 1:
		return "<1m"
	case mins < 60:
		return fmt.Sprintf("%dm", mins)
	default:
		return fmt.Sprintf("%dh%dm", mins/60, mins%60)
	}
}

// toJSON marshals v to a pretty-printed JSON string.
// Returns an error string if marshalling fails.
func toJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}

// getArgs safely type-asserts req.Params.Arguments (typed as `any` in mcp-go v0.52+)
// to map[string]interface{}. Returns an empty map if the assertion fails.
func getArgs(raw any) map[string]interface{} {
	if m, ok := raw.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// argStr extracts a string from an args map, trimming whitespace.
func argStr(args map[string]interface{}, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// servedStorageVersion returns the first served+storage version name from a
// CRD's spec.versions list, or "" if none found.
func servedStorageVersion(versions []interface{}) string {
	for _, v := range versions {
		vm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if boolField(vm, "served") && boolField(vm, "storage") {
			return strField(vm, "name")
		}
	}
	// Fall back to first served version
	for _, v := range versions {
		vm, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if boolField(vm, "served") {
			return strField(vm, "name")
		}
	}
	return ""
}
