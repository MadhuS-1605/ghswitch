package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

func requireGH() bool {
	if hasCmd("gh") {
		return true
	}
	errf("gh CLI not installed.")
	info("Install: https://cli.github.com/")
	return false
}

func newPRCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pr",
		Short: "Open a pull request via gh (uses active profile's token)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() || !requireGH() {
				return nil
			}
			section("Create pull request")
			return runInherit("gh", "pr", "create")
		},
	}
}

func newPRsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "prs",
		Aliases: []string{"pr-list"},
		Short:   "Rich PR list with check status + review decision",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() || !requireGH() {
				return nil
			}
			section("Open pull requests")
			return runInherit("gh", "pr", "list",
				"--json", "number,title,author,reviewDecision,statusCheckRollup",
				"--template",
				`{{range .}}#{{.number}}	{{.title}}	@{{.author.login}}	{{.reviewDecision}}
{{end}}`)
		},
	}
}

func newPRCheckoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "pr-checkout [N]",
		Aliases: []string{"prco"},
		Short:   "Check out PR by number, or pick from menu",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() || !requireGH() {
				return nil
			}
			if len(args) == 1 {
				return runInherit("gh", "pr", "checkout", args[0])
			}
			list, e := exec.Command("gh", "pr", "list", "--json", "number,title", "--template",
				`{{range .}}#{{.number}} {{.title}}
{{end}}`).Output()
			if e != nil || len(list) == 0 {
				warn("No open PRs.")
				return nil
			}
			chosen, perr := selectOne("Checkout which PR?", splitLines(strings.TrimRight(string(list), "\n")))
			if perr != nil {
				info("Cancelled.")
				return nil
			}
			num := strings.TrimPrefix(strings.Fields(chosen)[0], "#")
			return runInherit("gh", "pr", "checkout", num)
		},
	}
}

func newPRStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pr-status",
		Short: "gh pr status for the current branch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() || !requireGH() {
				return nil
			}
			return runInherit("gh", "pr", "status")
		},
	}
}

func newReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Pick a PR + action (view/diff/approve/request-changes/comment/web)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() || !requireGH() {
				return nil
			}
			list, e := exec.Command("gh", "pr", "list", "--json", "number,title", "--template",
				`{{range .}}#{{.number}} {{.title}}
{{end}}`).Output()
			if e != nil || len(list) == 0 {
				warn("No open PRs.")
				return nil
			}
			chosenPR, err := selectOne("Review which PR?", splitLines(strings.TrimRight(string(list), "\n")))
			if err != nil {
				info("Cancelled.")
				return nil
			}
			num := strings.TrimPrefix(strings.Fields(chosenPR)[0], "#")

			action, err := selectOne("Action", []string{"view", "diff", "approve", "request-changes", "comment", "web"})
			if err != nil {
				info("Cancelled.")
				return nil
			}
			switch action {
			case "view", "diff", "web":
				return runInherit("gh", "pr", action, num)
			case "approve":
				return runInherit("gh", "pr", "review", num, "--approve")
			case "request-changes":
				msg := prompt("Reason for requesting changes:")
				return runInherit("gh", "pr", "review", num, "--request-changes", "--body", msg)
			case "comment":
				msg := prompt("Comment:")
				return runInherit("gh", "pr", "review", num, "--comment", "--body", msg)
			}
			return nil
		},
	}
}

func newIssuesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "issues",
		Short: "List open issues with number, title, labels, author",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() || !requireGH() {
				return nil
			}
			section("Open issues")
			return runInherit("gh", "issue", "list")
		},
	}
}

func newInboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "inbox",
		Aliases: []string{"notifications"},
		Short:   "Review-requested + mentioning-you + assigned, across all repos",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireGH() {
				return nil
			}
			section("Review requested")
			_ = runInherit("gh", "search", "prs", "--review-requested=@me", "--state=open")
			section("Assigned to you")
			_ = runInherit("gh", "search", "issues", "--assignee=@me", "--state=open")
			section("Mentions you")
			return runInherit("gh", "search", "issues", "--mentions=@me", "--state=open")
		},
	}
}

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open [target]",
		Short: "Open repo / branch / pr / issues / actions / settings in browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !requireRepo() {
				return nil
			}
			target := "repo"
			if len(args) == 1 {
				target = args[0]
			}
			remote := gitutil.ConfigGet("", "remote.origin.url")
			base := remoteToURL(remote)
			if base == "" {
				errf("No origin remote configured.")
				return nil
			}
			var url string
			switch target {
			case "repo", "":
				url = base
			case "branch":
				url = base + "/tree/" + gitutil.CurrentBranch()
			case "pr":
				url = base + "/pulls"
			case "issues":
				url = base + "/issues"
			case "actions":
				url = base + "/actions"
			case "settings":
				url = base + "/settings"
			default:
				errf("Unknown target %q (repo/branch/pr/issues/actions/settings)", target)
				return nil
			}
			return openURL(url)
		},
	}
}

// remoteToURL converts a git remote URL (ssh or https) to its https web URL,
// mirroring bash `_remote_to_url()` (ghswitch.sh:1968-1988).
func remoteToURL(remote string) string {
	remote = strings.TrimSuffix(remote, ".git")
	if strings.HasPrefix(remote, "git@") {
		// git@host:owner/repo -> https://host/owner/repo
		rest := strings.TrimPrefix(remote, "git@")
		host, path, ok := strings.Cut(rest, ":")
		if !ok {
			return ""
		}
		return fmt.Sprintf("https://%s/%s", host, path)
	}
	if strings.HasPrefix(remote, "https://") || strings.HasPrefix(remote, "http://") {
		return remote
	}
	return ""
}

func openURL(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "windows":
		c = exec.Command("cmd", "/c", "start", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	if err := c.Run(); err != nil {
		info("Open this URL: %s", url)
		return nil
	}
	ok("Opened %s", url)
	return nil
}
