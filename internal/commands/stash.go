package commands

import (
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func newStashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stash",
		Short: "Menu: save / list / pop / apply / drop / clear",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			action, err := selectOne("Stash", []string{"save", "list", "pop", "apply", "drop", "clear"})
			if err != nil {
				info("Cancelled.")
				return nil
			}
			switch action {
			case "save":
				msg := prompt("Stash message (Enter for none):")
				var e error
				if msg != "" {
					_, e = gitutil.Run("stash", "push", "-u", "-m", msg)
				} else {
					_, e = gitutil.Run("stash", "push", "-u")
				}
				if e == nil {
					ok("Stashed.")
				}
			case "list":
				out, _ := gitutil.Run("stash", "list")
				if out == "" {
					info("No stashes.")
				} else {
					info("%s", out)
				}
			case "pop", "apply", "drop":
				out, _ := gitutil.Run("stash", "list", "--pretty=format:%gd %s")
				if out == "" {
					warn("No stashes.")
					return nil
				}
				chosen, err := selectOne("Pick a stash", strings.Split(out, "\n"))
				if err != nil {
					info("Cancelled.")
					return nil
				}
				ref := strings.Fields(chosen)[0]
				if _, err := gitutil.Run("stash", action, ref); err == nil {
					ok("Done.")
				}
			case "clear":
				if confirm("Drop ALL stashes? (y/N):") {
					if _, err := gitutil.Run("stash", "clear"); err == nil {
						ok("All stashes cleared.")
					}
				} else {
					info("Cancelled.")
				}
			}
			return nil
		},
	}
}

func newTagCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tag",
		Short: "Menu: list / create / delete / push-all",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			action, err := selectOne("Tags", []string{"list", "create", "delete", "push-all"})
			if err != nil {
				info("Cancelled.")
				return nil
			}
			switch action {
			case "list":
				out, _ := gitutil.Run("tag", "-l", "--sort=-creatordate")
				if out == "" {
					info("No tags.")
				} else {
					info("%s", out)
				}
			case "create":
				name := prompt("Tag name (e.g. v1.2.3):")
				if name == "" {
					errf("Name required.")
					return nil
				}
				msg := prompt("Message (blank = lightweight tag):")
				var e error
				if msg != "" {
					_, e = gitutil.Run("tag", "-a", name, "-m", msg)
				} else {
					_, e = gitutil.Run("tag", name)
				}
				if e == nil {
					ok("Created tag %s", name)
				}
			case "delete":
				out, _ := gitutil.Run("tag", "-l")
				if out == "" {
					warn("No tags.")
					return nil
				}
				chosen, err := selectOne("Delete which tag?", strings.Split(out, "\n"))
				if err != nil {
					info("Cancelled.")
					return nil
				}
				_, _ = gitutil.Run("tag", "-d", chosen)
				if confirm("Also delete on remote? (y/N):") {
					_, _ = gitutil.Run("push", "origin", ":refs/tags/"+chosen)
				}
				ok("Deleted tag %s", chosen)
			case "push-all":
				if _, err := gitutil.Run("push", "--tags"); err == nil {
					ok("Pushed all tags.")
				}
			}
			return nil
		},
	}
}

func newWorktreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "worktree",
		Aliases: []string{"wt"},
		Short:   "Menu: list / add / remove",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			action, err := selectOne("Worktrees", []string{"list", "add", "remove"})
			if err != nil {
				info("Cancelled.")
				return nil
			}
			switch action {
			case "list":
				out, _ := gitutil.Run("worktree", "list")
				info("%s", out)
			case "add":
				path := prompt("New worktree path:")
				if path == "" {
					errf("Path required.")
					return nil
				}
				branch := prompt("Branch (blank = new branch from current):")
				var e error
				if branch != "" {
					_, e = gitutil.Run("worktree", "add", path, branch)
				} else {
					_, e = gitutil.Run("worktree", "add", "-b", "wt-"+path, path)
				}
				if e == nil {
					ok("Added worktree at %s", path)
				} else {
					errf("%v", e)
				}
			case "remove":
				out, _ := gitutil.Run("worktree", "list", "--porcelain")
				var paths []string
				for _, line := range strings.Split(out, "\n") {
					if p, ok := strings.CutPrefix(line, "worktree "); ok {
						paths = append(paths, p)
					}
				}
				if len(paths) <= 1 {
					warn("No removable worktrees.")
					return nil
				}
				chosen, err := selectOne("Remove which worktree?", paths[1:])
				if err != nil {
					info("Cancelled.")
					return nil
				}
				if _, err := gitutil.Run("worktree", "remove", chosen); err == nil {
					ok("Removed %s", chosen)
				} else {
					errf("%v", err)
				}
			}
			return nil
		},
	}
}

func newConflictCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "conflict",
		Aliases: []string{"merge"},
		Short:   "Walk through unmerged files in $EDITOR, mark resolved",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			out, _ := gitutil.Run("diff", "--name-only", "--diff-filter=U")
			if out == "" {
				ok("No merge conflicts.")
				return nil
			}
			files := strings.Split(out, "\n")
			section("Merge conflicts")
			for _, f := range files {
				info("✗ %s", f)
			}
			ed := envOr("EDITOR", envOr("VISUAL", "vi"))
			for _, f := range files {
				if !confirm("Edit " + f + "? (Y/n/skip all):") {
					continue
				}
				_ = runInherit(ed, f)
				if confirm("Mark " + f + " resolved (git add)? (y/N):") {
					_, _ = gitutil.Run("add", f)
				}
			}
			remaining, _ := gitutil.Run("diff", "--name-only", "--diff-filter=U")
			if remaining == "" {
				ok("All conflicts resolved. Continue your merge/rebase.")
			} else {
				warn("Still unresolved: %s", remaining)
			}
			return nil
		},
	}
}
