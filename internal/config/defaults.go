package config

// applyDefaults fills in zero-valued top-level fields with the wtmux defaults.
// It must run AFTER unmarshal and BEFORE Validate so the abspath/oneof/min
// checks see fully-populated values. Group-level overrides are NOT defaulted
// — they remain nil/empty so preflight.BuildPlan can see "user did not
// override this" and fall back to the top-level field at plan-build time.
func (c *Config) applyDefaults() {
	if len(c.SymlinkDirectories) == 0 {
		c.SymlinkDirectories = []string{"node_modules", ".env"}
	}
	if c.WorktreePathPattern == "" {
		c.WorktreePathPattern = ".worktrees/{name}"
	}
	if len(c.LaunchCommand) == 0 {
		c.LaunchCommand = []string{"claude"}
	}
}

// Default returns a fresh Config populated with wtmux defaults and no
// groups. Used by the CLI when no config file is discoverable.
func Default() *Config {
	c := &Config{}
	c.applyDefaults()
	return c
}

// ApplyDefaults is the exported counterpart of applyDefaults, for callers
// outside the package that need to fill zero-valued top-level fields
// before validating.
func (c *Config) ApplyDefaults() { c.applyDefaults() }
