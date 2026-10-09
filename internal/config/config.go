// Package config reads and writes ~/.config/ghswitch/profiles.conf, the same
// INI-style format ghswitch.sh has always used — no migration needed.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Profile mirrors one [name] section of profiles.conf.
type Profile struct {
	Name    string // section name, e.g. "work"
	GitName string // git user.name
	Email   string
	SSHKey  string
	GPGKey  string
	GHToken string
	GHHost  string
	GHUser  string     // GitHub login for `gh auth switch`; defaults to GitName
	Hosts   []HostCred // additional hosts for this profile (multi-host), from [name@host] sections
}

// HostCred overrides SSH key / GPG key / token for one additional host on a
// profile — e.g. a "work" profile used against both github.com and a GitHub
// Enterprise host, each with its own key and token.
type HostCred struct {
	Host    string
	SSHKey  string
	GPGKey  string
	GHToken string
}

// ForHost returns a copy of p with SSH/GPG/token overridden by the matching
// entry in p.Hosts, if any. Falls back to p's own (primary-host) credentials
// otherwise, so callers can always just call ForHost(remoteHost) even when
// the profile has no extra hosts configured.
func (p Profile) ForHost(host string) Profile {
	if host == "" {
		return p
	}
	for _, h := range p.Hosts {
		if h.Host != host {
			continue
		}
		out := p
		out.GHHost = h.Host
		if h.SSHKey != "" {
			out.SSHKey = h.SSHKey
		}
		if h.GPGKey != "" {
			out.GPGKey = h.GPGKey
		}
		if h.GHToken != "" {
			out.GHToken = h.GHToken
		}
		return out
	}
	return p
}

// Login is the GitHub account name gh knows this profile by.
func (p Profile) Login() string {
	if p.GHUser != "" {
		return p.GHUser
	}
	return p.GitName
}

// SigningKey picks how this profile signs commits: an explicit gpg_key
// (openpgp), else the SSH key's .pub (ssh signing), else nothing — so a
// switch never leaves the previous profile's key signing your commits.
func (p Profile) SigningKey() (format, key string) {
	if p.GPGKey != "" {
		return "openpgp", p.GPGKey
	}
	if p.SSHKey != "" {
		pub := ExpandSSHKey(p.SSHKey) + ".pub"
		if _, err := os.Stat(pub); err == nil {
			return "ssh", pub
		}
	}
	return "", ""
}

func Dir() string {
	if d := os.Getenv("GHSWITCH_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghswitch")
}

func ConfigFile() string { return filepath.Join(Dir(), "profiles.conf") }

// ActiveFile is the legacy global-active pointer, kept only as a fallback for
// tools/checks that don't read the per-session env vars (see internal/session).
func ActiveFile() string { return filepath.Join(Dir(), "active") }

// PreviousFile tracks the global-active profile as it was before the most
// recent switch, so `ghswitch back` can jump to it.
func PreviousFile() string { return filepath.Join(Dir(), "previous") }

// AuditFile is an append-only local log of switches (timestamp, profile,
// cwd) — a record of what identity was active when, not a sync mechanism.
func AuditFile() string { return filepath.Join(Dir(), "switch.log") }

func EnsureDir() error {
	return os.MkdirAll(Dir(), 0o755)
}

// Load parses profiles.conf, preserving section order.
func Load() ([]Profile, error) {
	f, err := os.Open(ConfigFile())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var profiles []Profile
	var cur *Profile      // current [name] section
	var curHost *HostCred // current [name@host] sub-section, if any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inner := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if base, host, ok := strings.Cut(inner, "@"); ok {
				curHost = nil
				for i := range profiles {
					if profiles[i].Name == base {
						profiles[i].Hosts = append(profiles[i].Hosts, HostCred{Host: host})
						curHost = &profiles[i].Hosts[len(profiles[i].Hosts)-1]
						break
					}
				}
				cur = nil
				continue
			}
			profiles = append(profiles, Profile{Name: inner})
			cur = &profiles[len(profiles)-1]
			curHost = nil
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if curHost != nil {
			switch key {
			case "ssh_key":
				curHost.SSHKey = val
			case "gpg_key":
				curHost.GPGKey = val
			case "gh_token":
				curHost.GHToken = val
			}
			continue
		}
		if cur == nil {
			continue
		}
		switch key {
		case "name":
			cur.GitName = val
		case "email":
			cur.Email = val
		case "ssh_key":
			cur.SSHKey = val
		case "gpg_key":
			cur.GPGKey = val
		case "gh_token":
			cur.GHToken = val
		case "gh_host":
			cur.GHHost = val
		case "gh_user":
			cur.GHUser = val
		}
	}
	return profiles, sc.Err()
}

func Get(name string) (Profile, bool) {
	profiles, _ := Load()
	for _, p := range profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

func Exists(name string) bool {
	_, ok := Get(name)
	return ok
}

// Save rewrites profiles.conf from scratch in the same [name]\nkey=value\n
// format the bash script produced.
func Save(profiles []Profile) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	var b strings.Builder
	for _, p := range profiles {
		fmt.Fprintf(&b, "[%s]\n", p.Name)
		fmt.Fprintf(&b, "name=%s\n", p.GitName)
		fmt.Fprintf(&b, "email=%s\n", p.Email)
		if p.SSHKey != "" {
			fmt.Fprintf(&b, "ssh_key=%s\n", p.SSHKey)
		}
		if p.GPGKey != "" {
			fmt.Fprintf(&b, "gpg_key=%s\n", p.GPGKey)
		}
		if p.GHToken != "" {
			fmt.Fprintf(&b, "gh_token=%s\n", p.GHToken)
		}
		if p.GHUser != "" {
			fmt.Fprintf(&b, "gh_user=%s\n", p.GHUser)
		}
		host := p.GHHost
		if host == "" {
			host = "github.com"
		}
		fmt.Fprintf(&b, "gh_host=%s\n\n", host)

		for _, h := range p.Hosts {
			fmt.Fprintf(&b, "[%s@%s]\n", p.Name, h.Host)
			if h.SSHKey != "" {
				fmt.Fprintf(&b, "ssh_key=%s\n", h.SSHKey)
			}
			if h.GPGKey != "" {
				fmt.Fprintf(&b, "gpg_key=%s\n", h.GPGKey)
			}
			if h.GHToken != "" {
				fmt.Fprintf(&b, "gh_token=%s\n", h.GHToken)
			}
			b.WriteString("\n")
		}
	}
	return os.WriteFile(ConfigFile(), []byte(b.String()), 0o600)
}

func Add(p Profile) error {
	if Exists(p.Name) {
		return fmt.Errorf("profile %q already exists", p.Name)
	}
	if p.GHHost == "" {
		p.GHHost = "github.com"
	}
	profiles, err := Load()
	if err != nil {
		return err
	}
	return Save(append(profiles, p))
}

func Delete(name string) error {
	profiles, err := Load()
	if err != nil {
		return err
	}
	out := profiles[:0]
	for _, p := range profiles {
		if p.Name != name {
			out = append(out, p)
		}
	}
	return Save(out)
}

// ExpandSSHKey expands a leading ~ the same way the bash script's `eval echo`
// did, without shelling out.
func ExpandSSHKey(path string) string {
	if path == "" {
		return ""
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	return path
}

// GetActiveGlobal / SetActiveGlobal manage the legacy global-active pointer.
func GetActiveGlobal() string {
	b, err := os.ReadFile(ActiveFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func SetActiveGlobal(name string) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return os.WriteFile(ActiveFile(), []byte(name), 0o644)
}

// GetPreviousGlobal / SetPreviousGlobal back `ghswitch back`.
func GetPreviousGlobal() string {
	b, err := os.ReadFile(PreviousFile())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func SetPreviousGlobal(name string) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return os.WriteFile(PreviousFile(), []byte(name), 0o644)
}

// AppendAudit records one switch event. Best-effort: a failure here should
// never block the switch itself.
func AppendAudit(profile string) {
	if err := EnsureDir(); err != nil {
		return
	}
	cwd, _ := os.Getwd()
	f, err := os.OpenFile(AuditFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s\t%s\t%s\n", time.Now().Format("2006-01-02T15:04:05Z07:00"), profile, cwd)
}
