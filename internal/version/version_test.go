package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		buildTime string
		want      string
	}{
		{
			name:    "all values provided",
			version: "v1.0.0",
			commit:  "abcdef1234567890",
			want:    "v1.0.0-abcdef1",
		},
		{
			name:   "empty version",
			commit: "abcdef1234567890",
			want:   "dev-abcdef1",
		},
		{
			name:    "short commit",
			version: "v1.0.0",
			commit:  "abc",
			want:    "v1.0.0-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetVersion(tt.version, tt.commit, tt.buildTime)
			if got != tt.want {
				t.Errorf("GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDetailedVersion(t *testing.T) {
	result := GetDetailedVersion("v1.0.0", "abcdef1234567890", "2024-01-01T00:00:00Z")

	// Check that required elements are present
	if !strings.Contains(result, "F.I.R.E.") {
		t.Error("GetDetailedVersion() should contain F.I.R.E.")
	}
	if !strings.Contains(result, "Version:    v1.0.0") {
		t.Error("GetDetailedVersion() should contain version")
	}
	if !strings.Contains(result, "Commit:     abcdef1234567890") {
		t.Error("GetDetailedVersion() should contain commit")
	}
	if !strings.Contains(result, "Go version:") {
		t.Error("GetDetailedVersion() should contain Go version")
	}
	if !strings.Contains(result, "OS/Arch:") {
		t.Error("GetDetailedVersion() should contain OS/Arch")
	}
}
