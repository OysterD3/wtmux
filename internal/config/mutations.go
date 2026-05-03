package config

import "fmt"

// AddGroup returns a copy of c with g appended to Groups. The result is
// re-validated; the input is not mutated.
func AddGroup(c *Config, g Group) (*Config, error) {
	for _, existing := range c.Groups {
		if existing.Name == g.Name {
			return nil, fmt.Errorf("duplicate group name: %s", g.Name)
		}
	}
	out := cloneConfig(c)
	out.Groups = append(out.Groups, cloneGroup(g))
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveGroup returns a copy of c with the group named name removed.
func RemoveGroup(c *Config, name string) (*Config, error) {
	idx := -1
	for i, g := range c.Groups {
		if g.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	out := cloneConfig(c)
	out.Groups = append(out.Groups[:idx:idx], out.Groups[idx+1:]...)
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// RenameGroup returns a copy of c with the group named oldName renamed to
// newName. Same-name renames are no-ops; renaming onto an existing name
// fails with a duplicate-name error.
func RenameGroup(c *Config, oldName, newName string) (*Config, error) {
	idx := -1
	for i, g := range c.Groups {
		if g.Name == oldName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("group not found: %s", oldName)
	}
	if oldName != newName {
		for _, g := range c.Groups {
			if g.Name == newName {
				return nil, fmt.Errorf("duplicate group name: %s", newName)
			}
		}
	}
	out := cloneConfig(c)
	out.Groups[idx].Name = newName
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateGroup returns a copy of c with the group named name replaced by
// replacement. The replacement is a complete-replacement Group (not a
// partial); callers build the full target state before invoking.
func UpdateGroup(c *Config, name string, replacement Group) (*Config, error) {
	idx := -1
	for i, g := range c.Groups {
		if g.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	out := cloneConfig(c)
	out.Groups[idx] = cloneGroup(replacement)
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// cloneConfig returns a deep copy of c. Slice fields are duplicated so
// callers can safely mutate the returned config without affecting the
// original.
func cloneConfig(c *Config) *Config {
	out := *c
	out.SymlinkDirectories = append([]string(nil), c.SymlinkDirectories...)
	out.LaunchCommand = append([]string(nil), c.LaunchCommand...)
	out.AddDirArgs = append([]string(nil), c.AddDirArgs...)
	if c.Groups != nil {
		out.Groups = make([]Group, len(c.Groups))
		for i, g := range c.Groups {
			out.Groups[i] = cloneGroup(g)
		}
	}
	return &out
}

// cloneGroup returns a deep copy of g.
func cloneGroup(g Group) Group {
	out := g
	out.Repos = append([]string(nil), g.Repos...)
	out.SymlinkDirectories = append([]string(nil), g.SymlinkDirectories...)
	out.LaunchCommand = append([]string(nil), g.LaunchCommand...)
	out.AddDirArgs = append([]string(nil), g.AddDirArgs...)
	return out
}

// AddReposToGroup returns a copy of c with the given repos appended (in
// order) to the named group's Repos, skipping any that are already present.
func AddReposToGroup(c *Config, groupName string, repos []string) (*Config, error) {
	idx := findGroup(c, groupName)
	if idx < 0 {
		return nil, fmt.Errorf("group not found: %s", groupName)
	}
	out := cloneConfig(c)
	existing := make(map[string]struct{}, len(out.Groups[idx].Repos))
	for _, r := range out.Groups[idx].Repos {
		existing[r] = struct{}{}
	}
	for _, r := range repos {
		if _, dup := existing[r]; dup {
			continue
		}
		existing[r] = struct{}{}
		out.Groups[idx].Repos = append(out.Groups[idx].Repos, r)
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveReposFromGroup returns a copy of c with the given repos removed
// from the named group's Repos. Fails if the resulting list would have
// fewer than 2 entries (the schema minimum).
func RemoveReposFromGroup(c *Config, groupName string, repos []string) (*Config, error) {
	idx := findGroup(c, groupName)
	if idx < 0 {
		return nil, fmt.Errorf("group not found: %s", groupName)
	}
	toRemove := make(map[string]struct{}, len(repos))
	for _, r := range repos {
		toRemove[r] = struct{}{}
	}
	remaining := make([]string, 0, len(c.Groups[idx].Repos))
	for _, r := range c.Groups[idx].Repos {
		if _, drop := toRemove[r]; drop {
			continue
		}
		remaining = append(remaining, r)
	}
	if len(remaining) < 2 {
		return nil, fmt.Errorf("group %q must have at least 2 repos after removal", groupName)
	}
	out := cloneConfig(c)
	out.Groups[idx].Repos = remaining
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// findGroup returns the index of the group with the given name, or -1.
func findGroup(c *Config, name string) int {
	for i, g := range c.Groups {
		if g.Name == name {
			return i
		}
	}
	return -1
}
