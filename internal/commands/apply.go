package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/MadhuS-1605/ghswitch/internal/session"
)

func hasCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// applyGlobal is a "session switch": it prints export lines to stdout for
// `eval "$(ghswitch <profile>)"` to pick up (the real, per-shell switch),
// and best-effort mirrors the same values into `git config --global` as a
// fallback for tools that only read gitconfig, not env vars.
//
// ttl, if non-zero, is exported as GHSWITCH_EXPIRES — root.go warns (but
// does not silently revert) once that time passes, since only the shell
// that ran shellinit can actually change its own env.
func applyGlobal(p config.Profile, ttl time.Duration) {
	if host := gitutil.RemoteHost(); host != "" {
		p = p.ForHost(host)
	}
	section("Applying profile: %s (session + global fallback)", p.Name)

	if p.GitName != "" {
		_ = gitutil.ConfigSet("--global", "user.name", p.GitName)
		ok("user.name  = %s", p.GitName)
	}
	if p.Email != "" {
		_ = gitutil.ConfigSet("--global", "user.email", p.Email)
		ok("user.email = %s", p.Email)
	}
	applySSHAndGPG(p, "--global")
	applyGH(p)

	if previous := config.GetActiveGlobal(); previous != "" && previous != p.Name {
		_ = config.SetPreviousGlobal(previous)
	}
	_ = config.SetActiveGlobal(p.Name)
	config.AppendAudit(p.Name)

	env := session.EnvFor(p)
	if ttl > 0 {
		expires := time.Now().Add(ttl).Unix()
		env["GHSWITCH_EXPIRES"] = strconv.FormatInt(expires, 10)
		ok("expires    = %s (%s)", time.Now().Add(ttl).Format("15:04"), ttl)
	}
	session.WriteExports(os.Stdout, env)
	ok("Active profile → %s", p.Name)
	info("Not session-isolated? Run: eval \"$(ghswitch shellinit)\" in your rc file, then re-run this switch.")
}

// applyLocal pins a profile to the current repo only, via `git config
// --local` — already repo-scoped, no session env needed.
func applyLocal(p config.Profile) error {
	if !gitutil.InRepo() {
		return fmt.Errorf("not inside a git repository")
	}
	if host := gitutil.RemoteHost(); host != "" {
		p = p.ForHost(host)
	}
	section("Applying profile: %s (local, this repo only)", p.Name)

	if p.GitName != "" {
		_ = gitutil.ConfigSet("--local", "user.name", p.GitName)
		ok("user.name  = %s", p.GitName)
	}
	if p.Email != "" {
		_ = gitutil.ConfigSet("--local", "user.email", p.Email)
		ok("user.email = %s", p.Email)
	}
	applySSHAndGPG(p, "--local")
	applyGH(p)
	registerRepo()
	return nil
}

func applySSHAndGPG(p config.Profile, scope string) {
	if p.SSHKey != "" {
		expanded := config.ExpandSSHKey(p.SSHKey)
		if _, err := os.Stat(expanded); err == nil {
			_ = gitutil.ConfigSet(scope, "core.sshCommand", fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", expanded))
			ok("SSH key    = %s", expanded)
			if hasCmd("ssh-add") && os.Getenv("SSH_AUTH_SOCK") != "" {
				if err := exec.Command("ssh-add", expanded).Run(); err == nil {
					ok("SSH key added to agent")
				}
			}
		} else {
			warn("SSH key file not found: %s", expanded)
		}
	}

	if p.GPGKey != "" {
		if hasCmd("gpg") && exec.Command("gpg", "--list-secret-keys", p.GPGKey).Run() == nil {
			_ = gitutil.ConfigSet(scope, "user.signingKey", p.GPGKey)
			_ = gitutil.ConfigSet(scope, "commit.gpgSign", "true")
			ok("GPG key    = %s", p.GPGKey)
		} else {
			warn("GPG key not in keyring: %s", p.GPGKey)
		}
	}
}

func applyGH(p config.Profile) {
	host := p.GHHost
	if host == "" {
		host = "github.com"
	}
	if p.GHToken != "" {
		helper := fmt.Sprintf(`!f() { echo "username=git"; echo "password=%s"; }; f`, p.GHToken)
		_ = gitutil.ConfigSet("--global", fmt.Sprintf("credential.https://%s.helper", host), helper)
		ok("git token  = configured via credential.helper")
		if hasCmd("gh") {
			cmd := exec.Command("gh", "auth", "login", "--with-token", "--hostname", host)
			cmd.Stdin = strings.NewReader(p.GHToken)
			if cmd.Run() == nil {
				ok("gh token   = authenticated for %s", host)
				if p.GitName != "" {
					if exec.Command("gh", "auth", "switch", "--hostname", host, "--user", p.GitName).Run() == nil {
						ok("gh active  = %s", p.GitName)
					} else {
						warn("gh active = %q not switched (need gh ≥ 2.40 + matching user)", p.GitName)
					}
				}
			} else {
				warn("gh token failed to authenticate gh CLI")
			}
		} else {
			info("gh CLI missing, token applied only to git HTTPS")
		}
		return
	}
	if hasCmd("gh") {
		if p.GitName != "" && exec.Command("gh", "auth", "switch", "--hostname", host, "--user", p.GitName).Run() == nil {
			ok("gh CLI     = switched to account %s", p.GitName)
		} else if exec.Command("gh", "auth", "status", "--hostname", host).Run() == nil {
			ok("gh CLI     = authenticated on %s", host)
		} else {
			warn("gh CLI not authed for %s — run: gh auth login --hostname %s", host, host)
		}
	}
}
