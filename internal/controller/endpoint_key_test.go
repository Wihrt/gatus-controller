package controller

import "testing"

func TestGatusEndpointKey(t *testing.T) {
	tests := []struct {
		name, group, epName, want string
	}{
		{"no group", "", "My Service", "_my-service"},
		{"group and name", "Core", "API", "core_api"},
		{"lowercase", "PROD", "Web", "prod_web"},
		{"trim spaces", "  core ", " api  ", "core_api"},
		{"slash", "a/b", "c/d", "a-b_c-d"},
		{"underscore", "a_b", "c_d", "a-b_c-d"},
		{"dot", "a.b", "c.d", "a-b_c-d"},
		{"comma", "a,b", "c,d", "a-b_c-d"},
		{"space", "a b", "c d", "a-b_c-d"},
		{"hash", "a#b", "c#d", "a-b_c-d"},
		{"plus", "a+b", "c+d", "a-b_c-d"},
		{"ampersand", "a&b", "c&d", "a-b_c-d"},
		{"other characters kept", "a:b", "c(d)", "a:b_c(d)"},
		{"underscore in name only", "a", "b_c", "a_b-c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gatusEndpointKey(tt.group, tt.epName); got != tt.want {
				t.Errorf("gatusEndpointKey(%q, %q) = %q, want %q", tt.group, tt.epName, got, tt.want)
			}
		})
	}
}
