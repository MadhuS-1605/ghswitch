package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// newLoginCmd authenticates via GitHub's OAuth device flow — but instead of
// reimplementing device-flow HTTP calls and a public client ID ourselves,
// it delegates to `gh auth login`, which already does this correctly. The
// resulting token is captured and saved into a profile, replacing the old
// bash `add_profile` manual-paste-a-PAT flow.
func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login [profile]",
		Short: "Authenticate via GitHub device-flow login (delegates to gh auth login) and save the token to a profile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireGH() {
				return nil
			}
			host := prompt("GitHub host (blank = github.com):")
			if host == "" {
				host = "github.com"
			}
			section("Device-flow login for %s", host)
			info("gh will print a one-time code — follow the browser prompt.")
			if err := runInherit("gh", "auth", "login", "--hostname", host); err != nil {
				errf("gh auth login failed: %v", err)
				return nil
			}
			tokenOut, err := exec.Command("gh", "auth", "token", "--hostname", host).Output()
			if err != nil {
				errf("Could not read token from gh: %v", err)
				return nil
			}
			token := strings.TrimSpace(string(tokenOut))

			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				name = prompt("Save as which profile name? (existing name updates it, new name creates it):")
			}
			if name == "" {
				errf("Profile name required.")
				return nil
			}

			if p, found := config.Get(name); found {
				p.GHToken = token
				p.GHHost = host
				profiles, _ := config.Load()
				for i := range profiles {
					if profiles[i].Name == name {
						profiles[i] = p
					}
				}
				if err := config.Save(profiles); err != nil {
					errf("%v", err)
					return nil
				}
				ok("Updated profile %q with new token.", name)
				return nil
			}

			gitName := prompt("Git name for this profile:")
			email := prompt("Email:")
			p := config.Profile{Name: name, GitName: gitName, Email: email, GHToken: token, GHHost: host}
			if err := config.Add(p); err != nil {
				errf("%v", err)
				return nil
			}
			ok("Profile %q created and authenticated.", name)
			return nil
		},
	}
}

func newRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate [profile]",
		Short: "Open settings page, paste new token, rewrite profiles.conf, re-auth gh",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				profiles, _ := config.Load()
				if len(profiles) == 0 {
					warn("No profiles yet. Run: ghswitch add")
					return nil
				}
				names := make([]string, len(profiles))
				for i, p := range profiles {
					names[i] = p.Name
				}
				chosen, err := selectOne("Rotate token for which profile?", names)
				if err != nil {
					info("Cancelled.")
					return nil
				}
				name = chosen
			}
			p, found := config.Get(name)
			if !found {
				errf("Profile %q not found.", name)
				return nil
			}
			host := p.GHHost
			if host == "" {
				host = "github.com"
			}
			section("Rotate token for %s", name)
			_ = openURL(fmt.Sprintf("https://%s/settings/tokens", host))
			newToken := prompt("Paste new token:")
			if newToken == "" {
				errf("Token cannot be empty.")
				return nil
			}
			p.GHToken = newToken
			profiles, _ := config.Load()
			for i := range profiles {
				if profiles[i].Name == name {
					profiles[i] = p
				}
			}
			if err := config.Save(profiles); err != nil {
				errf("%v", err)
				return nil
			}
			ok("Token updated for %q.", name)
			applyGH(p)
			return nil
		},
	}
}

func newReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release",
		Short: "Wizard: bump (patch/minor/major/custom) -> tag -> push -> gh release create",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			latest, _ := gitutil.Run("describe", "--tags", "--abbrev=0")
			info("Latest tag: %s", orNotSet(latest))

			bump, err := selectOne("Bump", []string{"patch", "minor", "major", "custom"})
			if err != nil {
				info("Cancelled.")
				return nil
			}
			var next string
			if bump == "custom" {
				next = prompt("New tag (e.g. v2.0.0):")
			} else {
				next = bumpSemver(latest, bump)
			}
			if next == "" {
				errf("Could not determine next version — enter manually.")
				return nil
			}
			info("Next tag: %s", next)
			if !confirm(fmt.Sprintf("Create and push %s? (y/N):", next)) {
				info("Cancelled.")
				return nil
			}
			notes := prompt("Release notes (blank = gh auto-generate):")
			if _, err := gitutil.Run("tag", "-a", next, "-m", next); err != nil {
				errf("Tag failed: %v", err)
				return nil
			}
			if _, err := gitutil.Run("push", "origin", next); err != nil {
				errf("Push tag failed: %v", err)
				return nil
			}
			ok("Tag %s pushed.", next)
			if !requireGH() {
				return nil
			}
			ghArgs := []string{"release", "create", next}
			if notes != "" {
				ghArgs = append(ghArgs, "--notes", notes)
			} else {
				ghArgs = append(ghArgs, "--generate-notes")
			}
			return runInherit("gh", ghArgs...)
		},
	}
}

// bumpSemver bumps a "vMAJOR.MINOR.PATCH" tag; returns "" if latest doesn't
// parse (caller falls back to asking for a custom tag).
func bumpSemver(latest, part string) string {
	v := strings.TrimPrefix(latest, "v")
	segs := strings.SplitN(v, ".", 3)
	if len(segs) != 3 {
		return ""
	}
	major, e1 := strconv.Atoi(segs[0])
	minor, e2 := strconv.Atoi(segs[1])
	patch, e3 := strconv.Atoi(segs[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return ""
	}
	switch part {
	case "major":
		major, minor, patch = major+1, 0, 0
	case "minor":
		minor, patch = minor+1, 0
	case "patch":
		patch++
	}
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}
