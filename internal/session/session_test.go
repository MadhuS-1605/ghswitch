package session

import (
	"bytes"
	"strings"
	"testing"

	"github.com/MadhuS-1605/ghswitch/internal/config"
)

func TestEnvForSigning(t *testing.T) {
	gpg := EnvFor(config.Profile{Name: "w", GitName: "J", Email: "j@x", GPGKey: "ABC"})
	if gpg["GIT_CONFIG_COUNT"] != "3" || gpg["GIT_CONFIG_VALUE_0"] != "ABC" || gpg["GIT_CONFIG_VALUE_2"] != "true" {
		t.Fatalf("gpg profile should set key, format and gpgSign: %v", gpg)
	}
	// No key must actively disable signing, or the previous profile's
	// global signingKey would sign this profile's commits.
	none := EnvFor(config.Profile{Name: "p", Email: "p@x"})
	if none["GIT_CONFIG_COUNT"] != "1" || none["GIT_CONFIG_KEY_0"] != "commit.gpgSign" || none["GIT_CONFIG_VALUE_0"] != "false" {
		t.Fatalf("keyless profile should turn signing off: %v", none)
	}
	if none["GIT_AUTHOR_EMAIL"] != "p@x" || none["GHSWITCH_ACTIVE_PROFILE"] != "p" {
		t.Fatalf("identity missing: %v", none)
	}
}

func TestWriteExports(t *testing.T) {
	var b bytes.Buffer
	WriteExports(&b, map[string]string{"GIT_AUTHOR_NAME": "O'Brien"})
	out := b.String()
	if !strings.Contains(out, `export GIT_AUTHOR_NAME='O'\''Brien'`) {
		t.Fatalf("quote not escaped:\n%s", out)
	}
	// Everything not exported is unset, so nothing leaks from the last profile.
	if !strings.Contains(out, "unset GH_TOKEN\n") || !strings.Contains(out, "unset GIT_CONFIG_KEY_2\n") {
		t.Fatalf("stale vars not unset:\n%s", out)
	}
}
