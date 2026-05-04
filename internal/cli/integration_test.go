package cli_test

// Integration tests against the built wtmux binary. TestMain builds it once
// into a temp dir; each test runs it via os/exec inside its own t.TempDir
// git repo (or non-repo dir) and asserts on stdout / stderr / exit code.

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "wtmux-bin-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdir tmp: %v\n", err)
		os.Exit(2)
	}
	defer os.RemoveAll(tmp)

	binPath = filepath.Join(tmp, "wtmux")
	cmd := exec.Command("go", "build", "-o", binPath, "github.com/OysterD3/wtmux/cmd/wtmux")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build wtmux: %v\n", err)
		os.Exit(2)
	}

	os.Exit(m.Run())
}

type runResult struct {
	stdout string
	stderr string
	code   int
}

func runWtmux(t *testing.T, dir string, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	res := runResult{stdout: outBuf.String(), stderr: errBuf.String()}
	if err == nil {
		res.code = 0
		return res
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.code = exitErr.ExitCode()
		return res
	}
	t.Fatalf("spawn failed: %v", err)
	return res
}

// gitInitCommit creates a git repo at dir with one empty initial commit on
// "main", so commands that touch HEAD have something to work with.
func gitInitCommit(t *testing.T, dir string) {
	t.Helper()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "--allow-empty", "-m", "init", "-q")
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s in %s: %v: %s", strings.Join(args, " "), dir, err, errBuf.String())
	}
}

func TestVersion(t *testing.T) {
	res := runWtmux(t, t.TempDir(), "--version")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	if strings.TrimSpace(res.stdout) == "" {
		t.Fatalf("expected version on stdout, got %q", res.stdout)
	}
}

func TestRootHelp(t *testing.T) {
	res := runWtmux(t, t.TempDir(), "--help")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	for _, want := range []string{
		"coordinated git worktrees across sibling repos",
		"wtmux new <name>",
		"wtmux rm <name>",
		"wtmux ls",
		"wtmux config",
	} {
		if !strings.Contains(res.stdout, want) {
			t.Fatalf("help missing %q in:\n%s", want, res.stdout)
		}
	}
}

func TestConfigHelp(t *testing.T) {
	res := runWtmux(t, t.TempDir(), "config", "--help")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "wtmux config — Interactively edit wtmux config") {
		t.Fatalf("config help wrong:\n%s", res.stdout)
	}
}

func TestLsOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	res := runWtmux(t, dir, "ls")
	if res.code != 2 {
		t.Fatalf("exit=%d (want 2) stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "[wtmux] ") {
		t.Fatalf("expected [wtmux] prefix on stderr: %q", res.stderr)
	}
	if !strings.Contains(res.stderr, "not inside any git repository") {
		t.Fatalf("unexpected message: %q", res.stderr)
	}
}

func TestLsEmptyRepo(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	res := runWtmux(t, dir, "ls")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	if strings.TrimSpace(res.stdout) != "(no coordinated worktrees)" {
		t.Fatalf("unexpected stdout: %q", res.stdout)
	}
}

func TestLsBadGroupFlag(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	res := runWtmux(t, dir, "ls", "-g", "nope")
	if res.code != 1 {
		t.Fatalf("exit=%d (want 1 user error) stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "does not match") {
		t.Fatalf("unexpected stderr: %q", res.stderr)
	}
}

func TestRmMissingNameArg(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	res := runWtmux(t, dir, "rm")
	if res.code == 0 {
		t.Fatalf("expected nonzero exit, got 0; stderr=%q", res.stderr)
	}
	if !strings.Contains(res.stderr, "[wtmux]") {
		t.Fatalf("expected [wtmux] prefix on stderr: %q", res.stderr)
	}
}

func TestRmDryRunAbsent(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	// no worktree exists yet; rm --dry-run should be a no-op success
	res := runWtmux(t, dir, "rm", "ghost", "--dry-run")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	if res.stdout != "" {
		t.Fatalf("expected empty stdout for absent worktree, got %q", res.stdout)
	}
}

func TestRmDryRunPresent(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	// Create a worktree at .worktrees/wt-name (matching default pattern).
	mustGit(t, dir, "worktree", "add", "-b", "wt-name", ".worktrees/wt-name", "HEAD")
	res := runWtmux(t, dir, "rm", "wt-name", "--dry-run")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "dry-run: would remove") {
		t.Fatalf("expected dry-run line in stderr, got %q", res.stderr)
	}
}

func TestRmRefusesOnDirty(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	wt := filepath.Join(dir, ".worktrees", "wt-dirty")
	mustGit(t, dir, "worktree", "add", "-b", "wt-dirty", wt, "HEAD")

	// Dirty the worktree so the precheck has something to refuse on.
	if err := os.WriteFile(filepath.Join(wt, "newfile.txt"), []byte("scratch"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	res := runWtmux(t, dir, "rm", "wt-dirty")
	if res.code != 2 {
		t.Fatalf("exit=%d (want 2 precondition) stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "refusing to remove") {
		t.Fatalf("expected refusal in stderr, got %q", res.stderr)
	}
	if !strings.Contains(res.stderr, "dirty:") {
		t.Fatalf("expected dirty marker in stderr, got %q", res.stderr)
	}
	// Worktree must still exist — refusal is all-or-nothing.
	if !fileExists(t, wt) {
		t.Fatalf("worktree was removed despite refusal")
	}
}

func TestCreateMissingName(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	res := runWtmux(t, dir, "new")
	if res.code == 0 {
		t.Fatalf("expected nonzero exit, got 0; stderr=%q", res.stderr)
	}
	if !strings.Contains(res.stderr, "requires at least 1 arg") {
		t.Fatalf("unexpected stderr: %q", res.stderr)
	}
}

func TestCreateOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	res := runWtmux(t, dir, "new", "foo", "--no-launch")
	if res.code != 2 {
		t.Fatalf("exit=%d (want 2 precondition) stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "not inside any git repository") {
		t.Fatalf("unexpected stderr: %q", res.stderr)
	}
}

func TestCreateDryRun(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	res := runWtmux(t, dir, "new", "foo", "--dry-run")
	if res.code != 0 {
		t.Fatalf("exit=%d stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stderr, "dry-run: would create") {
		t.Fatalf("expected dry-run line, got %q", res.stderr)
	}
}

func TestCreateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	gitInitCommit(t, dir)
	res := runWtmux(t, dir, "new", "wt-foo", "--no-launch")
	if res.code != 0 {
		t.Fatalf("create exit=%d stderr=%q", res.code, res.stderr)
	}
	if !fileExists(t, filepath.Join(dir, ".worktrees", "wt-foo")) {
		t.Fatalf("worktree dir not created")
	}

	res = runWtmux(t, dir, "ls")
	if res.code != 0 {
		t.Fatalf("ls exit=%d stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "wt-foo") {
		t.Fatalf("ls didn't list wt-foo: %q", res.stdout)
	}

	res = runWtmux(t, dir, "rm", "wt-foo", "--force")
	if res.code != 0 {
		t.Fatalf("rm exit=%d stderr=%q", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "removed ") {
		t.Fatalf("rm stdout missing removed: %q", res.stdout)
	}
}

func fileExists(t *testing.T, p string) bool {
	t.Helper()
	_, err := os.Stat(p)
	return err == nil
}
