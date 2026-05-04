package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/OysterD3/wtmux/internal/config"
)

// MainMenuResult is the outcome of MainMenu. Save reports whether the
// caller should persist Config to disk; Config is the (possibly mutated)
// state to persist.
type MainMenuResult struct {
	Config *config.Config
	Save   bool
}

// MainMenu runs the top-level config-edit loop. Returns once the user
// chooses save/exit or discard.
func MainMenu(initial *config.Config, targetPath, cwd string) (MainMenuResult, error) {
	current := initial
	dirty := false

	for {
		printSummary(current, targetPath, dirty)

		options := []huh.Option[string]{
			huh.NewOption("Create a new group", "create"),
		}
		if len(current.Groups) > 0 {
			options = append(options,
				huh.NewOption("Edit a group", "edit"),
				huh.NewOption("Delete a group", "delete"),
			)
		}
		saveLabel := "Exit"
		discardLabel := "Cancel"
		if dirty {
			saveLabel = "Save & exit"
			discardLabel = "Discard changes & exit"
		}
		options = append(options,
			huh.NewOption(saveLabel, "save"),
			huh.NewOption(discardLabel, "discard"),
		)

		action := "create"
		err := huh.NewSelect[string]().
			Title("What do you want to do?").
			Options(options...).
			Value(&action).
			Run()
		if err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				if !dirty {
					return MainMenuResult{Config: current, Save: false}, nil
				}
				confirm := true
				cerr := huh.NewConfirm().
					Title("Unsaved changes — save before exiting?").
					Affirmative("Save").
					Negative("Discard").
					Value(&confirm).
					Run()
				if cerr != nil {
					if errors.Is(cerr, huh.ErrUserAborted) {
						continue
					}
					return MainMenuResult{}, cerr
				}
				return MainMenuResult{Config: current, Save: confirm}, nil
			}
			return MainMenuResult{}, err
		}

		switch action {
		case "create":
			next, err := CreateGroupWizard(current, cwd)
			if err != nil {
				return MainMenuResult{}, err
			}
			if next != nil {
				current = next
				dirty = true
			}
		case "edit":
			next, err := EditGroupWizard(current, cwd)
			if err != nil {
				return MainMenuResult{}, err
			}
			if next != nil {
				current = next
				dirty = true
			}
		case "delete":
			next, err := DeleteGroupWizard(current)
			if err != nil {
				return MainMenuResult{}, err
			}
			if next != nil {
				current = next
				dirty = true
			}
		case "save":
			return MainMenuResult{Config: current, Save: dirty}, nil
		case "discard":
			return MainMenuResult{Config: current, Save: false}, nil
		}
	}
}

func printSummary(c *config.Config, targetPath string, dirty bool) {
	suffix := ""
	if dirty {
		suffix = "  (unsaved changes)"
	}
	var b strings.Builder
	b.WriteString("\n--- wtmux config ---\n")
	fmt.Fprintf(&b, "Editing: %s%s\n\n", targetPath, suffix)
	if len(c.Groups) == 0 {
		b.WriteString("Groups: (no groups yet)\n")
	} else {
		b.WriteString("Groups:\n")
		for _, g := range c.Groups {
			fmt.Fprintf(&b, "  • %s (%d repos)\n", g.Name, len(g.Repos))
		}
	}
	b.WriteString("--------------------\n")
	fmt.Fprint(os.Stderr, b.String())
}
