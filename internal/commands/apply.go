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
	if p.GitName != "" {
		_ = gitutil.ConfigSet("--global", "user.name", p.GitName)
	}
	if p.Email != "" {
		_ = gitutil.ConfigSet("--global", "user.email", p.Email)
	}
	signing := applySSHAndGPG(p, "--global")
	ghUser := applyGH(p)

	if previous := config.GetActiveGlobal(); previous != "" && previous != p.Name {
		_ = config.SetPreviousGlobal(previous)
	}
	_ = config.SetActiveGlobal(p.Name)
	config.AppendAudit(p.Name)

	env := session.EnvFor(p)
	if ttl > 0 {
		expires := time.Now().Add(ttl).Unix()
		env["GHSWITCH_EXPIRES"] = strconv.FormatInt(expires, 10)
	}
	// GH_TOKEN pins gh (and gh's git credential helper) to this profile's
	// account in this shell only — `gh auth switch` alone is global.
	if token := ghToken(p, ghUser); token != "" {
		env["GH_TOKEN"] = token
	}
	session.WriteExports(os.Stdout, env)

	summary(p, signing, ghUser)
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
	if p.GitName != "" {
		_ = gitutil.ConfigSet("--local", "user.name", p.GitName)
	}
	if p.Email != "" {
		_ = gitutil.ConfigSet("--local", "user.email", p.Email)
	}
	signing := applySSHAndGPG(p, "--local")
	ghUser := applyGH(p)
	registerRepo()

	summary(p, signing, ghUser)
	return nil
}

// summary is the single line a switch prints; problems are reported as
// warnings above it by the apply* helpers.
func summary(p config.Profile, signing, ghUser string) {
	parts := []string{fmt.Sprintf("%s <%s>", p.GitName, p.Email)}
	if signing != "" {
		parts = append(parts, signing+" signing")
	}
	if ghUser != "" {
		parts = append(parts, "gh: "+ghUser)
	}
	ok("%s  %s", color(colorBold, p.Name), strings.Join(parts, " · "))
}

// applySSHAndGPG sets the SSH command and signing key at scope, returning
// the signing format in use ("ssh", "gpg", or "" when signing is off).
func applySSHAndGPG(p config.Profile, scope string) string {
	if p.SSHKey != "" {
		expanded := config.ExpandSSHKey(p.SSHKey)
		if _, err := os.Stat(expanded); err == nil {
			_ = gitutil.ConfigSet(scope, "core.sshCommand", fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", expanded))
			if hasCmd("ssh-add") && os.Getenv("SSH_AUTH_SOCK") != "" {
				_ = exec.Command("ssh-add", "-q", expanded).Run()
			}
		} else {
			warn("SSH key not found: %s", expanded)
		}
	}

	format, key := p.SigningKey()
	if format == "openpgp" && (!hasCmd("gpg") || exec.Command("gpg", "--list-secret-keys", key).Run() != nil) {
		warn("GPG key %s not in keyring — signing disabled", key)
		key = ""
	}
	if key == "" {
		// Never leave the previous profile's key signing this one's commits.
		_ = gitutil.ConfigUnset(scope, "user.signingKey")
		if gitutil.ConfigGet(scope, "commit.gpgSign") == "true" {
			_ = gitutil.ConfigSet(scope, "commit.gpgSign", "false")
		}
		return ""
	}
	_ = gitutil.ConfigSet(scope, "gpg.format", format)
	_ = gitutil.ConfigSet(scope, "user.signingKey", key)
	if format == "openpgp" {
		_ = gitutil.ConfigSet(scope, "commit.gpgSign", "true")
		return "gpg"
	}
	return "ssh"
}

// ghCmd runs gh with GH_TOKEN/GITHUB_TOKEN stripped: a token exported by an
// earlier switch would otherwise override the keyring and make
// `gh auth switch`/`gh auth token --user` fail or answer for the old account.
func ghCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("gh", args...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GH_TOKEN=") && !strings.HasPrefix(kv, "GITHUB_TOKEN=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	return cmd
}

func ghHost(p config.Profile) string {
	if p.GHHost != "" {
		return p.GHHost
	}
	return "github.com"
}

// applyGH makes p's account the active gh account, returning that login
// ("" if gh is missing or the switch failed — with a warning saying why).
func applyGH(p config.Profile) string {
	host := ghHost(p)
	if p.GHToken != "" {
		helper := fmt.Sprintf(`!f() { echo "username=git"; echo "password=%s"; }; f`, p.GHToken)
		_ = gitutil.ConfigSet("--global", fmt.Sprintf("credential.https://%s.helper", host), helper)
		if !hasCmd("gh") {
			return ""
		}
		cmd := ghCmd("auth", "login", "--with-token", "--hostname", host)
		cmd.Stdin = strings.NewReader(p.GHToken)
		if cmd.Run() != nil {
			warn("gh rejected the token for %s — run: ghswitch rotate %s", p.Name, p.Name)
			return ""
		}
	}
	if !hasCmd("gh") {
		return ""
	}
	login := p.Login()
	if login == "" {
		return ""
	}
	if out, err := ghCmd("auth", "switch", "--hostname", host, "--user", login).CombinedOutput(); err != nil &&
		!strings.Contains(string(out), "already") {
		warn("gh has no account %q on %s — run: gh auth login, or set gh_user=<login> under [%s] in %s",
			login, host, p.Name, config.ConfigFile())
		return ""
	}
	return login
}

// ghToken returns the token gh should use for p in this shell.
func ghToken(p config.Profile, ghUser string) string {
	if p.GHToken != "" {
		return p.GHToken
	}
	if ghUser == "" {
		return ""
	}
	out, err := ghCmd("auth", "token", "--hostname", ghHost(p), "--user", ghUser).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
