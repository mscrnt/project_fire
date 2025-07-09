package version

import (
	"fmt"
	"runtime"
)

// GetVersion returns a formatted version string
func GetVersion(version, commit, buildTime string) string {
	if version == "" {
		version = "dev"
	}
	if commit != "" && len(commit) > 7 {
		commit = commit[:7]
	}
	return fmt.Sprintf("%s-%s", version, commit)
}

// GetDetailedVersion returns detailed version information
func GetDetailedVersion(version, commit, buildTime string) string {
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}

	return fmt.Sprintf(`F.I.R.E. (Full Intensity Rigorous Evaluation)
Version:    %s
Commit:     %s
Built:      %s
Go version: %s
OS/Arch:    %s/%s`,
		version, commit, buildTime,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH)
}