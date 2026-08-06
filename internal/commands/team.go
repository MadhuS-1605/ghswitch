package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func reposFile() string { return filepath.Join(config.Dir(), "repos.list") }

// registerRepo adds the current repo root to the multi-repo registry,
// mirroring bash `_register_repo()` (ghswitch.sh:2314-2323) — called
// automatically after `local`/`auto` apply a profile.
func registerRepo() {
	root := gitutil.RepoRoot()
	if root == "" {
		return
	}
	existing := readRepos()
	for _, r := range existing {
		if r == root {
			return
		}
	}
	_ = config.EnsureDir()
	f, err := os.OpenFile(reposFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, root)
}

func readRepos() []string {
	b, err := os.ReadFile(reposFile())
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func writeRepos(repos []string) error {
	_ = config.EnsureDir()
	return os.WriteFile(reposFile(), []byte(strings.Join(repos, "\n")+"\n"), 0o644)
}

func newBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap <url> [profile]",
		Short: "git clone + apply chosen profile locally",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			section("Bootstrap repo")
			info("URL: %s", url)
			dir := strings.TrimSuffix(filepath.Base(url), ".git")
			if _, err := os.Stat(dir); err == nil {
				errf("Directory %q already exists.", dir)
				return nil
			}
			if err := runInherit("git", "clone", url, dir); err != nil {
				errf("Clone failed: %v", err)
				return nil
			}
			ok("Cloned into ./%s", dir)

			profileName := ""
			if len(args) == 2 {
				profileName = args[1]
			} else {
				profiles, _ := config.Load()
				if len(profiles) > 0 {
					names := make([]string, len(profiles))
					for i, p := range profiles {
						names[i] = p.Name
					}
					chosen, err := selectOne("Pick profile for this repo", names)
					if err == nil {
						profileName = chosen
					}
				}
			}
			if profileName != "" {
				if p, found := config.Get(profileName); found {
					cwd, _ := os.Getwd()
					_ = os.Chdir(dir)
					_ = applyLocal(p) // registers the repo too
					_ = os.Chdir(cwd)
				} else {
					warn("Profile %q not found — skipping", profileName)
				}
			}
			ok("Done. Run: cd %s", dir)
			return nil
		},
	}
}

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [file]",
		Short: "Export profiles.conf with tokens redacted, ready to share",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := "team-profiles.conf"
			if len(args) == 1 {
				out = args[0]
			}
			profiles, err := config.Load()
			if err != nil {
				return err
			}
			for i := range profiles {
				profiles[i].GHToken = ""
			}
			var b strings.Builder
			for _, p := range profiles {
				fmt.Fprintf(&b, "[%s]\nname=%s\nemail=%s\n", p.Name, p.GitName, p.Email)
				if p.SSHKey != "" {
					fmt.Fprintf(&b, "ssh_key=%s\n", p.SSHKey)
				}
				if p.GPGKey != "" {
					fmt.Fprintf(&b, "gpg_key=%s\n", p.GPGKey)
				}
				host := p.GHHost
				if host == "" {
					host = "github.com"
				}
				fmt.Fprintf(&b, "gh_host=%s\n\n", host)
			}
			if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
				return err
			}
			ok("Exported %d profile(s) (tokens redacted) to %s", len(profiles), out)
			return nil
		},
	}
}

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file-or-url>",
		Short: "Merge profiles from a shared file or URL — team presets (skips existing names)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imported, err := loadFrom(args[0])
			if err != nil {
				return err
			}
			existing, _ := config.Load()
			added := 0
			for _, p := range imported {
				if config.Exists(p.Name) {
					warn("Skipping existing profile %q", p.Name)
					continue
				}
				existing = append(existing, p)
				added++
			}
			if err := config.Save(existing); err != nil {
				return err
			}
			ok("Imported %d new profile(s).", added)
			return nil
		},
	}
}

// loadFrom parses a profiles.conf-formatted file at an arbitrary path or URL
// (config.Load only reads the default profiles.conf location, so this
// temporarily points GHSWITCH_CONFIG_DIR at a staged copy). This is the
// "team presets" flow — teams host a redacted profiles.conf (see `export`)
// and members `import` it by URL instead of passing files around.
//
// ponytail: no signature/checksum verification on fetched config — anyone
// with the URL can hand you profiles. Upgrade path: require an accompanying
// .sig file + a trusted-signers list if this needs to be tamper-proof.
func loadFrom(path string) ([]config.Profile, error) {
	var b []byte
	var err error
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		b, err = fetchURL(path)
	} else {
		b, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	stageDir, err := os.MkdirTemp("", "ghswitch-import")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stageDir)
	if err := os.WriteFile(filepath.Join(stageDir, "profiles.conf"), b, 0o600); err != nil {
		return nil, err
	}
	saved := os.Getenv("GHSWITCH_CONFIG_DIR")
	os.Setenv("GHSWITCH_CONFIG_DIR", stageDir)
	defer os.Setenv("GHSWITCH_CONFIG_DIR", saved)
	return config.Load()
}

func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func newReposCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repos [list|add|rm|clear]",
		Short: "Multi-repo registry",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sub := "list"
			if len(args) > 0 {
				sub = args[0]
			}
			switch sub {
			case "list", "":
				repos := readRepos()
				section("Registered repos")
				if len(repos) == 0 {
					info("(none) — repos are auto-registered on 'local'/'auto'")
					return nil
				}
				for _, r := range repos {
					info("%s", r)
				}
			case "add":
				cwd, _ := os.Getwd()
				path := cwd
				if len(args) > 1 {
					path = args[1]
				}
				repos := readRepos()
				for _, r := range repos {
					if r == path {
						info("Already registered.")
						return nil
					}
				}
				repos = append(repos, path)
				if err := writeRepos(repos); err != nil {
					return err
				}
				ok("Registered %s", path)
			case "rm":
				if len(args) < 2 {
					errf("Usage: ghswitch repos rm <path>")
					return nil
				}
				repos := readRepos()
				out := repos[:0]
				for _, r := range repos {
					if r != args[1] {
						out = append(out, r)
					}
				}
				if err := writeRepos(out); err != nil {
					return err
				}
				ok("Removed %s", args[1])
			case "clear":
				if err := writeRepos(nil); err != nil {
					return err
				}
				ok("Registry cleared.")
			default:
				errf("Unknown subcommand %q (list/add/rm/clear)", sub)
			}
			return nil
		},
	}
}

func newSyncAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync-all",
		Short: "Run sync across every repo in the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := readRepos()
			if len(repos) == 0 {
				warn("No repos registered. Run 'ghswitch local' or 'ghswitch auto' in a repo first.")
				return nil
			}
			section("Sync all (%d repos)", len(repos))
			var synced, skipped, failed []string
			cwd, _ := os.Getwd()
			for _, r := range repos {
				if _, err := os.Stat(r); err != nil {
					skipped = append(skipped, r+" (missing)")
					continue
				}
				_ = os.Chdir(r)
				if !gitutil.InRepo() {
					skipped = append(skipped, r+" (not a repo)")
					_ = os.Chdir(cwd)
					continue
				}
				if gitutil.DirtyCount() > 0 {
					skipped = append(skipped, r+" (dirty)")
					_ = os.Chdir(cwd)
					continue
				}
				cur := gitutil.CurrentBranch()
				var e error
				if gitutil.UpstreamOf("") == "" {
					_, e = gitutil.Run("push", "-u", "origin", cur)
				} else {
					if _, err := gitutil.Run("pull", "--rebase"); err != nil {
						e = err
					} else {
						_, e = gitutil.Run("push")
					}
				}
				if e != nil {
					failed = append(failed, r)
				} else {
					synced = append(synced, r)
				}
				_ = os.Chdir(cwd)
			}
			ok("Synced: %d", len(synced))
			if len(skipped) > 0 {
				warn("Skipped: %s", strings.Join(skipped, ", "))
			}
			if len(failed) > 0 {
				errf("Failed: %s", strings.Join(failed, ", "))
			}
			return nil
		},
	}
}

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats [days]",
		Short: "Per-profile commit count in the current repo (default 30 days)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			days := "30"
			if len(args) == 1 {
				days = args[0]
			}
			section("Commit stats — last %s days", days)
			out, _ := gitutil.Run("log", "--since="+days+".days", "--pretty=format:%ae")
			if out == "" {
				info("No commits in range.")
				return nil
			}
			counts := map[string]int{}
			for _, email := range strings.Split(out, "\n") {
				counts[email]++
			}
			profiles, _ := config.Load()
			emailToProfile := map[string]string{}
			for _, p := range profiles {
				emailToProfile[p.Email] = p.Name
			}
			for email, n := range counts {
				label := email
				if name, ok := emailToProfile[email]; ok {
					label = fmt.Sprintf("%s (%s)", name, email)
				}
				info("%-40s %d commit(s)", label, n)
			}
			return nil
		},
	}
}
