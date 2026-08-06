package commands

import (
	"fmt"
	"os"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/spf13/cobra"
)

// switchByName resolves a profile and applies it globally (session-scoped).
// Shared by the bare `ghswitch <profile>` form (root.go) and `ghswitch switch`.
func switchByName(name string) error {
	p, found := config.Get(name)
	if !found {
		return fmt.Errorf("profile %q not found — run: ghswitch list", name)
	}
	applyGlobal(p)
	return nil
}

// switchInteractive shows a picker over all profiles, then applies the choice.
func switchInteractive() error {
	profiles, err := config.Load()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		warn("No profiles yet. Run: ghswitch add")
		return nil
	}
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name
	}
	active := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
	if active == "" {
		active = config.GetActiveGlobal()
	}
	chosen, err := selectOneMarked("Switch global profile", names, active, "active")
	if err != nil {
		info("Cancelled.")
		return nil
	}
	return switchByName(chosen)
}

func newSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "switch [profile]",
		Aliases: []string{"use"},
		Short:   "Switch the active profile (interactive if no name given)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return switchByName(args[0])
			}
			return switchInteractive()
		},
	}
}
