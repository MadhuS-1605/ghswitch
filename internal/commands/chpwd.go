package commands

import (
	"os"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// chpwdShellHook is appended by shellinit so every `cd` gets a fast,
// silent-unless-mismatched identity check — an ambient version of the
// opt-in pre-commit guard (hooks-install), catching the wrong-account case
// before you start working, not just before you commit.
const chpwdShellHook = `
if [ -n "$ZSH_VERSION" ]; then
  _ghswitch_chpwd() { command ghswitch __chpwd-check; }
  chpwd_functions+=(_ghswitch_chpwd)
elif [ -n "$BASH_VERSION" ]; then
  _ghswitch_chpwd() { command ghswitch __chpwd-check; }
  case "$PROMPT_COMMAND" in
    *_ghswitch_chpwd*) ;;
    *) PROMPT_COMMAND="_ghswitch_chpwd;${PROMPT_COMMAND}" ;;
  esac
fi
`

// __chpwd-check is hidden plumbing invoked by the chpwd hook above. It
// prints nothing and exits 0 in the common case (no active profile, no
// repo, or identity already matches) — same mismatch check doctor.go
// performs, just fired on `cd` instead of on demand.
func newHiddenChpwdCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "__chpwd-check",
		Hidden:        true,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				return nil
			}
			active := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
			if active == "" {
				active = config.GetActiveGlobal()
			}
			if active == "" {
				return nil
			}
			p, found := config.Get(active)
			if !found || p.Email == "" {
				return nil
			}
			curEmail := gitutil.ConfigGet("", "user.email")
			if curEmail == "" || curEmail == p.Email {
				return nil
			}
			warn("this repo expects a different identity: user.email=%q, active profile %q expects %q", curEmail, active, p.Email)
			info("  fix: ghswitch local   (or 'ghswitch auto' to detect)")
			return nil
		},
	}
}
