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

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorBlue   = "\033[34m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// color wraps s in an ANSI code, unless NO_COLOR is set (https://no-color.org).
func color(code, s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return code + s + colorReset
}

// quiet suppresses ok/info/warn (but not errf) — apply.go sets it around its
// per-field git/ssh/gh calls so a switch prints just two lines instead of
// one per field.
var quiet bool

func section(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "\n"+color(colorBold+colorBlue, "▸ "+fmt.Sprintf(format, a...))+"\n")
}
func ok(format string, a ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, "  "+color(colorGreen, "✓")+" %s\n", fmt.Sprintf(format, a...))
}
func info(format string, a ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, "  "+color(colorCyan, "ℹ")+" %s\n", fmt.Sprintf(format, a...))
}
func warn(format string, a ...any) {
	if quiet {
		return
	}
	fmt.Fprintf(os.Stderr, "  "+color(colorYellow, "⚠")+" %s\n", fmt.Sprintf(format, a...))
}
func errf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "  "+color(colorRed, "✗")+" %s\n", fmt.Sprintf(format, a...))
}

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
