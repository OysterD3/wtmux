package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/OysterD3/wtmux/internal/config"
	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
	wtlog "github.com/OysterD3/wtmux/internal/log"
	"github.com/OysterD3/wtmux/internal/tui"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config",
		Short:         "Interactively edit wtmux config",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			applyVerbose()
			return runConfig()
		},
	}
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		helpToStdout(configHelp)
	})
	return cmd
}

func runConfig() error {
	cwd, err := mustGetwd()
	if err != nil {
		return err
	}

	target, cancelled, err := resolveConfigTarget(cwd)
	if err != nil {
		return err
	}
	if cancelled {
		fmt.Fprintln(os.Stderr, "[wtmux] No changes.")
		return nil
	}

	initial, ok, err := loadInitialConfig(target)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "[wtmux] No changes.")
		return nil
	}

	res, err := tui.MainMenu(initial, target, cwd)
	if err != nil {
		return wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
	}

	if res.Save {
		if err := config.Save(target, res.Config); err != nil {
			return wtmuxerrors.New(wtmuxerrors.KindInternal, "config save failed: %s", err.Error())
		}
		fmt.Fprintf(os.Stderr, "[wtmux] Saved to %s\n", target)
		return nil
	}
	fmt.Fprintln(os.Stderr, "[wtmux] Exited without saving.")
	return nil
}

// resolveConfigTarget returns the path to use for editing. Order:
//  1. Explicit --config flag (used as-is, even if missing).
//  2. config.Discover.
//  3. Prompt the user for project (.wtmux.json in cwd) or user
//     (~/.config/wtmux/config.json).
func resolveConfigTarget(cwd string) (string, bool, error) {
	if pf.configPath != "" {
		return pf.configPath, false, nil
	}
	d, err := config.Discover(config.DiscoveryInputs{Cwd: cwd})
	if err == nil {
		return d.Path, false, nil
	}
	if !errors.Is(err, config.ErrNotFound) {
		return "", false, wtmuxerrors.New(wtmuxerrors.KindUser, "%s", err.Error())
	}

	proceed := true
	if perr := huh.NewConfirm().
		Title("No config found. Create one now?").
		Value(&proceed).
		Run(); perr != nil {
		if errors.Is(perr, huh.ErrUserAborted) {
			return "", true, nil
		}
		return "", false, wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", perr.Error())
	}
	if !proceed {
		return "", true, nil
	}

	home, herr := os.UserHomeDir()
	if herr != nil {
		return "", false, wtmuxerrors.New(wtmuxerrors.KindInternal, "cannot determine home: %s", herr.Error())
	}
	projectPath := filepath.Join(cwd, ".wtmux.json")
	userPath := filepath.Join(home, ".config", "wtmux", "config.json")

	choice := "project"
	if perr := huh.NewSelect[string]().
		Title("Where should the new config live?").
		Options(
			huh.NewOption("Project — "+projectPath, "project"),
			huh.NewOption("User — "+userPath, "user"),
			huh.NewOption("Cancel", "cancel"),
		).
		Value(&choice).
		Run(); perr != nil {
		if errors.Is(perr, huh.ErrUserAborted) {
			return "", true, nil
		}
		return "", false, wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", perr.Error())
	}
	switch choice {
	case "project":
		return projectPath, false, nil
	case "user":
		return userPath, false, nil
	default:
		return "", true, nil
	}
}

// loadInitialConfig reads the file at target. If the file doesn't exist,
// returns a defaulted config with no groups (used as the starting point
// for a brand new config). On JSON or schema errors, prompts the user
// whether to start with an empty config — returning ok=false if they
// decline. Other I/O errors propagate.
func loadInitialConfig(target string) (*config.Config, bool, error) {
	raw, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			c := config.Default()
			c.Groups = nil
			return c, true, nil
		}
		return nil, false, wtmuxerrors.New(wtmuxerrors.KindInternal, "could not read %s: %s", target, err.Error())
	}

	var c config.Config
	if jerr := json.Unmarshal(raw, &c); jerr != nil {
		fmt.Fprintf(os.Stderr, "[wtmux] Invalid JSON in %s: %s\n", target, jerr.Error())
		return askEmptyFallback()
	}
	c.ApplyDefaults()
	if verr := c.Validate(); verr != nil {
		fmt.Fprintf(os.Stderr, "[wtmux] Schema errors in %s:\n%s\n", target, verr.Error())
		return askEmptyFallback()
	}
	return &c, true, nil
}

func askEmptyFallback() (*config.Config, bool, error) {
	fallback := false
	if err := huh.NewConfirm().
		Title("Start with an empty config instead? (existing file will only be overwritten on save)").
		Value(&fallback).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, false, nil
		}
		return nil, false, wtmuxerrors.New(wtmuxerrors.KindInternal, "%s", err.Error())
	}
	if !fallback {
		return nil, false, nil
	}
	c := config.Default()
	c.Groups = nil
	return c, true, nil
}

// silence unused-import warning if log isn't used here (it stays for parity)
var _ = wtlog.Info
