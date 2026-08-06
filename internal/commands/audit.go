package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newAuditCmd shows the local switch log written by config.AppendAudit —
// a plain record of which profile was active when, not a sync mechanism.
func newAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "audit [N]",
		Short: "Show recent profile switches (default last 20)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := 20
			if len(args) == 1 {
				if v, err := strconv.Atoi(args[0]); err == nil {
					n = v
				}
			}
			lines, err := tailLines(config.AuditFile(), n)
			if err != nil {
				return fmt.Errorf("no switch history yet")
			}
			section("Last %d switches", len(lines))
			for _, line := range lines {
				parts := strings.SplitN(line, "\t", 3)
				if len(parts) != 3 {
					continue
				}
				fmt.Printf("  %s  %-16s %s\n", parts[0], parts[1], parts[2])
			}
			return nil
		},
	}
}

func tailLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var all []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		all = append(all, sc.Text())
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all, nil
}

// newGendocsCmd generates man pages and markdown reference docs from the
// live command tree — hidden plumbing for releases, not everyday use.
func newGendocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "__gendocs <dir>",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			header := &doc.GenManHeader{Title: "GHSWITCH", Section: "1"}
			if err := doc.GenManTree(cmd.Root(), header, dir); err != nil {
				return err
			}
			return doc.GenMarkdownTree(cmd.Root(), dir)
		},
	}
}
