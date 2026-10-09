package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/MadhuS-1605/ghswitch/internal/config"
)

func TestAutoTarget(t *testing.T) {
	home, _ := os.UserHomeDir()
	profiles := []config.Profile{
		{Name: "work", Dirs: []string{"~/work"}},
		{Name: "oss"},
	}
	// .ghswitch.yml beats a dir rule.
	if p, rule, ok := autoTarget(profiles, home+"/work/repo", home+"/work/repo", "oss"); !ok || p.Name != "oss" || rule != "repo:"+home+"/work/repo" {
		t.Fatalf("preset should win, got %q %q", p.Name, rule)
	}
	// Unknown preset falls back to the dir rule.
	if p, rule, _ := autoTarget(profiles, home+"/work/repo", home+"/work/repo", "nope"); p.Name != "work" || rule != "dir:"+home+"/work" {
		t.Fatalf("want dir rule, got %q %q", p.Name, rule)
	}
	if _, _, ok := autoTarget(profiles, home+"/elsewhere", "", ""); ok {
		t.Fatal("no rule should match outside ~/work")
	}
}

func TestReadCredentialHost(t *testing.T) {
	if h := readCredentialHost(strings.NewReader("protocol=https\nhost=github.com\n\n")); h != "github.com" {
		t.Fatalf("got %q", h)
	}
	// Never hand a token out over plain http.
	if h := readCredentialHost(strings.NewReader("protocol=http\nhost=github.com\n\n")); h != "" {
		t.Fatalf("http must be refused, got %q", h)
	}
}

func TestProfileServes(t *testing.T) {
	p := config.Profile{GHHost: "github.com", Hosts: []config.HostCred{{Host: "ghe.corp"}}}
	if !profileServes(p, "github.com") || !profileServes(p, "ghe.corp") || profileServes(p, "gitlab.com") {
		t.Fatal("profileServes must only match the profile's own hosts")
	}
	if !profileServes(config.Profile{}, "github.com") {
		t.Fatal("empty gh_host means github.com")
	}
}
