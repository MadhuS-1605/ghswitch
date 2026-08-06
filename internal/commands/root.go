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
		Short:   "GitHub-aware git toolkit — switch identities, manage branches, commit, sync, and open PRs",
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

	root.AddCommand(
		newSwitchCmd(),
		newBackCmd(),
		newAuditCmd(),
		newGendocsCmd(),
		newAddCmd(),
		newAddHostCmd(),
		newListCmd(),
		newDeleteCmd(),
		newLocalCmd(),
		newStatusCmd(),
		newUnsetCmd(),
		newEditCmd(),
		newAutoCmd(),
		newDoctorCmd(),
		newShellinitCmd(),
		newHiddenIsProfileCmd(),
		// Git toolkit (Phase 4)
		newBranchCmd(),
		newNewBranchCmd(),
		newRenameCmd(),
		newCleanCmd(),
		newFreshCmd(),
		newRecentCmd(),
		newCommitCmd(),
		newAmendCmd(),
		newUndoCmd(),
		newConventionalCommitCmd(),
		newWipCmd(),
		newWipPopCmd(),
		newFixupCmd(),
		newSyncCmd(),
		newPullCmd(),
		newPushCmd(),
		newFetchCmd(),
		newRepoInfoCmd(),
		newLogCmd(),
		newDiffCmd(),
		newWhoCmd(),
		newRevertCmd(),
		newCherryPickCmd(),
		newRebaseCmd(),
		newStashCmd(),
		newTagCmd(),
		newWorktreeCmd(),
		newConflictCmd(),
		// GitHub integration (Phase 5)
		newPRCmd(),
		newPRsCmd(),
		newPRCheckoutCmd(),
		newPRStatusCmd(),
		newReviewCmd(),
		newIssuesCmd(),
		newInboxCmd(),
		newOpenCmd(),
		newLoginCmd(),
		newRotateCmd(),
		newReleaseCmd(),
		// New + updated for session isolation (Phase 6)
		newReauthorCmd(),
		newHooksInstallCmd(),
		// Team features (Phase 7)
		newBootstrapCmd(),
		newExportCmd(),
		newImportCmd(),
		newReposCmd(),
		newSyncAllCmd(),
		newStatsCmd(),
	)
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
