package launch_test

import (
	"reflect"
	"testing"

	"github.com/OysterD3/wtmux/internal/agents"
	"github.com/OysterD3/wtmux/internal/launch"
)

func TestBuildArgv_FlagStrategy(t *testing.T) {
	got, err := launch.BuildArgv(
		[]string{"claude"},
		[]string{"/repo/a", "/repo/b"},
		agents.ResolvedStrategy{
			Kind:     agents.StrategyFlag,
			FlagArgs: []string{"--add-dir", "{path}"},
			Source:   agents.SourceBasename,
		},
		[]string{"--mode", "agent"},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"claude", "--add-dir", "/repo/a", "--add-dir", "/repo/b", "--mode", "agent"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBuildArgv_PositionalStrategy(t *testing.T) {
	got, err := launch.BuildArgv(
		[]string{"my-wrapper"},
		[]string{"/repo/a"},
		agents.ResolvedStrategy{Kind: agents.StrategyPositional, Source: agents.SourceFallback},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"my-wrapper", "/repo/a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBuildArgv_EmptySiblings(t *testing.T) {
	got, err := launch.BuildArgv(
		[]string{"claude"},
		nil,
		agents.ResolvedStrategy{
			Kind:     agents.StrategyFlag,
			FlagArgs: []string{"--add-dir", "{path}"},
		},
		[]string{"hello"},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := []string{"claude", "hello"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBuildArgv_NoneStrategyIsError(t *testing.T) {
	_, err := launch.BuildArgv(
		[]string{"opencode"},
		[]string{"/x"},
		agents.ResolvedStrategy{Kind: agents.StrategyNone, AgentID: agents.AgentOpenCode, Source: agents.SourceBasename},
		nil,
	)
	if err == nil {
		t.Fatalf("expected error for none strategy")
	}
}
