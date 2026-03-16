package windows

import "testing"

func TestNormalizeMountPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty-is-root", input: "", want: "/"},
		{name: "dot-is-root", input: ".", want: "/"},
		{name: "slashes", input: `\\src\\pkg\\main.go`, want: "/src/pkg/main.go"},
		{name: "relative", input: `src/pkg`, want: "/src/pkg"},
		{name: "dedupe", input: `/src//pkg///main.go`, want: "/src/pkg/main.go"},
		{name: "reject-parent", input: `src/../pkg`, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeMountPath(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeMountPath(%q) error = nil, want error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeMountPath(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeMountPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestJoinMountPath(t *testing.T) {
	if got := JoinMountPath(`/src`, `pkg\\main.go`); got != "/src/pkg/main.go" {
		t.Fatalf("JoinMountPath() = %q, want %q", got, "/src/pkg/main.go")
	}
}
