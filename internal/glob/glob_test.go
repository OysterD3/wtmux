package glob

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasGlobChars(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"node_modules", false},
		{".env", false},
		{"", false},
		{".env*", true},
		{"foo?", true},
		{"foo[abc]", true},
		{"foo]", true},
		{"foo{a,b}", true},
		{"foo}", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, hasGlobChars(tc.in))
		})
	}
}

func TestMatchSegs(t *testing.T) {
	cases := []struct {
		name string
		p    []string
		f    []string
		want bool
	}{
		{"both empty match", nil, nil, true},
		{"empty pattern non-empty file fails", nil, []string{"a"}, false},
		{"non-empty pattern empty file fails", []string{"a"}, nil, false},
		{"literal match", []string{"foo"}, []string{"foo"}, true},
		{"literal mismatch", []string{"foo"}, []string{"bar"}, false},
		{"star within segment", []string{"*.json"}, []string{"a.json"}, true},
		{"star within segment no match", []string{"*.json"}, []string{"a.txt"}, false},
		{"question mark", []string{"a?c"}, []string{"abc"}, true},
		{"char class", []string{"[abc]"}, []string{"a"}, true},
		{"char class no match", []string{"[abc]"}, []string{"d"}, false},
		{"** at start matches anything", []string{"**", "x"}, []string{"a", "b", "x"}, true},
		{"** zero segments", []string{"a", "**", "b"}, []string{"a", "b"}, true},
		{"** one segment", []string{"a", "**", "b"}, []string{"a", "x", "b"}, true},
		{"** multiple segments", []string{"a", "**", "b"}, []string{"a", "x", "y", "b"}, true},
		{"** at end matches anything", []string{"a", "**"}, []string{"a", "b", "c"}, true},
		{"** at end matches zero", []string{"a", "**"}, []string{"a"}, true},
		{"different lengths no **", []string{"a", "b"}, []string{"a"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := matchSegs(tc.p, tc.f)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMatchSegs_BadPattern(t *testing.T) {
	_, err := matchSegs([]string{"[abc"}, []string{"a"})
	assert.Error(t, err)
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"foo", "foo", true},
		{"a/b", "a/b", true},
		{"a/*/c", "a/x/c", true},
		{"a/**/c", "a/x/y/c", true},
		{"a/**/c", "a/c", true},
		{"*.json", "a.json", true},
		{"*.json", "a.txt", false},
		{"a/b", "a/b/c", false},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+" vs "+tc.path, func(t *testing.T) {
			got, err := matchPattern(tc.pattern, tc.path)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMatchPattern_BadPattern(t *testing.T) {
	_, err := matchPattern("[abc", "abc")
	assert.Error(t, err)
}

func TestWalkAndMatch(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "config", "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "config", "a.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "config", "sub", "b.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "ignored.txt"), []byte("x"), 0o644))

	matches, err := walkAndMatch(repo, "config/**/*.json")
	require.NoError(t, err)
	sort.Strings(matches)
	assert.Equal(t, []string{"config/a.json", "config/sub/b.json"}, matches)
}

func TestWalkAndMatch_NoHits(t *testing.T) {
	repo := t.TempDir()
	matches, err := walkAndMatch(repo, "*.json")
	require.NoError(t, err)
	assert.Empty(t, matches)
}

func TestWalkAndMatch_BadPattern(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "a"), []byte("x"), 0o644))
	_, err := walkAndMatch(repo, "[abc")
	assert.Error(t, err)
}

func TestExpandSymlinkItems_LiteralPassThrough(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "node_modules"), 0o755))
	got, err := ExpandSymlinkItems(repo, []string{"node_modules"})
	require.NoError(t, err)
	assert.Equal(t, []string{"node_modules"}, got)
}

func TestExpandSymlinkItems_GlobIncludesDotfiles(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("A=1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env.local"), []byte("B=2"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env.development"), []byte("C=3"), 0o644))
	got, err := ExpandSymlinkItems(repo, []string{".env*"})
	require.NoError(t, err)
	sort.Strings(got)
	assert.Equal(t, []string{".env", ".env.development", ".env.local"}, got)
}

func TestExpandSymlinkItems_EmptyMatchReturnsNonNil(t *testing.T) {
	repo := t.TempDir()
	got, err := ExpandSymlinkItems(repo, []string{".env*"})
	require.NoError(t, err)
	assert.Equal(t, []string{}, got)
	assert.NotNil(t, got)
}

func TestExpandSymlinkItems_DedupLiteralAndGlob(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("x"), 0o644))
	got, err := ExpandSymlinkItems(repo, []string{".env", ".env*"})
	require.NoError(t, err)
	assert.Equal(t, []string{".env"}, got)
}

func TestExpandSymlinkItems_MixedLiteralAndGlob(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env.local"), []byte("y"), 0o644))
	got, err := ExpandSymlinkItems(repo, []string{"node_modules", ".env*"})
	require.NoError(t, err)
	sort.Strings(got)
	assert.Equal(t, []string{".env", ".env.local", "node_modules"}, got)
}

func TestExpandSymlinkItems_DoubleStarRecursion(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "config", "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "config", "a.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "config", "sub", "b.json"), []byte("{}"), 0o644))
	got, err := ExpandSymlinkItems(repo, []string{"config/**/*.json"})
	require.NoError(t, err)
	sort.Strings(got)
	assert.Equal(t, []string{"config/a.json", "config/sub/b.json"}, got)
}

func TestExpandSymlinkItems_PreservesFirstSeenOrder(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "node_modules"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".env"), []byte("x"), 0o644))
	got, err := ExpandSymlinkItems(repo, []string{"node_modules", ".env"})
	require.NoError(t, err)
	assert.Equal(t, []string{"node_modules", ".env"}, got)
}
