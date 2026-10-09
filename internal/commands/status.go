package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"whoami"},
		Short:   "Current git identity + gh-CLI auth",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Effective values for this shell: `git var` and `git config`
			// (no scope) already fold in session env and repo overrides.
			profile, scope := os.Getenv("GHSWITCH_ACTIVE_PROFILE"), "this shell"
			if profile == "" {
				profile, scope = config.GetActiveGlobal(), "global"
			}
			if gitutil.InRepo() && gitutil.ConfigGet("--local", "user.email") != "" {
				scope = "repo override"
			}
			section("%s (%s)", orNotSet(profile), scope)

			ident, _ := gitutil.Run("var", "GIT_AUTHOR_IDENT")
			if i := strings.LastIndex(ident, ">"); i >= 0 {
				ident = ident[:i+1]
			}
			ssh := os.Getenv("GIT_SSH_COMMAND")
			if ssh == "" {
				ssh = gitutil.ConfigGet("", "core.sshCommand")
			}
			signing := gitutil.ConfigGet("", "user.signingKey")
			if signing != "" && gitutil.ConfigGet("", "commit.gpgSign") != "true" {
				signing += " (commit.gpgSign off)"
			}
			field := func(label, v string) { fmt.Fprintf(os.Stderr, "  %-9s%s\n", label, orNotSet(v)) }
			field("Identity", ident)
			field("SSH", ssh)
			field("Signing", signing)
			if hasCmd("gh") {
				out, _ := exec.Command("gh", "auth", "status", "--active", "--json", "hosts",
					"--jq", `.hosts[][] | .login + " @ " + .host + (if .state == "success" then "" else " (" + .state + ")" end)`).Output()
				field("gh", strings.TrimSpace(string(out)))
			}
			return nil
		},
	}
}

func newUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "unset",
		Aliases: []string{"reset"},
		Short:   "Clear global git identity",
		RunE: func(cmd *cobra.Command, args []string) error {
			section("Clear global git identity")
			if !confirm("Remove global user.name, email, SSH command, GPG config? (y/N):") {
				info("Cancelled.")
				return nil
			}
			for _, key := range []string{"user.name", "user.email", "core.sshCommand", "user.signingKey", "commit.gpgSign"} {
				if gitutil.ConfigUnset("--global", key) == nil {
					ok("Cleared %s", key)
				}
			}
			active := config.GetActiveGlobal()
			host := "github.com"
			if active != "" {
				if p, found := config.Get(active); found && p.GHHost != "" {
					host = p.GHHost
				}
			}
			_ = gitutil.ConfigUnset("--global", fmt.Sprintf("credential.https://%s.helper", host))
			_ = config.SetActiveGlobal("")
			ok("Global identity cleared.")
			info("Session vars (if any) are unaffected — open a new shell or run: eval \"$(ghswitch shellinit --clear)\"")
			return nil
		},
	}
}
