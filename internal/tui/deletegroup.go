package tui

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"github.com/OysterD3/wtmux/internal/config"
)

// DeleteGroupWizard runs the delete-a-group flow.
func DeleteGroupWizard(cfg *config.Config) (*config.Config, error) {
	if len(cfg.Groups) == 0 {
		fmt.Fprintln(os.Stderr, "[wtmux] No groups to delete.")
		return nil, nil
	}

	opts := make([]huh.Option[string], 0, len(cfg.Groups))
	for _, g := range cfg.Groups {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%s  (%d repos)", g.Name, len(g.Repos)), g.Name))
	}
	var picked string
	if err := huh.NewSelect[string]().
		Title("Which group to delete?").
		Options(opts...).
		Value(&picked).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	confirm := false
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Delete %q? (Can still be undone via Discard & exit)", picked)).
		Affirmative("Delete").
		Negative("Cancel").
		Value(&confirm).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}
	if !confirm {
		return nil, nil
	}

	next, err := config.RemoveGroup(cfg, picked)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(os.Stderr, "[wtmux] Deleted group %q\n", picked)
	return next, nil
}
