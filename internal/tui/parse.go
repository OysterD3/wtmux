package tui

import "strings"

// ParseCommaList splits input on ",", trims each piece, and drops empties.
func ParseCommaList(input string) []string {
	parts := strings.Split(input, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// FormatCommaList joins items with ", ".
func FormatCommaList(items []string) string {
	return strings.Join(items, ", ")
}

// ParseLaunchCommand splits on whitespace runs and drops empties.
func ParseLaunchCommand(input string) []string {
	parts := strings.Fields(input)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// FormatLaunchCommand joins argv with single spaces.
func FormatLaunchCommand(argv []string) string {
	return strings.Join(argv, " ")
}
