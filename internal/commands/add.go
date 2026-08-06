package commands

import (
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new profile (guided prompts)",
		RunE: func(cmd *cobra.Command, args []string) error {
			section("Add new profile")

			name := strings.ToLower(strings.ReplaceAll(prompt("Profile name (e.g. work, personal):"), " ", "-"))
			if name == "" {
				errf("Name cannot be empty.")
				return nil
			}
			if config.Exists(name) {
				warn("Profile %q already exists. Use 'ghswitch edit' to modify.", name)
				return nil
			}

			p := config.Profile{
				Name:    name,
				GitName: prompt("Git name (e.g. Jane Doe):"),
				Email:   prompt("Email:"),
				SSHKey:  prompt("SSH key path (blank to skip, e.g. ~/.ssh/id_ed25519_work):"),
				GPGKey:  prompt("GPG key ID (blank to skip):"),
				GHToken: prompt("GitHub Token (blank to skip):"),
				GHHost:  "github.com",
			}

			if err := config.Add(p); err != nil {
				errf("%v", err)
				return nil
			}
			ok("Profile %q saved.", name)

			if confirm("Apply this profile now? (y/N):") {
				applyGlobal(p, 0)
			}
			return nil
		},
	}
}
