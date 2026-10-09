package commands

import (
	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func newLocalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "local",
		Short: "Apply a profile to the current repo only (git --local)",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := config.Load()
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				warn("No profiles yet. Run: ghswitch init")
				return nil
			}
			if !gitutil.InRepo() {
				errf("Not inside a git repository.")
				return nil
			}
			names := make([]string, len(profiles))
			byName := map[string]config.Profile{}
			for i, p := range profiles {
				names[i] = p.Name
				byName[p.Name] = p
			}
			info("Repo: %s", gitutil.RepoRoot())
			chosen, err := selectOne("Apply profile to this repo only", names)
			if err != nil {
				info("Cancelled.")
				return nil
			}
			return applyLocal(byName[chosen])
		},
	}
}
