package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// ghAccount is one account `gh` is logged into, plus what GitHub says about it.
type ghAccount struct {
	Login, Host, Name, Email string
	Keys                     []string // registered SSH public keys, "type base64"
}

// ghAccounts lists every account in gh's keyring and looks each one up with
// its own token, so we can propose a profile per account.
func ghAccounts() []ghAccount {
	if !hasCmd("gh") {
		return nil
	}
	out, err := ghCmd("auth", "status", "--json", "hosts", "--jq", `.hosts[][] | .login + " " + .host`).Output()
	if err != nil {
		return nil
	}
	var accts []ghAccount
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		login, host, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		a := ghAccount{Login: login, Host: host}
		tok, err := ghCmd("auth", "token", "--hostname", host, "--user", login).Output()
		if err == nil {
			api := func(path string, v any) {
				c := ghCmd("api", "--hostname", host, path)
				c.Env = append(c.Env, "GH_TOKEN="+strings.TrimSpace(string(tok)))
				if b, err := c.Output(); err == nil {
					_ = json.Unmarshal(b, v)
				}
			}
			var u struct {
				ID          int64
				Name, Email string
			}
			api("user", &u)
			a.Name, a.Email = u.Name, u.Email
			if a.Email == "" && u.ID != 0 && host == "github.com" {
				// GitHub's noreply address keeps a private email private and
				// still attributes commits to this account.
				a.Email = fmt.Sprintf("%d+%s@users.noreply.github.com", u.ID, login)
			}
			var keys []struct{ Key string }
			api("user/keys", &keys)
			for _, k := range keys {
				a.Keys = append(a.Keys, k.Key)
			}
		}
		accts = append(accts, a)
	}
	return accts
}

// localSSHKeys maps each ~/.ssh private key that has a .pub to its
// "type base64" public key, for matching against GitHub's registered keys.
func localSSHKeys() map[string]string {
	home, _ := os.UserHomeDir()
	pubs, _ := filepath.Glob(filepath.Join(home, ".ssh", "*.pub"))
	keys := map[string]string{}
	for _, pub := range pubs {
		b, err := os.ReadFile(pub)
		if err != nil {
			continue
		}
		f := strings.Fields(string(b))
		priv := strings.TrimSuffix(pub, ".pub")
		if _, err := os.Stat(priv); err == nil && len(f) >= 2 {
			keys["~"+strings.TrimPrefix(priv, home)] = f[0] + " " + f[1]
		}
	}
	return keys
}

func promptDefault(label, def string) string {
	if def != "" {
		label = fmt.Sprintf("%s [%s]", label, def)
	}
	if v := prompt(label + ":"); v != "" {
		return v
	}
	return def
}

// newInitCmd is first-run setup: one profile per gh account (name, email
// and SSH key detected), optional folder rules, and shell integration.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Guided setup: detect gh accounts + SSH keys, create profiles, enable shell integration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			section("ghswitch setup")
			info("detecting gh accounts and SSH keys…")
			accts := ghAccounts()
			if len(accts) == 0 {
				warn("no gh accounts found — falling back to your global git identity")
				info("tip: 'gh auth login' (once per account) lets init detect everything")
				accts = []ghAccount{{
					Name:  gitutil.ConfigGet("--global", "user.name"),
					Email: gitutil.ConfigGet("--global", "user.email"),
				}}
			}
			local := localSSHKeys()

			for _, a := range accts {
				section("%s", firstNonEmpty(a.Login, "global identity"))
				if existing := profileForLogin(a.Login); existing != "" {
					info("already configured as profile %q — skipping", existing)
					continue
				}
				if !confirm("Create a profile for this account? (Y/n):") {
					continue
				}
				p := config.Profile{
					Name:    strings.ToLower(strings.ReplaceAll(promptDefault("Profile name", strings.ToLower(a.Login)), " ", "-")),
					GitName: promptDefault("Git name", firstNonEmpty(a.Name, a.Login)),
					Email:   promptDefault("Email", a.Email),
					GHUser:  a.Login,
					GHHost:  a.Host,
				}
				for path, pub := range local {
					for _, k := range a.Keys {
						if pub == k {
							p.SSHKey = path
							ok("SSH key %s is registered to %s", path, a.Login)
						}
					}
				}
				if p.SSHKey == "" && len(local) > 0 {
					opts := []string{"(none)"}
					for path := range local {
						opts = append(opts, path)
					}
					if k, err := selectOne("SSH key for "+p.Name, opts); err == nil && k != "(none)" {
						p.SSHKey = k
					}
				}
				if d := prompt("Folder that should always use this profile (e.g. ~/work, blank to skip):"); d != "" {
					p.Dirs = []string{d}
				}
				if p.Name == "" || p.Email == "" {
					warn("profile name and email are required — skipped")
					continue
				}
				if err := config.Add(p); err != nil {
					errf("%v", err)
					continue
				}
				ok("profile %q saved", p.Name)
			}

			setupShell()
			section("Done")
			info("switch with: ghswitch <profile>   ·   check with: ghswitch doctor")
			return nil
		},
	}
}

func profileForLogin(login string) string {
	if login == "" {
		return ""
	}
	profiles, _ := config.Load()
	for _, p := range profiles {
		if p.GHUser == login {
			return p.Name
		}
	}
	return ""
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// setupShell appends the shellinit line to the user's rc file, with consent.
func setupShell() {
	if os.Getenv("GHSWITCH_SHELLINIT") != "" {
		return
	}
	rc := rcFile()
	if !strings.HasPrefix(rc, "~/") {
		info("add to your shell rc:  %s", shellinitLine)
		return
	}
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, rc[2:])
	if b, err := os.ReadFile(path); err == nil && strings.Contains(string(b), "ghswitch shellinit") {
		info("%s already loads ghswitch — open a new terminal to pick it up", rc)
		return
	}
	section("Shell integration")
	info("gives each terminal its own identity and auto-switches by folder")
	if !confirm(fmt.Sprintf("Add `%s` to %s? (Y/n):", shellinitLine, rc)) {
		info("add it yourself later:  %s", shellinitLine)
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		errf("%v", err)
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\n# ghswitch: per-terminal GitHub identity\n%s\n", shellinitLine)
	ok("added to %s — run: exec $SHELL", rc)
}
