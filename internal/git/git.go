// Package git is a thin wrapper around the system git CLI. Every exported
// function shells out to git via the internal run helper. err is reserved for
// spawn failures (e.g. git binary missing); a nonzero git exit appears as
// (zero-value, nil) for read functions and as a wrapped error for write
// functions. Tests require git on PATH (TestMain hard-fails otherwise).
package git
