// zeb is the entrypoint for the Zebra Link command-line client.
// Command wiring lives under internal/cli; this file only resolves version
// metadata and passes it into the root command.
package main

import (
	"runtime/debug"
	"strings"

	"github.com/zeb-link/zeb/internal/cli"
)

// devVersion is what a build reports when it has no better answer.
const devVersion = "dev"

// version is injected at build time by `make build` (-X main.version), which
// takes it from npm/package.json. `go install` runs no ldflags and leaves it at
// the sentinel, so resolveVersion falls back to the module version the Go
// toolchain records in the binary.
var version = devVersion

func main() {
	cli.Execute(resolveVersion())
}

func resolveVersion() string {
	if version != devVersion {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return devVersion
	}
	return normalizeModuleVersion(info.Main.Version)
}

// normalizeModuleVersion converts a Go module version into the bare form the
// CLI prints. Builds from a working tree report "(devel)"; released builds
// report a tag like "v0.1.0"; untagged commits report a pseudo-version.
func normalizeModuleVersion(v string) string {
	if v == "" || v == "(devel)" {
		return devVersion
	}
	return strings.TrimPrefix(v, "v")
}
