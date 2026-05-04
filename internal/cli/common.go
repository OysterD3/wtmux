package cli

import (
	"errors"
	"os"

	"github.com/OysterD3/wtmux/internal/config"
	wtmuxerrors "github.com/OysterD3/wtmux/internal/errors"
	wtlog "github.com/OysterD3/wtmux/internal/log"
)

// persistentFlags holds values bound to the root command's PersistentFlags.
// Set by Cobra; read by command RunE callbacks.
type persistentFlags struct {
	configPath string
	group      string
	verbose    bool
}

// pf is the shared instance used by every subcommand. Cobra writes to it
// during flag parsing, so commands read flags via pf.* in their RunE.
var pf persistentFlags

// loadConfigOrDefault loads config via discovery, falling back to
// config.Default() when no config exists at any source. Mirrors TS:
// `loaded?.config ?? DEFAULT_CONFIG`. ErrNotFound is the only "use
// defaults" path; every other error (explicit-but-missing, JSON syntax,
// validation) is returned as a wtmux user error.
func loadConfigOrDefault(cwd string) (*config.Config, error) {
	loaded, err := config.Load(config.LoadInputs{
		DiscoveryInputs: config.DiscoveryInputs{
			Explicit: pf.configPath,
			Cwd:      cwd,
		},
	})
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return config.Default(), nil
		}
		return nil, wtmuxerrors.New(wtmuxerrors.KindUser, "%s", err.Error())
	}
	return loaded.Config, nil
}

// applyVerbose copies pf.verbose into the log package.
func applyVerbose() {
	wtlog.SetVerbose(pf.verbose)
}

// mustGetwd is a tiny helper so commands don't bury the same boilerplate.
// A working directory failure is a precondition error.
func mustGetwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", wtmuxerrors.New(wtmuxerrors.KindPrecondition, "cannot determine cwd: %s", err.Error())
	}
	return cwd, nil
}
