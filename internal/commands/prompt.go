package commands

import (
	"fmt"
	"os"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// activeProfileName resolves the effective profile the same way status.go
// and doctor.go display it: this shell's session switch first, then a
// repo-local email match (mirrors auto.go's detection), then the global
// fallback. Returns "" if nothing is active — callers should render that as
// no prompt segment, not an error.
func activeProfileName() string {
	if p := os.Getenv("GHSWITCH_ACTIVE_PROFILE"); p != "" {
		return p
	}
	if gitutil.InRepo() {
		if email := gitutil.ConfigGet("--local", "user.email"); email != "" {
			profiles, _ := config.Load()
			for _, p := range profiles {
				if p.Email == email {
					return p.Name
				}
			}
		}
	}
	return config.GetActiveGlobal()
}

// newPromptCmd prints just the plain profile name to stdout, no color and
// no trailing newline — meant for `$(ghswitch prompt)` inside PS1/PROMPT.
// Color is deliberately left to the caller's own prompt syntax (e.g. zsh's
// %F{cyan}...%f) rather than emitted as raw ANSI here, since raw escapes
// embedded in PS1/PROMPT without shell-specific zero-width markers throw
// off cursor-position math on line wraps.
func newPromptCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "prompt",
		Short:         "Print the active profile name, for embedding in a shell prompt",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(os.Stdout, activeProfileName())
			return nil
		},
	}
}
