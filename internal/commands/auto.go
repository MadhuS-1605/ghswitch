package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// readPreset reads the `profile:` key from .ghswitch.yml in the repo root,
// same one-line lookup the bash version did with awk (ghswitch.sh:1082-1088).
func readPreset(root string) string {
	b, err := os.ReadFile(filepath.Join(root, ".ghswitch.yml"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "profile" {
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)
			return v
		}
	}
	return ""
}

func newAutoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auto",
		Short: "Detect a matching profile (.ghswitch.yml or user.email) and apply locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			root := gitutil.RepoRoot()
			preset := readPreset(root)
			curEmail := gitutil.ConfigGet("", "user.email")

			section("Auto-detect profile")
			if preset != "" {
				info("preset     : .ghswitch.yml → %s", preset)
			}
			info("user.email : %s", orNotSet(curEmail))

			var best config.Profile
			var found bool
			if preset != "" {
				best, found = config.Get(preset)
				if !found {
					warn("Preset profile %q not found locally", preset)
				}
			}
			if !found && curEmail != "" {
				profiles, _ := config.Load()
				for _, p := range profiles {
					if p.Email == curEmail {
						best, found = p, true
						break
					}
				}
			}

			if !found {
				warn("No matching profile (preset or user.email)")
				info("Use 'ghswitch local' to pick one manually")
				return nil
			}

			ok("Match: %s", best.Name)
			if !confirm("Apply to this repo? (Y/n):") {
				info("Cancelled.")
				return nil
			}
			return applyLocal(best)
		},
	}
}

func orNotSet(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}
