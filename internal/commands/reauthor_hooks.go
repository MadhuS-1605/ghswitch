package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// newReauthorCmd fixes commits already made under the wrong identity — the
// command gitswitch.dev has and ghswitch.sh never did. It rewrites author
// (and committer) info on a commit range to match the currently active
// identity via `git rebase --exec ... --reset-author`, the standard git
// mechanism for this (no custom history surgery needed).
func newReauthorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reauthor",
		Short: "Rewrite author/committer identity on past commits to match the active profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}

			active := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
			if active == "" {
				active = config.GetActiveGlobal()
			}
			if active == "" {
				errf("No active profile — run 'ghswitch <profile>' first so reauthor knows which identity to apply.")
				return nil
			}
			p, found := config.Get(active)
			if !found {
				errf("Active profile %q not found in profiles.conf.", active)
				return nil
			}
			section("Reauthor commits as %s <%s>", p.GitName, p.Email)

			mode, err := selectOne("Rewrite which commits?", []string{"last N commits", "since branch point (upstream)", "since default branch"})
			if err != nil {
				info("Cancelled.")
				return nil
			}

			var base string
			switch mode {
			case "last N commits":
				n := prompt("How many commits back?")
				if n == "" {
					n = "1"
				}
				base = "HEAD~" + n
			case "since branch point (upstream)":
				up := gitutil.UpstreamOf("")
				if up == "" {
					errf("No upstream configured.")
					return nil
				}
				base = up
			case "since default branch":
				base = gitutil.DefaultBranch()
			}

			out, _ := gitutil.Run("log", "--oneline", base+"..HEAD")
			if out == "" {
				warn("No commits in range %s..HEAD.", base)
				return nil
			}
			info("Commits to reauthor:\n%s", out)
			if !confirm(fmt.Sprintf("Rewrite these as %s <%s>? This rewrites history — force-push will be needed if already pushed. (y/N):", p.GitName, p.Email)) {
				info("Cancelled.")
				return nil
			}

			env := append(os.Environ(),
				"GIT_AUTHOR_NAME="+p.GitName, "GIT_AUTHOR_EMAIL="+p.Email,
				"GIT_COMMITTER_NAME="+p.GitName, "GIT_COMMITTER_EMAIL="+p.Email,
			)
			if err := runInheritEnv(env, "git", "rebase", "-r", base, "--exec", "git commit --amend --no-edit --reset-author"); err != nil {
				errf("Rebase failed — resolve manually, then: git rebase --continue")
				return nil
			}
			ok("Reauthored. Force-push with --force-with-lease if this branch was already pushed.")
			return nil
		},
	}
}

func newHooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hooks-install",
		Short: "Install a pre-commit guard that blocks commits with the wrong identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			hooksDir, _ := gitutil.Run("rev-parse", "--git-path", "hooks")
			hookPath := hooksDir + "/pre-commit"
			section("Install pre-commit guard")
			info("Hook path: %s", hookPath)

			if b, err := os.ReadFile(hookPath); err == nil && !strings.Contains(string(b), "ghswitch-guard") {
				warn("Existing pre-commit hook found (not from ghswitch)")
				if !confirm("Overwrite? (y/N):") {
					info("Cancelled.")
					return nil
				}
			}

			if err := os.WriteFile(hookPath, []byte(preCommitGuard), 0o755); err != nil {
				errf("%v", err)
				return nil
			}
			ok("Installed pre-commit guard")
			info("Checks $GIT_AUTHOR_EMAIL / $GHSWITCH_ACTIVE_PROFILE first (session-scoped), falls back to global profiles.conf.")
			info("Bypass with --no-verify if you really need to (not recommended)")
			return nil
		},
	}
}

// preCommitGuard checks the SESSION identity first (GIT_AUTHOR_EMAIL /
// GHSWITCH_ACTIVE_PROFILE, both set by `eval "$(ghswitch <profile>)"` in
// this shell) before falling back to the legacy global active-profile file
// — the guard now follows per-terminal identity, not one shared global one.
const preCommitGuard = `#!/usr/bin/env bash
# ghswitch-guard: blocks commit if the effective identity mismatches the
# active profile's declared email.
CONFIG="${HOME}/.config/ghswitch/profiles.conf"
[ -f "$CONFIG" ] || exit 0

ACTIVE="${GHSWITCH_ACTIVE_PROFILE:-}"
if [ -z "$ACTIVE" ]; then
  ACTIVE_FILE="${HOME}/.config/ghswitch/active"
  [ -f "$ACTIVE_FILE" ] && ACTIVE=$(tr -d '[:space:]' < "$ACTIVE_FILE")
fi
[ -z "$ACTIVE" ] && exit 0

EXPECTED=$(awk -v p="[$ACTIVE]" '$0==p{f=1;next} f && /^\[/{exit} f && /^email=/{sub(/^email=/,"");print;exit}' "$CONFIG")
[ -z "$EXPECTED" ] && exit 0

CURRENT="${GIT_AUTHOR_EMAIL:-$(git config user.email 2>/dev/null)}"
if [ -n "$EXPECTED" ] && [ "$EXPECTED" != "$CURRENT" ]; then
  printf '\033[31m✗\033[0m ghswitch-guard: effective email is "%s" but active profile "%s" expects "%s"\n' "$CURRENT" "$ACTIVE" "$EXPECTED" >&2
  printf '   fix: eval "$(ghswitch %s)"   (or ghswitch local / ghswitch auto)\n' "$ACTIVE" >&2
  exit 1
fi
exit 0
`
