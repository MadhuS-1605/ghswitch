package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/spf13/cobra"
)

// switchByName resolves a profile and applies it globally (session-scoped).
// Shared by the bare `ghswitch <profile>` form (root.go) and `ghswitch switch`.
func switchByName(name string) error {
	return switchByNameTTL(name, 0)
}

func switchByNameTTL(name string, ttl time.Duration) error {
	p, found := config.Get(name)
	if !found {
		return fmt.Errorf("profile %q not found — run: ghswitch list", name)
	}
	applyGlobal(p, ttl)
	return nil
}

// switchBack jumps to whichever profile was active immediately before the
// most recent switch (tracked in config.PreviousFile).
func switchBack() error {
	prev := config.GetPreviousGlobal()
	if prev == "" {
		return fmt.Errorf("no previous profile recorded yet")
	}
	return switchByName(prev)
}

// switchInteractive shows a picker over all profiles, then applies the choice.
func switchInteractive() error {
	profiles, err := config.Load()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		warn("No profiles yet. Run: ghswitch init")
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
	var ttl time.Duration
	cmd := &cobra.Command{
		Use:     "switch [profile]",
		Aliases: []string{"use"},
		Short:   "Switch the active profile (interactive if no name given)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return switchByNameTTL(args[0], ttl)
			}
			if ttl > 0 {
				return fmt.Errorf("--ttl requires a profile name")
			}
			return switchInteractive()
		},
	}
	cmd.Flags().DurationVar(&ttl, "ttl", 0, "warn once this long has passed without a re-switch (e.g. 4h)")
	return cmd
}

func newBackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "back",
		Short: "Switch back to the profile that was active before the last switch",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return switchBack()
		},
	}
}
