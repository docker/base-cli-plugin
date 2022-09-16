/*
Package internal contains all build time metadata (version, build time, git commit, etc).
*/
package internal

import (
	"fmt"
	"runtime"
)

// build-time arguments
var (
	version = "n/a"
	commit  = "n/a"
)

// Version information from build time args and environment
type Version struct {
	Version   string
	Commit    string
	GoVersion string
	Compiler  string
	Platform  string
}

// FromBuild provides all version details
func FromBuild() Version {
	return Version{
		Version:   fmt.Sprintf("v%s", version),
		Commit:    commit,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
