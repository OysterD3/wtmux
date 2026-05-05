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

// EditGroupWizard runs the edit-a-group flow. Returns nil on cancel without
// changes; otherwise the mutated config.
func EditGroupWizard(cfg *config.Config, cwd string) (*config.Config, error) {
	if len(cfg.Groups) == 0 {
		fmt.Fprintln(os.Stderr, "[wtmux] No groups to edit.")
		return nil, nil
	}

	groupOpts := make([]huh.Option[string], 0, len(cfg.Groups))
	for _, g := range cfg.Groups {
		groupOpts = append(groupOpts, huh.NewOption(fmt.Sprintf("%s  (%d repos)", g.Name, len(g.Repos)), g.Name))
	}
	var picked string
	if err := huh.NewSelect[string]().
		Title("Which group?").
		Options(groupOpts...).
		Value(&picked).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	current := cfg
	activeName := picked
	mutated := false

	for {
		var field string
		if err := huh.NewSelect[string]().
			Title(fmt.Sprintf("Editing %q — which field?", activeName)).
			Options(
				huh.NewOption("Rename group", "name"),
				huh.NewOption("Edit repos", "repos"),
				huh.NewOption("Override symlinkDirectories", "symlinks"),
				huh.NewOption("Override worktreePathPattern", "pattern"),
				huh.NewOption("Override launchCommand", "launch"),
				huh.NewOption("Override agent (sibling-injection strategy)", "agent"),
				huh.NewOption("← Back to main menu", "back"),
			).
			Value(&field).
			Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) || field == "back" {
				if !mutated {
					return nil, nil
				}
				return current, nil
			}
			return nil, err
		}
		if field == "back" {
			if !mutated {
				return nil, nil
			}
			return current, nil
		}

		idx := findGroupIndex(current, activeName)
		if idx < 0 {
			return current, nil
		}
		group := current.Groups[idx]

		switch field {
		case "name":
			others := make([]string, 0, len(current.Groups)-1)
			for _, g := range current.Groups {
				if g.Name != group.Name {
					others = append(others, g.Name)
				}
			}
			newName := group.Name
			if err := huh.NewInput().
				Title("New name").
				Value(&newName).
				Validate(func(v string) error { return ValidateGroupName(others, v) }).
				Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return nil, err
			}
			renamed := strings.TrimSpace(newName)
			next, err := config.RenameGroup(current, activeName, renamed)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			activeName = renamed
			mutated = true
		case "repos":
			next, ok, err := editRepos(current, activeName, cwd)
			if err != nil {
				return nil, err
			}
			if ok {
				current = next
				mutated = true
			}
		case "symlinks":
			input := strings.Join(group.SymlinkDirectories, ", ")
			if err := huh.NewInput().
				Title("symlinkDirectories (comma-separated, blank to unset override)").
				Placeholder(strings.Join(current.SymlinkDirectories, ", ")).
				Value(&input).
				Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return nil, err
			}
			updated := group
			if strings.TrimSpace(input) != "" {
				updated.SymlinkDirectories = ParseCommaList(input)
			} else {
				updated.SymlinkDirectories = nil
			}
			next, err := config.UpdateGroup(current, activeName, updated)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			mutated = true
		case "pattern":
			input := group.WorktreePathPattern
			if err := huh.NewInput().
				Title("worktreePathPattern (blank to unset override)").
				Placeholder(current.WorktreePathPattern).
				Value(&input).
				Validate(ValidateWorktreePattern).
				Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return nil, err
			}
			updated := group
			if strings.TrimSpace(input) != "" {
				updated.WorktreePathPattern = strings.TrimSpace(input)
			} else {
				updated.WorktreePathPattern = ""
			}
			next, err := config.UpdateGroup(current, activeName, updated)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			mutated = true
		case "launch":
			input := strings.Join(group.LaunchCommand, " ")
			if err := huh.NewInput().
				Title("launchCommand (space-separated, blank to unset override)").
				Placeholder(strings.Join(current.LaunchCommand, " ")).
				Value(&input).
				Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return nil, err
			}
			updated := group
			if strings.TrimSpace(input) != "" {
				updated.LaunchCommand = ParseLaunchCommand(input)
			} else {
				updated.LaunchCommand = nil
			}
			next, err := config.UpdateGroup(current, activeName, updated)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			mutated = true
		case "agent":
			result, err := AgentSelect(AgentSelectInput{
				Title:             "Agent (sibling-injection strategy)",
				CurrentAgent:      group.Agent,
				CurrentAddDirArgs: group.AddDirArgs,
			})
			if err != nil {
				return nil, err
			}
			if result.Cancelled {
				continue
			}
			updated := group
			updated.Agent = result.Agent
			updated.AddDirArgs = result.AddDirArgs
			next, err := config.UpdateGroup(current, activeName, updated)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			mutated = true
		}
	}
}

func editRepos(cfg *config.Config, groupName, cwd string) (*config.Config, bool, error) {
	current := cfg
	mutated := false
	for {
		idx := findGroupIndex(current, groupName)
		if idx < 0 {
			return current, mutated, nil
		}
		group := current.Groups[idx]

		baseNames := make([]string, 0, len(group.Repos))
		for _, r := range group.Repos {
			baseNames = append(baseNames, filepath.Base(r))
		}
		var action string
		removeHint := ""
		if len(group.Repos) <= 2 {
			removeHint = " (need at least 2)"
		}
		if err := huh.NewSelect[string]().
			Title(fmt.Sprintf("Repos (%d): %s", len(group.Repos), strings.Join(baseNames, ", "))).
			Options(
				huh.NewOption("Add a repo", "add"),
				huh.NewOption("Remove repos"+removeHint, "remove"),
				huh.NewOption("← Done", "done"),
			).
			Value(&action).
			Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return current, mutated, nil
			}
			return nil, false, err
		}

		switch action {
		case "done":
			return current, mutated, nil
		case "add":
			toAdd, cancelled, err := collectReposToAdd(cwd, group.Repos)
			if err != nil {
				return nil, false, err
			}
			if cancelled || len(toAdd) == 0 {
				continue
			}
			next, err := config.AddReposToGroup(current, groupName, toAdd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			mutated = true
		case "remove":
			if len(group.Repos) <= 2 {
				fmt.Fprintln(os.Stderr, "[wtmux] Cannot remove — a group must have at least 2 repos.")
				continue
			}
			opts := make([]huh.Option[string], 0, len(group.Repos))
			for _, r := range group.Repos {
				opts = append(opts, huh.NewOption(filepath.Base(r)+"  "+r, r))
			}
			var picked []string
			if err := huh.NewMultiSelect[string]().
				Title("Select repos to remove").
				Options(opts...).
				Value(&picked).
				Run(); err != nil {
				if errors.Is(err, huh.ErrUserAborted) {
					continue
				}
				return nil, false, err
			}
			if len(picked) == 0 {
				continue
			}
			next, err := config.RemoveReposFromGroup(current, groupName, picked)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[wtmux] %s\n", err)
				continue
			}
			current = next
			mutated = true
		}
	}
}

func collectReposToAdd(cwd string, alreadyIn []string) ([]string, bool, error) {
	siblings := DetectSiblings(cwd)
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

	var added []string
	if len(filtered) > 0 {
		opts := make([]huh.Option[string], 0, len(filtered))
		for _, s := range filtered {
			opts = append(opts, huh.NewOption(filepath.Base(s)+"  "+s, s))
		}
		if err := huh.NewMultiSelect[string]().
			Title("Which sibling repos to add?").
			Options(opts...).
			Value(&added).
			Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return nil, true, nil
			}
			return nil, false, err
		}
	}

	wantManual := len(added) == 0
	if err := huh.NewConfirm().
		Title("Add manual paths?").
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
				Title("Repo path (blank to finish)").
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
			if containsString(alreadyIn, result.Resolved) || containsString(added, result.Resolved) {
				fmt.Fprintln(os.Stderr, "[wtmux] Already added.")
				continue
			}
			added = append(added, result.Resolved)
		}
	}

	return added, false, nil
}
