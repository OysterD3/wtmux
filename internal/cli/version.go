package cli

// version is the wtmux release version. Set via -ldflags at build time:
//
//	go build -ldflags="-X github.com/OysterD3/wtmux/internal/cli.version=0.0.1" ./cmd/wtmux
//
// Local builds default to "dev".
var version = "dev"
