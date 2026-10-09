package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/MadhuS-1605/ghswitch/internal/session"
	"github.com/spf13/cobra"
)

// chpwdShellHook is appended by shellinit: on every directory change (and
// once at shell start) it evals `__chpwd-check`, which auto-switches this
// shell when a .ghswitch.yml or dir= rule applies, and otherwise only warns
// on an identity mismatch. bash has no chpwd, so PROMPT_COMMAND is guarded
// on $PWD actually changing — not re-run after every command.
const chpwdShellHook = `
_ghswitch_chpwd() {
  [ "$PWD" = "$_GHSWITCH_LASTPWD" ] && return
  _GHSWITCH_LASTPWD=$PWD
  eval "$(command ghswitch __chpwd-check)"
}
if [ -n "$ZSH_VERSION" ]; then
  chpwd_functions+=(_ghswitch_chpwd)
  _ghswitch_chpwd
elif [ -n "$BASH_VERSION" ]; then
  case "$PROMPT_COMMAND" in
    *_ghswitch_chpwd*) ;;
    *) PROMPT_COMMAND="_ghswitch_chpwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
  esac
fi
`

// autoTarget picks the profile a directory asks for: the repo's
// .ghswitch.yml preset first (the repo declares its identity), else the
// deepest dir= rule. rule identifies which of those matched, so the hook
// can tell "entered a new area" from "moved within the same one".
func autoTarget(profiles []config.Profile, cwd, repoRoot, preset string) (p config.Profile, rule string, ok bool) {
	if preset != "" {
		for _, p := range profiles {
			if p.Name == preset {
				return p, "repo:" + repoRoot, true
			}
		}
	}
	if p, dir, ok := config.MatchDir(profiles, cwd); ok {
		return p, "dir:" + dir, true
	}
	return config.Profile{}, "", false
}

// __chpwd-check is hidden plumbing invoked by the hook above; its stdout is
// eval'd, so it prints only export/unset lines there.
//
// A rule applies when you *enter* its area (GHSWITCH_RULE changes), so a
// manual `ghswitch personal` inside ~/work sticks until you leave ~/work.
func newHiddenChpwdCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "__chpwd-check",
		Hidden:        true,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, _ := config.Load()
			cwd, _ := os.Getwd()
			root, preset := "", ""
			if gitutil.InRepo() {
				root = gitutil.RepoRoot()
				preset = readPreset(root)
			}
			target, rule, matched := autoTarget(profiles, cwd, root, preset)
			if preset != "" && !strings.HasPrefix(rule, "repo:") {
				warn(".ghswitch.yml asks for profile %q, which isn't configured here — run: ghswitch add", preset)
			}

			prevRule := os.Getenv("GHSWITCH_RULE")
			switch {
			case matched && rule != prevRule:
				fmt.Printf("export GHSWITCH_RULE=%s\n", shellQuote(rule))
				if target.Name != os.Getenv("GHSWITCH_ACTIVE_PROFILE") {
					session.WriteExports(os.Stdout, sessionEnv(target, target.Login()))
					config.AppendAudit(target.Name)
					ok("%s  %s", color(colorBold, target.Name), color(colorCyan, "auto · "+tildePath(strings.SplitN(rule, ":", 2)[1])))
				}
				return nil
			case !matched && prevRule != "":
				fmt.Println("unset GHSWITCH_RULE")
			}

			if root != "" {
				mismatchWarning()
			}
			return nil
		},
	}
}

// mismatchWarning flags a repo whose own user.email differs from the
// active profile's — same check doctor.go performs, fired on cd.
func mismatchWarning() {
	active := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
	if active == "" {
		active = config.GetActiveGlobal()
	}
	p, found := config.Get(active)
	if !found {
		return
	}
	if owner := ownerProfile(active); owner != "" {
		warn("this repo belongs to %s, but this shell is %q (%s)", gitutil.RemoteOwner(), active, p.Login())
		info("  fix: ghswitch %s   ·   make it automatic: echo 'profile: %s' > .ghswitch.yml  (or dir= in the profile)", owner, owner)
		return
	}
	if p.Email == "" {
		return
	}
	curEmail := gitutil.ConfigGet("--local", "user.email")
	if curEmail == "" || curEmail == p.Email {
		return
	}
	warn("this repo is pinned to %q, but this shell is %q (%s)", curEmail, active, p.Email)
	info("  fix: ghswitch local   (or add a .ghswitch.yml / dir= rule so it switches itself)")
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func tildePath(p string) string {
	if home, _ := os.UserHomeDir(); home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// ownerProfile returns the profile whose GitHub login owns this repo's
// origin, when that isn't the active one — the "personal repo, work
// account" case an email check can't see. Org-owned repos match nothing.
func ownerProfile(active string) string {
	owner := gitutil.RemoteOwner()
	if owner == "" {
		return ""
	}
	profiles, _ := config.Load()
	return ownerProfileIn(profiles, owner, active)
}

func ownerProfileIn(profiles []config.Profile, owner, active string) string {
	for _, p := range profiles {
		if strings.EqualFold(p.Login(), owner) {
			if p.Name == active {
				return ""
			}
			return p.Name
		}
	}
	return ""
}
