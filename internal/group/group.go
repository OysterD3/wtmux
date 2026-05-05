// Package group resolves which configured group (or single repo) a wtmux
// invocation is operating on, given the current working directory and an
// optional --group flag.
package group

import (
	"errors"
	"fmt"
	"strings"

	"github.com/OysterD3/wtmux/internal/config"
	"github.com/OysterD3/wtmux/internal/git"
	"github.com/OysterD3/wtmux/internal/paths"
)

// determinePrimary returns the repo in g.Repos whose realpath matches
// cwd's git toplevel realpath, or g.Repos[0] when cwd isn't sitting inside
// any of the configured repos. Returns the original (un-realpathed) path
// so downstream preflight/git calls see the path the user wrote in their
// config.
func determinePrimary(cwd string, g *config.Group) string {
	top, ok, err := git.GetToplevel(cwd)
	if err != nil || !ok {
		return g.Repos[0]
	}
	realTop := paths.RealpathSafe(top)
	for _, r := range g.Repos {
		if paths.RealpathSafe(r) == realTop {
			return r
		}
	}
	return g.Repos[0]
}

// Kind identifies which of the three resolution outcomes was reached.
type Kind int

const (
	// KindOutside means cwd is not inside any git repository.
	KindOutside Kind = iota
	// KindSingle means cwd is inside a git repo, but no configured group
	// matches that repo. The repo's realpath is in Resolution.Single.
	KindSingle
	// KindGroup means cwd is inside a configured group, or matched via
	// --group. Resolution.Group and Resolution.Primary are populated.
	KindGroup
)

// String returns a debug-friendly label for the kind.
func (k Kind) String() string {
	switch k {
	case KindOutside:
		return "outside"
	case KindSingle:
		return "single"
	case KindGroup:
		return "group"
	default:
		return "unknown"
	}
}

// Resolution is the discriminated result of a Resolve call. Exactly the
// fields appropriate to Kind are populated; the rest are zero.
type Resolution struct {
	Kind    Kind
	Group   *config.Group // populated only when Kind == KindGroup
	Primary string        // populated only when Kind == KindGroup; one of Group.Repos
	Single  string        // populated only when Kind == KindSingle; realpath of cwd's git toplevel
}

// ResolveInput configures Resolve.
type ResolveInput struct {
	Cwd       string
	Config    *config.Config
	GroupFlag string // value of --group, "" if unset
}

// Sentinel errors. Callers use errors.Is.
var (
	// ErrGroupFlagNotFound is returned when --group <name> doesn't match
	// any configured group's Name.
	ErrGroupFlagNotFound = errors.New("group: --group does not match any configured group")
	// ErrAmbiguousGroups is returned when cwd matches more than one
	// configured group. The wrapped error includes the comma-joined group
	// names for surface text.
	ErrAmbiguousGroups = errors.New("group: cwd matches more than one configured group")
)

// Resolve walks the resolution algorithm and returns the matching
// Resolution, or one of the sentinel errors. Always returns a non-nil
// *Resolution on err == nil.
func Resolve(in ResolveInput) (*Resolution, error) {
	if in.GroupFlag != "" {
		for i := range in.Config.Groups {
			g := &in.Config.Groups[i]
			if g.Name == in.GroupFlag {
				return &Resolution{
					Kind:    KindGroup,
					Group:   g,
					Primary: determinePrimary(in.Cwd, g),
				}, nil
			}
		}
		return nil, ErrGroupFlagNotFound
	}

	top, ok, err := git.GetToplevel(in.Cwd)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &Resolution{Kind: KindOutside}, nil
	}
	realTop := paths.RealpathSafe(top)

	var matches []*config.Group
	for i := range in.Config.Groups {
		g := &in.Config.Groups[i]
		for _, r := range g.Repos {
			if paths.RealpathSafe(r) == realTop {
				matches = append(matches, g)
				break
			}
		}
	}

	switch len(matches) {
	case 0:
		return &Resolution{Kind: KindSingle, Single: realTop}, nil
	case 1:
		g := matches[0]
		return &Resolution{
			Kind:    KindGroup,
			Group:   g,
			Primary: determinePrimary(in.Cwd, g),
		}, nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.Name
		}
		return nil, fmt.Errorf("%w: %s", ErrAmbiguousGroups, strings.Join(names, ", "))
	}
}
