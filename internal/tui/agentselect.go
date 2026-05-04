package tui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/OysterD3/wtmux/internal/agents"
)

// AgentChoice is the result of AgentSelect. The TUI surfaces three states:
// "auto-detect" (Agent="" and AddDirArgs=nil), one of the registry agents
// (Agent set, AddDirArgs=nil), or a custom addDirArgs template
// (Agent="" and AddDirArgs populated). Cancelled distinguishes Esc/Ctrl-C
// from a real selection.
type AgentChoice struct {
	Cancelled  bool
	Agent      agents.AgentID
	AddDirArgs []string
}

// AgentSelectInput configures AgentSelect.
type AgentSelectInput struct {
	Title             string
	CurrentAgent      agents.AgentID
	CurrentAddDirArgs []string
}

const (
	agentSelectUnset  = "__unset__"
	agentSelectCustom = "__custom__"
)

// AgentSelect runs the agent picker. Returns Cancelled=true when the user
// aborts the form (Esc/Ctrl-C).
func AgentSelect(in AgentSelectInput) (AgentChoice, error) {
	options := []huh.Option[string]{
		huh.NewOption("auto-detect (from launchCommand)", agentSelectUnset),
	}
	for _, id := range agents.RegistryIDs() {
		label := string(id)
		if agents.Registry[id].Kind == agents.StrategyNone {
			label = string(id) + " (no multi-root)"
		}
		options = append(options, huh.NewOption(label, string(id)))
	}
	options = append(options, huh.NewOption("custom (configure addDirArgs)", agentSelectCustom))

	initial := agentSelectUnset
	if in.CurrentAgent != "" {
		initial = string(in.CurrentAgent)
	} else if len(in.CurrentAddDirArgs) > 0 {
		initial = agentSelectCustom
	}

	choice := initial
	if err := huh.NewSelect[string]().
		Title(in.Title).
		Options(options...).
		Value(&choice).
		Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return AgentChoice{Cancelled: true}, nil
		}
		return AgentChoice{Cancelled: true}, err
	}

	switch choice {
	case agentSelectUnset:
		return AgentChoice{}, nil
	case agentSelectCustom:
		template := "--add-dir {path}"
		if len(in.CurrentAddDirArgs) > 0 {
			template = strings.Join(in.CurrentAddDirArgs, " ")
		}
		if err := huh.NewInput().
			Title("addDirArgs template (space-separated, use {path} for sibling)").
			Placeholder("--add-dir {path}").
			Value(&template).
			Validate(func(s string) error {
				if !strings.Contains(s, "{path}") {
					return errors.New("must contain {path}")
				}
				return nil
			}).
			Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				return AgentChoice{Cancelled: true}, nil
			}
			return AgentChoice{Cancelled: true}, err
		}
		return AgentChoice{AddDirArgs: ParseLaunchCommand(template)}, nil
	default:
		return AgentChoice{Agent: agents.AgentID(choice)}, nil
	}
}
