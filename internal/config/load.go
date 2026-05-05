package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/OysterD3/wtmux/internal/paths"
)

// LoadInputs configures Load. Embeds DiscoveryInputs so all discovery
// knobs are reachable via the same struct.
type LoadInputs struct {
	DiscoveryInputs
}

// Loaded is the result of a successful Load call.
type Loaded struct {
	Config *Config
	Path   string
	Source DiscoverySource
}

// Load runs the full load pipeline:
//
//	Discover → ReadFile → json.Unmarshal → expandTildeOnGroupRepos →
//	applyDefaults → Validate → return.
//
// ErrNotFound from Discover propagates verbatim so callers can opt to use
// in-memory defaults. JSON syntax errors and validation errors are returned
// directly; the caller distinguishes via errors.As against ValidationErrors.
func Load(in LoadInputs) (*Loaded, error) {
	d, err := Discover(in.DiscoveryInputs)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(d.Path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", d.Path, err)
	}

	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("config: invalid JSON in %s: %w", d.Path, err)
	}

	expandTildeOnGroupRepos(&c)
	c.applyDefaults()

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &Loaded{Config: &c, Path: d.Path, Source: d.Source}, nil
}

// expandTildeOnGroupRepos rewrites every Group.Repos entry that begins with
// "~" or "~/" into an absolute path. Mirrors TS — top-level fields don't
// accept paths, so only Group.Repos needs the pass.
func expandTildeOnGroupRepos(c *Config) {
	for i := range c.Groups {
		for j, r := range c.Groups[i].Repos {
			if r == "~" || strings.HasPrefix(r, "~/") {
				c.Groups[i].Repos[j] = paths.ExpandTilde(r)
			}
		}
	}
}
