// Package version provides version information for klip
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version number
	Version = "2.0.0"

	// GitCommit is the git commit hash (populated at build time)
	GitCommit = "unknown"

	// BuildDate is the build timestamp (populated at build time)
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()

	// Platform is the OS/Arch combination
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info contains all version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetInfo returns version information as a struct
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// String returns a formatted version string
func String() string {
	return fmt.Sprintf("klip %s (commit: %s, built: %s, %s)",
		Version, GitCommit, BuildDate, Platform)
}

// ShortString returns just the version number
func ShortString() string {
	return Version
}
