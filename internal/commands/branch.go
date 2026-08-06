package commands

import (
	"fmt"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func listBranches() []string {
	out, _ := gitutil.Run("for-each-ref", "--format=%(refname:short)", "refs/heads")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func newBranchCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "branch",
		Aliases: []string{"br"},
		Short:   "Arrow-key switcher over all branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			branches := listBranches()
			if len(branches) == 0 {
				warn("No branches found.")
				return nil
			}
			cur := gitutil.CurrentBranch()
			section("Switch branch")
			info("Current: %s", cur)
			chosen, err := selectOneMarked("Pick a branch", branches, cur, "current")
			if err != nil {
				info("Cancelled.")
				return nil
			}
			if chosen == cur {
				info("Already on %q.", chosen)
				return nil
			}
			if _, err := gitutil.Run("checkout", chosen); err != nil {
				errf("Checkout failed: %v", err)
				return nil
			}
			ok("Switched to %s", chosen)
			return nil
		},
	}
}

func newNewBranchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [name]",
		Short: "Create + checkout a new branch from current",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				name = prompt("New branch name:")
			}
			if name == "" {
				errf("Name cannot be empty.")
				return nil
			}
			section("Create branch %s", name)
			if _, err := gitutil.Run("checkout", "-b", name); err != nil {
				errf("Branch creation failed: %v", err)
				return nil
			}
			short, _ := gitutil.Run("rev-parse", "--short", "HEAD")
			ok("On %s (branched from %s)", name, short)
			return nil
		},
	}
}

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rename [name]",
		Aliases: []string{"mv"},
		Short:   "Rename current branch (warns if pushed)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			cur := gitutil.CurrentBranch()
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				name = prompt(fmt.Sprintf("Rename %q to:", cur))
			}
			if name == "" {
				errf("Name cannot be empty.")
				return nil
			}
			section("Rename branch")
			if _, err := gitutil.Run("branch", "-m", name); err != nil {
				errf("Rename failed: %v", err)
				return nil
			}
			ok("Renamed %s → %s", cur, name)
			if gitutil.OK("rev-parse", "origin/"+cur) {
				info("Was pushed — to update remote run:")
				info("  git push origin -u %s && git push origin --delete %s", name, cur)
			}
			return nil
		},
	}
}

func newCleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Delete local branches merged into the default branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			def := gitutil.DefaultBranch()
			section("Delete merged branches")
			info("Comparing against: %s", def)

			out, _ := gitutil.Run("branch", "--merged", def)
			var merged []string
			for _, line := range strings.Split(out, "\n") {
				b := strings.TrimSpace(strings.TrimPrefix(line, "*"))
				if b == "" || b == def {
					continue
				}
				merged = append(merged, b)
			}
			if len(merged) == 0 {
				ok("No merged branches to clean.")
				return nil
			}
			info("Will delete: %s", strings.Join(merged, ", "))
			if !confirm("Delete these branches? (y/N):") {
				info("Cancelled.")
				return nil
			}
			for _, b := range merged {
				if _, err := gitutil.Run("branch", "-d", b); err == nil {
					ok("Deleted %s", b)
				} else {
					warn("Skipped %s", b)
				}
			}
			return nil
		},
	}
}

func newFreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "fresh",
		Aliases: []string{"nuke"},
		Short:   "fetch --prune + drop gone tracking branches + gc --auto",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			section("Fresh")
			if _, err := gitutil.Run("fetch", "--prune"); err != nil {
				errf("fetch failed: %v", err)
				return nil
			}
			ok("Fetched + pruned.")

			out, _ := gitutil.Run("for-each-ref", "--format=%(refname:short) %(upstream:track)", "refs/heads")
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "[gone]") {
					b := strings.Fields(line)[0]
					if _, err := gitutil.Run("branch", "-D", b); err == nil {
						ok("Dropped gone branch: %s", b)
					}
				}
			}
			if _, err := gitutil.Run("gc", "--auto"); err == nil {
				ok("gc --auto complete.")
			}
			return nil
		},
	}
}

func newRecentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "recent",
		Short: "Switcher over the 10 most recently active branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			out, _ := gitutil.Run("for-each-ref", "--sort=-committerdate", "--format=%(refname:short)", "--count=10", "refs/heads")
			branches := strings.Split(out, "\n")
			if len(branches) == 0 || branches[0] == "" {
				warn("No branches found.")
				return nil
			}
			cur := gitutil.CurrentBranch()
			chosen, err := selectOneMarked("Recent branches", branches, cur, "current")
			if err != nil {
				info("Cancelled.")
				return nil
			}
			if chosen == cur {
				info("Already on %q.", chosen)
				return nil
			}
			if _, err := gitutil.Run("checkout", chosen); err != nil {
				errf("Checkout failed: %v", err)
				return nil
			}
			ok("Switched to %s", chosen)
			return nil
		},
	}
}
