package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound is returned by Discover/Load when no config file exists at any
// of the four discovery sources. Callers can errors.Is this to distinguish
// "no config anywhere → use defaults" from "user pointed at a missing file".
var ErrNotFound = errors.New("config: not found")

// DiscoverySource identifies which of the four sources matched.
type DiscoverySource int

const (
	SourceFlag DiscoverySource = iota
	SourceEnv
	SourceWalk
	SourceXDG
)

// String returns a debug-friendly label for the source.
func (s DiscoverySource) String() string {
	switch s {
	case SourceFlag:
		return "flag"
	case SourceEnv:
		return "env"
	case SourceWalk:
		return "walk"
	case SourceXDG:
		return "xdg"
	default:
		return fmt.Sprintf("DiscoverySource(%d)", int(s))
	}
}

// DiscoveryInputs configures Discover. Env is a function rather than a map so
// tests can inject deterministic values without touching process state.
type DiscoveryInputs struct {
	Explicit string              // value of --config flag, "" if unset
	Cwd      string              // starting point for the upward walk
	Home     string              // "" → resolved via os.UserHomeDir()
	Env      func(string) string // nil → os.Getenv
}

// Discovery is the result of a successful Discover call.
type Discovery struct {
	Path   string
	Source DiscoverySource
}

// Discover walks the four sources (flag → env → upward .wtmux.json → XDG)
// in order, returning the first hit. Returns ErrNotFound when every source
// is exhausted. Explicit-but-missing paths return a non-ErrNotFound error.
func Discover(in DiscoveryInputs) (*Discovery, error) {
	env := in.Env
	if env == nil {
		env = os.Getenv
	}

	if in.Explicit != "" {
		if !exists(in.Explicit) {
			return nil, fmt.Errorf("config: file not found: %s", in.Explicit)
		}
		return &Discovery{Path: in.Explicit, Source: SourceFlag}, nil
	}

	if envPath := env("WTMUX_CONFIG"); envPath != "" {
		if !exists(envPath) {
			return nil, fmt.Errorf("config: WTMUX_CONFIG not found: %s", envPath)
		}
		return &Discovery{Path: envPath, Source: SourceEnv}, nil
	}

	if walked := walkUpward(in.Cwd); walked != "" {
		return &Discovery{Path: walked, Source: SourceWalk}, nil
	}

	home := in.Home
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("config: cannot determine home directory: %w", err)
		}
		home = h
	}
	xdgHome := env("XDG_CONFIG_HOME")
	if xdgHome == "" {
		xdgHome = filepath.Join(home, ".config")
	}
	xdg := filepath.Join(xdgHome, "wtmux", "config.json")
	if exists(xdg) {
		return &Discovery{Path: xdg, Source: SourceXDG}, nil
	}

	return nil, ErrNotFound
}

// walkUpward looks for .wtmux.json starting at cwd and ascending until it
// reaches the filesystem root. Returns the first hit, or "" when none.
func walkUpward(cwd string) string {
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".wtmux.json")
		if exists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// exists is a thin os.Stat-based existence check (separate from
// paths.Exists, which uses Lstat — discovery doesn't care about broken
// symlinks differently).
func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
