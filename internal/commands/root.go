// Package commands wires up the ghswitch cobra command tree. Subcommands are
// added incrementally per the phased rewrite plan (identity core, git
// toolkit, GitHub integration, team features).
package commands

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

func Root(version string) *cobra.Command {
	var ttl time.Duration
	root := &cobra.Command{
		Use:     "ghswitch",
		Short:   "Never commit as the wrong GitHub account — per-terminal identity that switches itself by folder",
		Version: version,
		Args:    cobra.ArbitraryArgs,
		// Errors here are almost always "profile not found" / git command
		// failures, not usage mistakes — dumping the full 60-command list
		// on every error is noise, not help. SilenceErrors avoids a double
		// print: main.go already prints the returned error itself.
		SilenceUsage:  true,
		SilenceErrors: true,
		// A shell can hold a --ttl'd session past its expiry (only the
		// shell itself can revert its own env — see applyGlobal). Warn on
		// every invocation rather than silently reverting underfoot.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			warnIfExpired()
		},
		// Bare `ghswitch` or `ghswitch <profile>` (no matching subcommand)
		// falls through here — mirrors the old bash script's `ghswitch work`
		// shorthand.
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return switchByNameTTL(args[0], ttl)
			}
			if ttl > 0 {
				return errNeedsProfileForTTL
			}
			return switchInteractive()
		},
	}
	root.Flags().DurationVar(&ttl, "ttl", 0, "warn once this long has passed without a re-switch (e.g. 4h)")

	// Help leads with identity — the reason ghswitch exists — and files the
	// git/GitHub conveniences under their own headings.
	groups := []struct {
		id, title string
		cmds      []*cobra.Command
	}{
		{"identity", "Identity:", []*cobra.Command{
			newInitCmd(), newSwitchCmd(), newBackCmd(), newStatusCmd(), newListCmd(),
			newAddCmd(), newAddHostCmd(), newLoginCmd(), newRotateCmd(), newEditCmd(), newDeleteCmd(),
			newLocalCmd(), newAutoCmd(), newUnsetCmd(), newDoctorCmd(), newReauthorCmd(),
			newHooksInstallCmd(), newAuditCmd(), newShellinitCmd(), newPromptCmd(),
		}},
		{"git", "Git toolkit:", []*cobra.Command{
			newBranchCmd(), newNewBranchCmd(), newRenameCmd(), newCleanCmd(), newFreshCmd(), newRecentCmd(),
			newCommitCmd(), newAmendCmd(), newUndoCmd(), newConventionalCommitCmd(), newWipCmd(), newWipPopCmd(),
			newFixupCmd(), newSyncCmd(), newPullCmd(), newPushCmd(), newFetchCmd(), newRepoInfoCmd(), newLogCmd(),
			newDiffCmd(), newWhoCmd(), newRevertCmd(), newCherryPickCmd(), newRebaseCmd(), newStashCmd(),
			newTagCmd(), newWorktreeCmd(), newConflictCmd(),
		}},
		{"github", "GitHub:", []*cobra.Command{
			newPRCmd(), newPRsCmd(), newPRCheckoutCmd(), newPRStatusCmd(), newReviewCmd(),
			newIssuesCmd(), newInboxCmd(), newOpenCmd(), newReleaseCmd(),
		}},
		{"team", "Team & multi-repo:", []*cobra.Command{
			newBootstrapCmd(), newExportCmd(), newImportCmd(), newReposCmd(), newSyncAllCmd(), newStatsCmd(),
		}},
	}
	cobra.EnableCommandSorting = false
	for _, g := range groups {
		root.AddGroup(&cobra.Group{ID: g.id, Title: g.title})
		for _, c := range g.cmds {
			c.GroupID = g.id
			root.AddCommand(c)
		}
	}
	root.AddCommand(newGendocsCmd(), newHiddenIsProfileCmd(), newHiddenChpwdCheckCmd(), newHiddenCredentialCmd())
	return root
}

var errNeedsProfileForTTL = &ttlError{}

type ttlError struct{}

func (*ttlError) Error() string { return "--ttl requires a profile name" }

// warnIfExpired checks GHSWITCH_EXPIRES (set by `--ttl`) against the clock.
// It only warns — the running shell's env can't be changed from here, so
// this is a nudge toward `ghswitch back` / a fresh switch, not an auto-revert.
func warnIfExpired() {
	raw := os.Getenv("GHSWITCH_EXPIRES")
	if raw == "" {
		return
	}
	expires, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return
	}
	if left := time.Until(time.Unix(expires, 0)); left < 0 {
		warn("session profile expired %s ago — run: ghswitch <profile> (or ghswitch back)", -left.Round(time.Minute))
	}
}
