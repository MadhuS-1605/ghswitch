package config

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if got.Hosts[0].SSHKey != "~/.ssh/id_corp" || got.Hosts[0].GHToken != "ghp_corp" {
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
