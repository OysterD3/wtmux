package tui_test

import (
	"reflect"
	"testing"

	"github.com/OysterD3/wtmux/internal/tui"
)

func TestParseCommaList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{",,a,,", []string{"a"}},
	}
	for _, c := range cases {
		got := tui.ParseCommaList(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("ParseCommaList(%q): got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseLaunchCommand(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", []string{}},
		{"claude --dangerous", []string{"claude", "--dangerous"}},
		{"  a  b   c  ", []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		got := tui.ParseLaunchCommand(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("ParseLaunchCommand(%q): got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := tui.FormatCommaList([]string{"a", "b"}); got != "a, b" {
		t.Fatalf("FormatCommaList: %q", got)
	}
	if got := tui.FormatLaunchCommand([]string{"a", "b"}); got != "a b" {
		t.Fatalf("FormatLaunchCommand: %q", got)
	}
}
