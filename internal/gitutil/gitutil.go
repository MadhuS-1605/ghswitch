// Package gitutil wraps `git` invocations, mirroring the context helpers
// ghswitch.sh defined at lines 522-541 (in_repo, current_branch, repo_root,
// upstream_of, dirty_count, ahead_behind, default_branch).
package gitutil

import (
	"os/exec"
	"strconv"
	"strings"
)

// Run executes git with args and returns trimmed stdout. Errors are returned
// as-is; callers that only care about success/output use OK.
func Run(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// OK runs git and reports whether it exited 0, discarding output.
func OK(args ...string) bool {
	_, err := Run(args...)
	return err == nil
}

func InRepo() bool { return OK("rev-parse", "--is-inside-work-tree") }

func CurrentBranch() string {
	out, err := Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

func RepoRoot() string {
	out, _ := Run("rev-parse", "--show-toplevel")
	return out
}

// RemoteHost extracts the host from the "origin" remote URL, handling both
// SSH ("git@host:owner/repo.git") and HTTPS ("https://host/owner/repo.git")
// forms. Returns "" if there's no origin remote or the URL doesn't parse.
func RemoteHost() string {
	url, err := Run("remote", "get-url", "origin")
	if err != nil || url == "" {
		return ""
	}
	if strings.HasPrefix(url, "git@") {
		rest := strings.TrimPrefix(url, "git@")
		host, _, ok := strings.Cut(rest, ":")
		if !ok {
			return ""
		}
		return host
	}
	for _, prefix := range []string{"https://", "http://", "ssh://git@", "ssh://"} {
		if strings.HasPrefix(url, prefix) {
			rest := strings.TrimPrefix(url, prefix)
			host, _, _ := strings.Cut(rest, "/")
			host, _, _ = strings.Cut(host, "@")
			return host
		}
	}
	return ""
}

// UpstreamOf returns the upstream ref for the given ref ("HEAD" if empty),
// or "" if none is configured.
func UpstreamOf(ref string) string {
	if ref == "" {
		ref = "HEAD"
	}
	out, err := Run("rev-parse", "--abbrev-ref", "--symbolic-full-name", ref+"@{u}")
	if err != nil {
		return ""
	}
	return out
}

func DirtyCount() int {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return 0
	}
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// AheadBehind returns (ahead, behind) commit counts vs upstream, or
// (0, 0, false) if there is no upstream.
func AheadBehind() (ahead, behind int, ok bool) {
	up := UpstreamOf("")
	if up == "" {
		return 0, 0, false
	}
	out, err := Run("rev-list", "--left-right", "--count", up+"...HEAD")
	if err != nil {
		return 0, 0, false
	}
	parts := strings.Fields(out)
	if len(parts) != 2 {
		return 0, 0, false
	}
	behind, _ = strconv.Atoi(parts[0])
	ahead, _ = strconv.Atoi(parts[1])
	return ahead, behind, true
}

func DefaultBranch() string {
	out, err := Run("symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil || out == "" {
		return "main"
	}
	return strings.TrimPrefix(out, "refs/remotes/origin/")
}

// ConfigGet reads a git config value at the given scope ("--global",
// "--local", or "" for the effective value).
func ConfigGet(scope, key string) string {
	args := []string{"config"}
	if scope != "" {
		args = append(args, scope)
	}
	args = append(args, key)
	out, _ := Run(args...)
	return out
}

func ConfigSet(scope, key, value string) error {
	_, err := Run("config", scope, key, value)
	return err
}

func ConfigUnset(scope, key string) error {
	_, err := Run("config", scope, "--unset", key)
	return err
}
