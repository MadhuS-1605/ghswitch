package commands

import (
	"fmt"
	"os"
	"os/exec"

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
			sessionActive := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
			if sessionActive != "" {
				section("Session identity (this shell only)")
				fmt.Fprintf(os.Stderr, "  Profile      : %s\n", sessionActive)
				fmt.Fprintf(os.Stderr, "  Name         : %s\n", os.Getenv("GIT_AUTHOR_NAME"))
				fmt.Fprintf(os.Stderr, "  Email        : %s\n", os.Getenv("GIT_AUTHOR_EMAIL"))
			}

			section("Global git identity")
			name := gitutil.ConfigGet("--global", "user.name")
			email := gitutil.ConfigGet("--global", "user.email")
			signKey := gitutil.ConfigGet("--global", "user.signingKey")
			sshCmd := gitutil.ConfigGet("--global", "core.sshCommand")
			printField := func(label, v string) {
				if v == "" {
					v = "(not set)"
				}
				fmt.Fprintf(os.Stderr, "  %-9s: %s\n", label, v)
			}
			printField("Name", name)
			printField("Email", email)
			printField("GPG Key", signKey)
			printField("SSH Cmd", sshCmd)
			if active := config.GetActiveGlobal(); active != "" {
				fmt.Fprintf(os.Stderr, "  ghswitch : %s (global fallback active)\n", active)
			}

			if gitutil.InRepo() {
				lname := gitutil.ConfigGet("--local", "user.name")
				lemail := gitutil.ConfigGet("--local", "user.email")
				lssh := gitutil.ConfigGet("--local", "core.sshCommand")
				if lname != "" || lemail != "" || lssh != "" {
					section("Local git identity (repo override)")
					if lname != "" {
						fmt.Fprintf(os.Stderr, "  Name     : %s\n", lname)
					}
					if lemail != "" {
						fmt.Fprintf(os.Stderr, "  Email    : %s\n", lemail)
					}
					if lssh != "" {
						fmt.Fprintf(os.Stderr, "  SSH Cmd  : %s\n", lssh)
					}
				}
			}

			if hasCmd("gh") {
				section("gh CLI status")
				out, _ := exec.Command("gh", "auth", "status").CombinedOutput()
				fmt.Fprintln(os.Stderr, string(out))
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
