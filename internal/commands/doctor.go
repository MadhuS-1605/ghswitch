package commands

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// tokenExpiryDays reads the same "github-authentication-token-expiration"
// response header the bash doctor() scraped via `gh api -i /user`
// (ghswitch.sh:994-1024) — present for fine-grained PATs, absent for
// classic PATs/OAuth (in which case hasExp is false).
func tokenExpiryDays(host string) (days int, hasExp bool) {
	c := exec.Command("gh", "api", "-i", "/user")
	c.Env = append(os.Environ(), "GH_HOST="+host)
	out, err := c.Output()
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(out), "\n") {
		low := strings.ToLower(line)
		if !strings.HasPrefix(low, "github-authentication-token-expiration:") {
			continue
		}
		val := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		val = strings.TrimSuffix(val, "\r")
		for _, layout := range []string{"2006-01-02 15:04:05 MST", "2006-01-02 15:04:05 -0700", time.RFC1123} {
			if t, err := time.Parse(layout, val); err == nil {
				return int(time.Until(t).Hours() / 24), true
			}
		}
		return 0, false
	}
	return 0, false
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Audit ssh-agent, key perms, gh auth, GPG keyring, identity-vs-profile mismatch",
		RunE: func(cmd *cobra.Command, args []string) error {
			issues := 0
			section("ghswitch doctor")

			if hasCmd("git") {
				out, _ := exec.Command("git", "--version").Output()
				ok("git installed: %s", strings.TrimSpace(string(out)))
			} else {
				errf("git missing")
				issues++
			}
			if hasCmd("gh") {
				ok("gh CLI installed")
			} else {
				warn("gh CLI missing — pr / prs commands will not work")
			}
			if hasCmd("gpg") {
				ok("gpg installed")
			} else {
				warn("gpg missing — signed commits disabled")
			}
			if hasCmd("ssh-agent") && os.Getenv("SSH_AUTH_SOCK") != "" {
				ok("ssh-agent active (socket: %s)", os.Getenv("SSH_AUTH_SOCK"))
			} else {
				warn("ssh-agent not running or SSH_AUTH_SOCK not set")
			}

			if os.Getenv("GHSWITCH_SHELLINIT") != "" {
				ok("shell integration loaded (per-terminal identity, auto-switch on cd)")
			} else {
				errf("shell integration not loaded — switches change global config only")
				info("  fix: add to %s:  %s", rcFile(), shellinitLine)
				issues++
			}

			issues += checkPlaintextTokens()

			active := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
			scope := "session"
			if active == "" {
				active = config.GetActiveGlobal()
				scope = "global fallback"
			}

			if active == "" {
				section("Active profile")
				info("(none) — run 'ghswitch' to pick one")
			} else {
				section("Active profile: %s (%s)", active, scope)
				p, found := config.Get(active)
				if found {
					if p.SSHKey != "" {
						expanded := config.ExpandSSHKey(p.SSHKey)
						if fi, err := os.Stat(expanded); err == nil {
							ok("SSH key exists: %s", expanded)
							if fi.Mode().Perm() != 0o600 {
								errf("SSH key permissions: %o (should be 600)", fi.Mode().Perm())
								info("  fix: chmod 600 %s", expanded)
								issues++
							} else {
								ok("SSH key permissions: 600")
							}
						} else {
							errf("SSH key file missing: %s", expanded)
							issues++
						}
					}

					host := p.GHHost
					if host == "" {
						host = "github.com"
					}
					if hasCmd("gh") {
						if exec.Command("gh", "auth", "status", "--hostname", host).Run() == nil {
							ok("gh authenticated for %s", host)
							if days, hasExp := tokenExpiryDays(host); hasExp {
								switch {
								case days < 0:
									errf("gh token EXPIRED")
									info("  fix: ghswitch rotate %s", active)
									issues++
								case days < 7:
									errf("gh token expires in %d day(s)", days)
									info("  fix: ghswitch rotate %s", active)
									issues++
								case days < 30:
									warn("gh token expires in %d days", days)
								default:
									ok("gh token valid · %d days remaining", days)
								}
							}
						} else {
							errf("gh NOT authenticated for %s", host)
							info("  fix: gh auth login --hostname %s", host)
							issues++
						}
					}

					if p.GPGKey != "" {
						if hasCmd("gpg") && exec.Command("gpg", "--list-secret-keys", p.GPGKey).Run() == nil {
							ok("GPG key in keyring: %s", p.GPGKey)
						} else {
							errf("GPG key NOT in keyring: %s", p.GPGKey)
							issues++
						}
					}
				}
			}

			if gitutil.InRepo() {
				section("Current repo")
				remote := gitutil.ConfigGet("", "remote.origin.url")
				curEmail := gitutil.ConfigGet("", "user.email")
				info("Remote     : %s", orNotSet(remote))
				info("user.email : %s", orNotSet(curEmail))
				if active != "" {
					if p, found := config.Get(active); found && p.Email != "" && curEmail != p.Email {
						warn("user.email mismatch: repo=%q vs profile=%q", curEmail, p.Email)
						info("  fix: ghswitch local   (or 'ghswitch auto' to detect)")
						issues++
					}
				}
			}

			if issues == 0 {
				ok("All checks passed")
			} else {
				warn("%d issue(s) found", issues)
			}
			return nil
		},
	}
}

// checkPlaintextTokens offers to move tokens still written in
// profiles.conf into the OS keychain (Save does the move). Returns the
// number of issues left over.
func checkPlaintextTokens() int {
	profiles, err := config.Load()
	if err != nil {
		return 0
	}
	var plain []string
	for _, p := range profiles {
		if p.GHToken != "" {
			plain = append(plain, p.Name)
		}
		for _, h := range p.Hosts {
			if h.GHToken != "" {
				plain = append(plain, p.Name+"@"+h.Host)
			}
		}
	}
	if len(plain) == 0 {
		ok("tokens: none stored in plain text")
		return 0
	}
	warn("tokens stored in plain text in profiles.conf: %s", strings.Join(plain, ", "))
	if !confirm("Move them into the OS keychain now? (Y/n):") {
		return 1
	}
	if err := config.Save(profiles); err != nil {
		errf("%v", err)
		return 1
	}
	if after, _ := config.Load(); slices.ContainsFunc(after, func(p config.Profile) bool { return p.GHToken != "" }) {
		errf("no usable keychain on this system — tokens left in profiles.conf (mode 0600)")
		return 1
	}
	ok("tokens moved to the OS keychain")
	return 0
}
