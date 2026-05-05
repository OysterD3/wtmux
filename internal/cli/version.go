package cli

import "runtime/debug"

// defaultVersion is the in-source version baked into local `go build`
// outputs. CI release builds override `version` via -ldflags; Version()
// detects the override by comparing against this constant, so it must
// stay the single source of truth — bumping just one of `version` or the
// override-detection check would silently skip the VCS suffix fallback.
const defaultVersion = "0.0.1"

// version is the wtmux release version. CI release builds override it via
// -ldflags at build time:
//
//	go build -ldflags="-X github.com/OysterD3/wtmux/internal/cli.version=v0.0.1" ./cmd/wtmux
//
// Local `go build` (no ldflags) reports defaultVersion, optionally
// suffixed with the VCS short SHA + "-dirty" — see Version().
var version = defaultVersion

// Version returns the user-facing version string. Resolution order:
//  1. The package-level `version` if it differs from defaultVersion — i.e.
//     it's been overridden via -ldflags in a CI release build.
//  2. defaultVersion suffixed with vcs.revision[:7] (and "-dirty" when
//     uncommitted) for local `go build` from a git clone.
//  3. Bare defaultVersion when no buildinfo is available.
//
// Note: `go install pkg@<version>` users see the same form as local
// builds — the module ref is intentionally not consulted, so the
// constant is the single source of truth for non-CI builds.
func Version() string {
	if version != defaultVersion {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	var rev, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				rev = s.Value[:7]
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if rev != "" {
		return version + "-" + rev + dirty
	}
	return version
}
