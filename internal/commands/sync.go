package commands

import (
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull --rebase then push (auto -u on first push)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			cur := gitutil.CurrentBranch()
			section("Sync %s", cur)
			if gitutil.UpstreamOf("") == "" {
				info("No upstream — pushing with -u to origin/%s", cur)
				if _, err := gitutil.Run("push", "-u", "origin", cur); err != nil {
					errf("Push failed: %v", err)
					return nil
				}
				ok("Pushed.")
				return nil
			}
			info("Pulling (rebase)...")
			if _, err := gitutil.Run("pull", "--rebase"); err != nil {
				errf("Pull failed (conflicts?). Resolve, then run sync again.")
				return nil
			}
			info("Pushing...")
			if _, err := gitutil.Run("push"); err != nil {
				errf("Push failed: %v", err)
				return nil
			}
			ok("Synced.")
			return nil
		},
	}
}

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull --rebase from upstream",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			cur := gitutil.CurrentBranch()
			section("Pull %s", cur)
			if gitutil.UpstreamOf("") == "" {
				warn("No upstream set.")
				info("Set with: git branch --set-upstream-to=origin/%s", cur)
				return nil
			}
			_, err := gitutil.Run("pull", "--rebase")
			return err
		},
	}
}

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push current branch (auto -u on first push)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			cur := gitutil.CurrentBranch()
			section("Push %s", cur)
			if gitutil.UpstreamOf("") == "" {
				info("First push — using -u origin/%s", cur)
				_, err := gitutil.Run("push", "-u", "origin", cur)
				return err
			}
			_, err := gitutil.Run("push")
			return err
		},
	}
}

func newFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "git fetch --all --prune",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			section("Fetch all remotes (--prune)")
			if _, err := gitutil.Run("fetch", "--all", "--prune"); err != nil {
				return err
			}
			ok("Fetch complete.")
			return nil
		},
	}
}

func newRepoInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "repo",
		Aliases: []string{"info"},
		Short:   "Dashboard: branch, ahead/behind, dirty count, profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			section("Repo dashboard")
			info("Path     : %s", gitutil.RepoRoot())
			cur := gitutil.CurrentBranch()
			info("Branch   : %s", cur)
			up := gitutil.UpstreamOf("")
			if up != "" {
				info("Upstream : %s", up)
				if ahead, behind, ok2 := gitutil.AheadBehind(); ok2 {
					info("Sync     : %d ahead · %d behind", ahead, behind)
				}
			} else {
				info("Upstream : (none)")
			}
			remote := gitutil.ConfigGet("", "remote.origin.url")
			info("Remote   : %s", orNotSet(remote))
			dirty := gitutil.DirtyCount()
			if dirty == 0 {
				info("Status   : clean")
			} else {
				info("Status   : %d changed file(s)", dirty)
			}
			return nil
		},
	}
}
