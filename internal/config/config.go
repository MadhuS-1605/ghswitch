// Package config reads and writes ~/.config/ghswitch/profiles.conf, the same
// INI-style format ghswitch.sh has always used — no migration needed.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	var cur *Profile
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			profiles = append(profiles, Profile{Name: strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")})
			cur = &profiles[len(profiles)-1]
			continue
		}
		if cur == nil {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
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
		host := p.GHHost
		if host == "" {
			host = "github.com"
		}
		fmt.Fprintf(&b, "gh_host=%s\n\n", host)
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
