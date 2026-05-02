package git

import (
	"os"
	"os/exec"
	"testing"
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
