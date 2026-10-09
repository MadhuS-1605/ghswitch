package commands

import (
	"fmt"
	"os"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all profiles with active marker",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := config.Load()
			if err != nil {
				return err
			}
			active := os.Getenv("GHSWITCH_ACTIVE_PROFILE")
			if active == "" {
				active = config.GetActiveGlobal()
			}

			section("Configured profiles")
			if len(profiles) == 0 {
				warn("No profiles yet. Run: ghswitch init")
				return nil
			}
			for _, p := range profiles {
				mark := ""
				if p.Name == active {
					mark = "  ← active"
				}
				fmt.Fprintf(os.Stderr, "\n  [%s]%s\n", p.Name, mark)
				fmt.Fprintf(os.Stderr, "    name  : %s\n", p.GitName)
				fmt.Fprintf(os.Stderr, "    email : %s\n", p.Email)
				if p.SSHKey != "" {
					fmt.Fprintf(os.Stderr, "    ssh   : %s\n", p.SSHKey)
				}
				if p.GPGKey != "" {
					fmt.Fprintf(os.Stderr, "    gpg   : %s\n", p.GPGKey)
				}
				if p.HasToken() {
					fmt.Fprintf(os.Stderr, "    token : [configured]\n")
				}
				if p.GHHost != "" {
					fmt.Fprintf(os.Stderr, "    host  : %s\n", p.GHHost)
				}
			}
			fmt.Fprintln(os.Stderr)
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "delete",
		Aliases: []string{"rm"},
		Short:   "Remove a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := config.Load()
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				warn("No profiles found.")
				return nil
			}
			names := make([]string, len(profiles))
			for i, p := range profiles {
				names[i] = p.Name
			}
			chosen, err := selectOne("Delete a profile", names)
			if err != nil {
				info("Cancelled.")
				return nil
			}
			if !confirm(fmt.Sprintf("Delete %q? This cannot be undone. (y/N):", chosen)) {
				info("Cancelled.")
				return nil
			}
			if err := config.Delete(chosen); err != nil {
				return err
			}
			ok("Profile %q deleted.", chosen)
			if config.GetActiveGlobal() == chosen {
				_ = config.SetActiveGlobal("")
			}
			return nil
		},
	}
}
