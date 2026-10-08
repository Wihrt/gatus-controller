package controller

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// alertTypeEnumFromGo returns the alert type enum declared by the kubebuilder marker on GatusAlertSpec.Type.
func alertTypeEnumFromGo(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("../../api/v1alpha1/gatusendpoint_types.go")
	if err != nil {
		t.Fatalf("failed to read types file: %v", err)
	}
	m := regexp.MustCompile(`Enum=([^\n]+)\n\s*Type\s`).FindSubmatch(src)
	if m == nil {
		t.Fatal("alert type Enum marker not found")
	}
	return strings.Split(string(m[1]), ";")
}

// collectTypeEnums gathers the enum of every schema property named "type".
func collectTypeEnums(node any, out *[][]string) {
	switch n := node.(type) {
	case map[string]any:
		for k, v := range n {
			if prop, ok := v.(map[string]any); ok && k == "type" {
				if list, ok := prop["enum"].([]any); ok {
					var vals []string
					for _, e := range list {
						vals = append(vals, e.(string))
					}
					*out = append(*out, vals)
				}
			}
			collectTypeEnums(v, out)
		}
	case []any:
		for _, v := range n {
			collectTypeEnums(v, out)
		}
	}
}

func TestCRDAlertTypeEnumMatchesGoMarker(t *testing.T) {
	want := alertTypeEnumFromGo(t)

	// Gatus expects "aws-ses", not "awsses" (alerting/alert/type.go in Gatus v5.37.0).
	found := false
	for _, v := range want {
		if v == "awsses" {
			t.Error("enum contains 'awsses', Gatus expects 'aws-ses'")
		}
		found = found || v == "aws-ses"
	}
	if !found {
		t.Error("enum should contain 'aws-ses'")
	}

	for _, file := range []string{
		"monitoring.gatus.io_gatusendpoints.yaml",
		"monitoring.gatus.io_gatusexternalendpoints.yaml",
	} {
		raw, err := os.ReadFile("../../charts/gatus-controller/crds/" + file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}
		var doc any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("failed to parse %s: %v", file, err)
		}
		var enums [][]string
		collectTypeEnums(doc, &enums)
		if len(enums) != 1 {
			t.Fatalf("%s: expected 1 alert type enum, got %d", file, len(enums))
		}
		if !reflect.DeepEqual(enums[0], want) {
			t.Errorf("%s: alert type enum = %v, want %v", file, enums[0], want)
		}
	}
}
