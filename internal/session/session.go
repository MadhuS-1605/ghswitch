// Package session implements gitswitch.dev's core differentiator: per-shell
// identity isolation. Instead of mutating ~/.gitconfig (shared across every
// terminal), a profile switch prints `export GIT_AUTHOR_*=...` etc. to
// stdout; `eval "$(ghswitch <profile>)"` (installed by `shellinit`) applies
// those exports to the *current* shell only, so two terminals can run two
// different GitHub identities at once.
package session

import (
	"fmt"
	"io"

	"github.com/MadhuS-1605/ghswitch/internal/config"
)

// Keys lists every env var this package ever exports, in export order —
// also the set `unset` must clear.
var Keys = []string{
	"GIT_AUTHOR_NAME",
	"GIT_AUTHOR_EMAIL",
	"GIT_COMMITTER_NAME",
	"GIT_COMMITTER_EMAIL",
	"GIT_SSH_COMMAND",
	"GHSWITCH_ACTIVE_PROFILE",
	"GHSWITCH_EXPIRES",
}

// EnvFor builds the session env vars for a profile. Git reads
// GIT_AUTHOR_*/GIT_COMMITTER_* in preference to user.name/user.email, and
// GIT_SSH_COMMAND in preference to core.sshCommand — both env-var forms win
// over global config without touching it.
func EnvFor(p config.Profile) map[string]string {
	env := map[string]string{
		"GHSWITCH_ACTIVE_PROFILE": p.Name,
	}
	if p.GitName != "" {
		env["GIT_AUTHOR_NAME"] = p.GitName
		env["GIT_COMMITTER_NAME"] = p.GitName
	}
	if p.Email != "" {
		env["GIT_AUTHOR_EMAIL"] = p.Email
		env["GIT_COMMITTER_EMAIL"] = p.Email
	}
	if p.SSHKey != "" {
		expanded := config.ExpandSSHKey(p.SSHKey)
		env["GIT_SSH_COMMAND"] = fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", expanded)
	}
	return env
}

// WriteExports writes POSIX `export KEY='VALUE'` lines, one per key in
// Keys order (only keys present in env are emitted). Values are single-quote
// escaped since GitHub tokens/names may contain shell-special characters.
func WriteExports(w io.Writer, env map[string]string) {
	for _, k := range Keys {
		v, ok := env[k]
		if !ok {
			continue
		}
		fmt.Fprintf(w, "export %s=%s\n", k, shQuote(v))
	}
}

// WriteUnsets writes `unset KEY1 KEY2 ...` clearing every session var —
// used by `ghswitch unset --session` to drop back to global/local config.
func WriteUnsets(w io.Writer) {
	fmt.Fprintf(w, "unset")
	for _, k := range Keys {
		fmt.Fprintf(w, " %s", k)
	}
	fmt.Fprintln(w)
}

func shQuote(s string) string {
	// POSIX single-quote escaping: close quote, escaped literal quote, reopen.
	out := "'"
	for _, r := range s {
		if r == '\'' {
			out += `'\''`
		} else {
			out += string(r)
		}
	}
	return out + "'"
}
