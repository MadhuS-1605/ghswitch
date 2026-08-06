// Package commands wires up the ghswitch cobra command tree. Subcommands are
// added incrementally per the phased rewrite plan (identity core, git
// toolkit, GitHub integration, team features).
package commands

import "github.com/spf13/cobra"

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:   "ghswitch",
		Short: "GitHub-aware git toolkit — switch identities, manage branches, commit, sync, and open PRs",
		Args:  cobra.ArbitraryArgs,
		// Errors here are almost always "profile not found" / git command
		// failures, not usage mistakes — dumping the full 60-command list
		// on every error is noise, not help. SilenceErrors avoids a double
		// print: main.go already prints the returned error itself.
		SilenceUsage:  true,
		SilenceErrors: true,
		// Bare `ghswitch` or `ghswitch <profile>` (no matching subcommand)
		// falls through here — mirrors the old bash script's `ghswitch work`
		// shorthand.
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return switchByName(args[0])
			}
			return switchInteractive()
		},
	}

	root.AddCommand(
		newSwitchCmd(),
		newAddCmd(),
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
