package git

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("git"); err != nil {
		panic("git binary required for internal/git tests: " + err.Error())
	}
	os.Exit(m.Run())
}

func TestSmoke(t *testing.T) {
	// Sentinel: confirms TestMain ran (suite would panic before reaching
	// any test if git was missing).
	t.Log("TestMain gate passed")
}

func TestCheckRefFormat(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"plain branch name", "main", true},
		{"slash in branch name", "feat/login", true},
		{"empty string", "", false},
		{"leading dash", "-bad", false},
		{"contains space", "bad name", false},
		{"double dot", "feat..bad", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CheckRefFormat(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
