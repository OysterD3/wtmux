// Package git is a thin wrapper around the system git CLI. Every exported
// function shells out to git via the internal run helper. err is reserved for
// spawn failures (e.g. git binary missing); a nonzero git exit appears as
// (zero-value, nil) for read functions and as a wrapped error for write
// functions. Tests require git on PATH (TestMain hard-fails otherwise).
package git

// CheckRefFormat reports whether name is a valid git branch name, per
// `git check-ref-format --branch`. No working directory is required.
func CheckRefFormat(name string) (bool, error) {
	_, _, code, err := run("", "check-ref-format", "--branch", name)
	if err != nil {
		return false, err
	}
	return code == 0, nil
}
