package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// Never touch the developer's real keychain from tests.
func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

// TestMultiHostRoundTrip verifies that a profile with an extra [name@host]
// section survives a Save/Load cycle, and that ForHost picks the right
// creds for the matching host while falling back to the primary ones.
func TestMultiHostRoundTrip(t *testing.T) {
	t.Setenv("GHSWITCH_CONFIG_DIR", t.TempDir())

	p := Profile{
		Name:    "work",
		GitName: "Jane Doe",
		Email:   "jane@company.com",
		SSHKey:  "~/.ssh/id_work",
		GHHost:  "github.com",
		Hosts: []HostCred{
			{Host: "github.corp.com", SSHKey: "~/.ssh/id_corp", GHToken: "ghp_corp"},
		},
	}
	if err := Save([]Profile{p}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded))
	}
	got := loaded[0]
	if len(got.Hosts) != 1 || got.Hosts[0].Host != "github.corp.com" {
		t.Fatalf("expected 1 host github.corp.com, got %+v", got.Hosts)
	}
	if got.Hosts[0].SSHKey != "~/.ssh/id_corp" || got.ForHost("github.corp.com").Token() != "ghp_corp" {
		t.Fatalf("host creds not round-tripped: %+v", got.Hosts[0])
	}

	corp := got.ForHost("github.corp.com")
	if corp.SSHKey != "~/.ssh/id_corp" {
		t.Fatalf("ForHost(corp) SSHKey = %q, want ~/.ssh/id_corp", corp.SSHKey)
	}
	primary := got.ForHost("github.com")
	if primary.SSHKey != "~/.ssh/id_work" {
		t.Fatalf("ForHost(primary) SSHKey = %q, want ~/.ssh/id_work", primary.SSHKey)
	}
	unknown := got.ForHost("")
	if unknown.SSHKey != "~/.ssh/id_work" {
		t.Fatalf("ForHost(\"\") should fall back to primary, got %q", unknown.SSHKey)
	}
}

func TestSigningKey(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "id_ed25519")
	if f, k := (Profile{GPGKey: "ABC", SSHKey: key}).SigningKey(); f != "openpgp" || k != "ABC" {
		t.Fatalf("gpg_key should win, got %q %q", f, k)
	}
	if _, k := (Profile{SSHKey: key}).SigningKey(); k != "" {
		t.Fatalf("missing .pub should disable signing, got %q", k)
	}
	_ = os.WriteFile(key+".pub", []byte("x"), 0o600)
	if f, k := (Profile{SSHKey: key}).SigningKey(); f != "ssh" || k != key+".pub" {
		t.Fatalf("want ssh %s.pub, got %q %q", key, f, k)
	}
}

// Tokens go to the keychain, never back into profiles.conf, and survive
// a Load→Save cycle that never read them.
func TestTokenStoredInKeyring(t *testing.T) {
	t.Setenv("GHSWITCH_CONFIG_DIR", t.TempDir())
	if err := Save([]Profile{{Name: "work", Email: "a@b", GHToken: "ghp_secret"}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(ConfigFile())
	if strings.Contains(string(b), "ghp_secret") || !strings.Contains(string(b), "gh_token=keyring") {
		t.Fatalf("token leaked into profiles.conf:\n%s", b)
	}
	loaded, _ := Load()
	if loaded[0].GHToken != "" || !loaded[0].HasToken() {
		t.Fatalf("Load should defer the keychain read, got %+v", loaded[0])
	}
	if err := Save(loaded); err != nil {
		t.Fatal(err)
	}
	again, _ := Load()
	if got := again[0].Token(); got != "ghp_secret" {
		t.Fatalf("token lost after resave: %q", got)
	}
	if err := Delete("work"); err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.Get(keyringService, "work"); err == nil {
		t.Fatal("Delete should remove the keychain entry")
	}
}

func TestMatchDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	profiles := []Profile{
		{Name: "work", Dirs: []string{"~/work"}},
		{Name: "acme", Dirs: []string{"~/work/acme"}},
	}
	cases := map[string]string{
		home + "/work":          "work",
		home + "/work/api":      "work",
		home + "/work/acme/web": "acme", // deepest rule wins
		home + "/workshop":      "",     // prefix of a name is not a match
		home:                    "",
	}
	for cwd, want := range cases {
		p, _, ok := MatchDir(profiles, cwd)
		if (want == "") == ok || p.Name != want {
			t.Errorf("MatchDir(%s) = %q, %v; want %q", cwd, p.Name, ok, want)
		}
	}
}

func TestDirsRoundTrip(t *testing.T) {
	t.Setenv("GHSWITCH_CONFIG_DIR", t.TempDir())
	_ = Save([]Profile{{Name: "work", Email: "a@b", Dirs: []string{"~/work", "~/clients"}}})
	loaded, _ := Load()
	if strings.Join(loaded[0].Dirs, ",") != "~/work,~/clients" {
		t.Fatalf("dirs not round-tripped: %v", loaded[0].Dirs)
	}
}
