package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func runInherit(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

func runInheritEnv(env []string, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	c.Env = env
	return c.Run()
}

func newLogCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "log [N]",
		Aliases: []string{"l"},
		Short:   "Pretty graph log, default 20 commits",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			n := "20"
			if len(args) == 1 {
				n = args[0]
			}
			section("Last %s commits", n)
			return runInherit("git", "log", "--graph", "--decorate", "--abbrev-commit", "--date=short",
				"--pretty=format:%C(yellow)%h%C(reset) %C(cyan)%ad%C(reset) %C(bold green)%an%C(reset) %s%C(magenta)%d%C(reset)",
				"-n", n)
		},
	}
}

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "diff",
		Aliases: []string{"d"},
		Short:   "Working-tree diff vs HEAD with stat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			section("Working tree diff vs HEAD")
			if gitutil.DirtyCount() == 0 {
				ok("No changes.")
				return nil
			}
			_ = runInherit("git", "-c", "color.ui=always", "diff", "--stat", "HEAD")
			return runInherit("git", "-c", "color.ui=always", "diff", "HEAD")
		},
	}
}

func newWhoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "who [file]",
		Short: "Top contributors (or git blame a file)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			if len(args) == 0 {
				section("Top contributors")
				return runInherit("git", "shortlog", "-sn", "--no-merges", "HEAD")
			}
			if _, err := os.Stat(args[0]); err != nil {
				errf("File not found: %s", args[0])
				return nil
			}
			section("Blame: %s", args[0])
			return runInherit("git", "-c", "color.ui=always", "blame", "--date=short", "--", args[0])
		},
	}
}

func newRevertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revert",
		Short: "Pick a commit from the last 30, generate a revert commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			out, _ := gitutil.Run("log", "-30", "--pretty=format:%h %s")
			if out == "" {
				errf("No commits found.")
				return nil
			}
			chosen, err := selectOne("Revert which commit?", splitLines(out))
			if err != nil {
				info("Cancelled.")
				return nil
			}
			sha := chosen[:7]
			if !confirm(fmt.Sprintf("Revert %s? (y/N):", sha)) {
				info("Cancelled.")
				return nil
			}
			return runInherit("git", "revert", sha)
		},
	}
}

func newCherryPickCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "cherry-pick",
		Aliases: []string{"cp"},
		Short:   "Pick a source branch, then a commit; cherry-pick onto current branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			branches := listBranches()
			if len(branches) == 0 {
				warn("No branches found.")
				return nil
			}
			src, err := selectOne("Cherry-pick from which branch?", branches)
			if err != nil {
				info("Cancelled.")
				return nil
			}
			out, _ := gitutil.Run("log", src, "-30", "--pretty=format:%h %s")
			if out == "" {
				errf("No commits on %s.", src)
				return nil
			}
			chosen, err := selectOne("Cherry-pick which commit?", splitLines(out))
			if err != nil {
				info("Cancelled.")
				return nil
			}
			sha := chosen[:7]
			return runInherit("git", "cherry-pick", sha)
		},
	}
}

func newRebaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rebase",
		Short: "Interactive rebase wizard: last N commits / onto branch / onto upstream",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			mode, err := selectOne("Rebase how?", []string{"last N commits", "onto branch", "onto upstream"})
			if err != nil {
				info("Cancelled.")
				return nil
			}
			switch mode {
			case "last N commits":
				n := prompt("How many commits back?")
				if n == "" {
					n = "5"
				}
				return runInherit("git", "rebase", "-i", "HEAD~"+n)
			case "onto branch":
				branches := listBranches()
				target, err := selectOne("Rebase onto which branch?", branches)
				if err != nil {
					info("Cancelled.")
					return nil
				}
				return runInherit("git", "rebase", "-i", target)
			case "onto upstream":
				up := gitutil.UpstreamOf("")
				if up == "" {
					errf("No upstream configured.")
					return nil
				}
				return runInherit("git", "rebase", "-i", up)
			}
			return nil
		},
	}
}
