package commands

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func requireRepo() bool {
	if gitutil.InRepo() {
		return true
	}
	errf("Not inside a git repository.")
	return false
}

func newCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "commit [message]",
		Aliases: []string{"c"},
		Short:   "git add -A + commit (prompts if no message)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			if gitutil.DirtyCount() == 0 {
				warn("Working tree clean — nothing to commit.")
				return nil
			}
			msg := ""
			if len(args) == 1 {
				msg = args[0]
			} else {
				msg = prompt("Commit message:")
			}
			if msg == "" {
				errf("Message cannot be empty.")
				return nil
			}
			section("Quick commit")
			if _, err := gitutil.Run("add", "-A"); err != nil {
				errf("git add failed: %v", err)
				return nil
			}
			if _, err := gitutil.Run("commit", "-m", msg); err != nil {
				errf("Commit failed: %v", err)
				return nil
			}
			short, _ := gitutil.Run("rev-parse", "--short", "HEAD")
			ok("Committed: %s", short)
			return nil
		},
	}
}

func newAmendCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "amend",
		Short: "Add staged + amend last commit, message unchanged",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			section("Amend last commit")
			_, _ = gitutil.Run("add", "-A")
			if _, err := gitutil.Run("commit", "--amend", "--no-edit"); err != nil {
				errf("Amend failed: %v", err)
				return nil
			}
			short, _ := gitutil.Run("rev-parse", "--short", "HEAD")
			ok("Amended (message unchanged): %s", short)
			if gitutil.UpstreamOf("") != "" {
				warn("Already pushed — use: git push --force-with-lease")
			}
			return nil
		},
	}
}

func newUndoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "undo",
		Short: "Soft reset HEAD~1, keeping changes staged",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			last, err := gitutil.Run("log", "-1", "--pretty=format:%h %s")
			if err != nil || last == "" {
				errf("No commits to undo.")
				return nil
			}
			section("Undo last commit (keep changes)")
			info("Last commit: %s", last)
			if !confirm("Soft reset HEAD~1? Files stay staged. (y/N):") {
				info("Cancelled.")
				return nil
			}
			if _, err := gitutil.Run("reset", "--soft", "HEAD~1"); err != nil {
				errf("Reset failed: %v", err)
				return nil
			}
			ok("Undone. Changes preserved in staging.")
			return nil
		},
	}
}

var conventionalTypes = []string{"feat", "fix", "chore", "docs", "refactor", "test", "style", "perf", "build", "ci", "revert"}

func newConventionalCommitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cc",
		Short: "Conventional commit wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			if gitutil.DirtyCount() == 0 {
				warn("Working tree clean — nothing to commit.")
				return nil
			}
			section("Conventional commit")
			typ, err := selectOne("Pick a type", conventionalTypes)
			if err != nil {
				info("Cancelled.")
				return nil
			}
			scope := prompt("Scope (optional, e.g. auth, api):")
			desc := prompt("Subject (imperative, no period):")
			breaking := confirm("Breaking change? (y/N):")
			if desc == "" {
				errf("Subject required.")
				return nil
			}
			bang := ""
			if breaking {
				bang = "!"
			}
			var msg string
			if scope != "" {
				msg = fmt.Sprintf("%s(%s)%s: %s", typ, scope, bang, desc)
			} else {
				msg = fmt.Sprintf("%s%s: %s", typ, bang, desc)
			}
			info("Message: %s", msg)
			if !confirm("Stage all + commit? (Y/n):") {
				info("Cancelled.")
				return nil
			}
			_, _ = gitutil.Run("add", "-A")
			if _, err := gitutil.Run("commit", "-m", msg); err != nil {
				errf("Commit failed: %v", err)
				return nil
			}
			short, _ := gitutil.Run("rev-parse", "--short", "HEAD")
			ok("Committed: %s", short)
			return nil
		},
	}
}

func newWipCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wip",
		Short: "One-shot stash with auto-message",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			branch := gitutil.CurrentBranch()
			n := gitutil.DirtyCount()
			msg := fmt.Sprintf("WIP @ %s · %d file(s) · %s", branch, n, time.Now().Format("15:04"))
			if _, err := gitutil.Run("stash", "push", "-u", "-m", msg); err != nil {
				errf("Stash failed: %v", err)
				return nil
			}
			ok("Stashed: %s", msg)
			return nil
		},
	}
}

func newWipPopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wip-pop",
		Short: "Pop the most recent stash",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			if _, err := gitutil.Run("stash", "pop"); err != nil {
				errf("Pop failed: %v", err)
				return nil
			}
			ok("Stash popped.")
			return nil
		},
	}
}

func newFixupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fixup",
		Short: "Pick a past commit, create a fixup, auto-squash via interactive rebase",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			if gitutil.DirtyCount() == 0 {
				warn("Working tree clean — nothing to fixup.")
				return nil
			}
			out, _ := gitutil.Run("log", "-20", "--pretty=format:%h %s")
			if out == "" {
				errf("No commits found.")
				return nil
			}
			lines := splitLines(out)
			chosen, err := selectOne("Fixup which commit?", lines)
			if err != nil {
				info("Cancelled.")
				return nil
			}
			sha := chosen[:7]
			_, _ = gitutil.Run("add", "-A")
			if _, err := gitutil.Run("commit", "--fixup", sha); err != nil {
				errf("fixup commit failed: %v", err)
				return nil
			}
			ok("Created fixup! commit for %s", sha)
			c := exec.Command("git", "-c", "sequence.editor=:", "rebase", "-i", "--autosquash", sha+"~1")
			if err := c.Run(); err != nil {
				errf("Auto-squash rebase failed — resolve manually: git rebase --continue")
				return nil
			}
			ok("Auto-squashed.")
			return nil
		},
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
