package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/tui"
)

// All human-facing output goes to stderr, never stdout — stdout is reserved
// for machine-readable output (session export lines) that `eval
// "$(ghswitch <profile>)"` consumes. Mixing the two would eval arbitrary
// status text as shell commands.

func section(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "\n▸ "+format+"\n", a...)
}
func ok(format string, a ...any)   { fmt.Fprintf(os.Stderr, "  ✓ "+format+"\n", a...) }
func info(format string, a ...any) { fmt.Fprintf(os.Stderr, "  ℹ %s\n", fmt.Sprintf(format, a...)) }
func warn(format string, a ...any) { fmt.Fprintf(os.Stderr, "  ⚠ %s\n", fmt.Sprintf(format, a...)) }
func errf(format string, a ...any) { fmt.Fprintf(os.Stderr, "  ✗ %s\n", fmt.Sprintf(format, a...)) }

var stdin = bufio.NewReader(os.Stdin)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func prompt(label string) string {
	fmt.Fprintf(os.Stderr, "  %s ", label)
	line, _ := stdin.ReadString('\n')
	return strings.TrimSpace(line)
}

// confirm prompts y/n, defaulting on empty input to whichever of Y/N is
// capitalized in the label (e.g. "(Y/n)" -> default yes, "(y/N)" -> default
// no) — the label is the single source of truth for which way Enter goes.
func confirm(label string) bool {
	def := strings.Contains(label, "(Y/")
	ans := strings.ToLower(prompt(label))
	if ans == "" {
		return def
	}
	return ans == "y" || ans == "yes"
}

// selectOne shows the arrow-key/mouse picker (internal/tui), falling back to
// a plain numbered prompt automatically when stdout isn't a real terminal.
func selectOne(title string, items []string) (string, error) {
	return selectOneMarked(title, items, "", "")
}

// selectOneMarked is selectOne with an "● active" marker on one entry,
// mirroring the bash arrow_menu's MENU_HIGHLIGHT behavior.
func selectOneMarked(title string, items []string, active, activeLabel string) (string, error) {
	return tui.Pick(title, items, active, activeLabel)
}
