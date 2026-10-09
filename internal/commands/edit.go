package commands

import (
	"errors"
	"os"
	"os/exec"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open profiles.conf in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = config.EnsureDir()
			if _, err := os.Stat(config.ConfigFile()); os.IsNotExist(err) {
				_ = os.WriteFile(config.ConfigFile(), nil, 0o600)
			}
			ed := os.Getenv("EDITOR")
			if ed == "" {
				ed = os.Getenv("VISUAL")
			}
			if ed == "" {
				ed = "vi"
			}
			info("Opening %s in %s", config.ConfigFile(), ed)
			c := exec.Command(ed, config.ConfigFile())
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
			return c.Run()
		},
	}
}

// shellinitScript is the shell function installed by `eval "$(ghswitch
// shellinit)"`. It wraps the binary so switching a profile evals the export
// lines ghswitch prints to stdout into the CURRENT shell — the mechanism
// that gives every terminal its own isolated identity. Everything else
// passes straight through, untouched, to avoid evaling arbitrary command
// output as shell code.
const shellinitScript = `export GHSWITCH_SHELLINIT=1
ghswitch() {
  if [ $# -eq 0 ]; then
    eval "$(command ghswitch switch)"
    return
  fi
  case "$1" in
    switch|use)
      eval "$(command ghswitch "$@")"
      ;;
    *)
      if command ghswitch __is-profile "$1" >/dev/null 2>&1; then
        eval "$(command ghswitch switch "$1")"
      else
        command ghswitch "$@"
      fi
      ;;
  esac
}
` + chpwdShellHook

func newShellinitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "shellinit",
		Short: "Emit a shell function enabling per-terminal session isolation",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := os.Stdout.WriteString(shellinitScript)
			return err
		},
	}
}

var errNotAProfile = errors.New("not a profile")

// __is-profile is hidden plumbing the shellinit wrapper calls to decide
// whether `ghswitch <word>` should be eval'd (a profile switch) or passed
// through untouched (any other subcommand). It prints nothing; only its
// exit code matters.
func newHiddenIsProfileCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "__is-profile <name>",
		Hidden:        true,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.Exists(args[0]) {
				return nil
			}
			return errNotAProfile
		},
	}
}
