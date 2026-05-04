package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/OysterD3/wtmux/internal/config"
)

// CreateGroupWizard prompts the user through new-group creation. Returns
// the mutated config on success; (nil, nil) if the user cancelled at any
// step.
func CreateGroupWizard(cfg *config.Config, cwd string) (*config.Config, error) {
	existingNames := make([]string, 0, len(cfg.Groups))
	for _, g := range cfg.Groups {
		existingNames = append(existingNames, g.Name)
	}

	var name string
	if err := huh.NewInput().
		Title("Group name").
		Value(&name).
		Validate(func(v string) error { return ValidateGroupName(existingNames, v) }).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}
	name = strings.TrimSpace(name)

	repos, cancelled, err := collectRepos(cwd, nil)
	if err != nil {
		return nil, err
	}
	if cancelled {
		return nil, nil
	}
	if len(repos) < 2 {
		fmt.Fprintln(os.Stderr, "[wtmux] A group needs at least 2 repos. Aborting.")
		return nil, nil
	}

	next, err := config.AddGroup(cfg, config.Group{Name: name, Repos: repos})
	if err != nil {
		return nil, err
	}

	wantOverrides := false
	if err := huh.NewConfirm().
		Title("Configure overrides for this group?").
		Value(&wantOverrides).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	if wantOverrides {
		next, err = collectOverrides(next, name)
		if err != nil {
			return nil, err
		}
	}

	fmt.Fprintf(os.Stderr, "[wtmux] Created group %q (%d repos)\n", name, len(repos))
	return next, nil
}

func collectRepos(cwd string, alreadyIn []string) ([]string, bool, error) {
	siblings := DetectSiblings(cwd)
	if len(alreadyIn) > 0 {
		excluded := map[string]bool{}
		for _, r := range alreadyIn {
			excluded[r] = true
		}
		filtered := make([]string, 0, len(siblings))
		for _, s := range siblings {
			if !excluded[s] {
				filtered = append(filtered, s)
			}
		}
		siblings = filtered
	}

	var selected []string
	if len(siblings) > 0 {
		opts := make([]huh.Option[string], 0, len(siblings))
		for _, s := range siblings {
			opts = append(opts, huh.NewOption(filepath.Base(s)+"  "+s, s))
		}
		if err := huh.NewMultiSelect[string]().
			Title("Which sibling repos?").
			Options(opts...).
			Value(&selected).
			Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil, true, nil
			}
			return nil, false, err
		}
	} else {
		fmt.Fprintln(os.Stderr, "[wtmux] No sibling git repos detected in the parent directory.")
	}

	wantManual := len(selected) < 2
	if err := huh.NewConfirm().
		Title("Add any other repo paths manually?").
		Value(&wantManual).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, true, nil
		}
		return nil, false, err
	}

	if wantManual {
		for {
			var raw string
			if err := huh.NewInput().
				Title(fmt.Sprintf("Repo path (%d entered; blank to finish)", len(selected))).
				Placeholder("~/code/another-repo").
				Value(&raw).
				Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					return nil, true, nil
				}
				return nil, false, err
			}
			if strings.TrimSpace(raw) == "" {
				break
			}
			result := ResolveRepoPath(raw)
			if !result.OK {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", result.Reason)
				continue
			}
			if containsString(selected, result.Resolved) || containsString(alreadyIn, result.Resolved) {
				fmt.Fprintln(os.Stderr, "[wtmux] Already added.")
				continue
			}
			selected = append(selected, result.Resolved)
		}
	}

	return selected, false, nil
}

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func collectOverrides(cfg *config.Config, groupName string) (*config.Config, error) {
	idx := findGroupIndex(cfg, groupName)
	if idx < 0 {
		return cfg, nil
	}
	g := cfg.Groups[idx]

	symlinks := strings.Join(g.SymlinkDirectories, ", ")
	if err := huh.NewInput().
		Title("Override symlinkDirectories? (comma-separated, blank to skip)").
		Placeholder(strings.Join(cfg.SymlinkDirectories, ", ")).
		Value(&symlinks).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return cfg, nil
		}
		return nil, err
	}

	pattern := g.WorktreePathPattern
	if err := huh.NewInput().
		Title("Override worktreePathPattern? (blank to skip, must include {name})").
		Placeholder(cfg.WorktreePathPattern).
		Value(&pattern).
		Validate(ValidateWorktreePattern).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return cfg, nil
		}
		return nil, err
	}

	launch := strings.Join(g.LaunchCommand, " ")
	if err := huh.NewInput().
		Title("Override launchCommand? (space-separated, blank to skip)").
		Placeholder(strings.Join(cfg.LaunchCommand, " ")).
		Value(&launch).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return cfg, nil
		}
		return nil, err
	}

	agent, err := AgentSelect(AgentSelectInput{
		Title:             "Agent (sibling-injection strategy, optional)",
		CurrentAgent:      g.Agent,
		CurrentAddDirArgs: g.AddDirArgs,
	})
	if err != nil {
		return nil, err
	}

	updated := g
	if strings.TrimSpace(symlinks) != "" {
		updated.SymlinkDirectories = ParseCommaList(symlinks)
	}
	if strings.TrimSpace(pattern) != "" {
		updated.WorktreePathPattern = strings.TrimSpace(pattern)
	}
	if strings.TrimSpace(launch) != "" {
		updated.LaunchCommand = ParseLaunchCommand(launch)
	}
	if !agent.Cancelled {
		updated.Agent = agent.Agent
		updated.AddDirArgs = agent.AddDirArgs
	}

	return config.UpdateGroup(cfg, groupName, updated)
}

func findGroupIndex(c *config.Config, name string) int {
	for i, g := range c.Groups {
		if g.Name == name {
			return i
		}
	}
	return -1
}
