package tui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// pickPlain is the non-interactive fallback for piped/CI stdout, where
// bubbletea can't take over the terminal. Uses the same stderr-bound
// renderer as the arrow-key picker, so colors (or their absence under
// NO_COLOR) stay consistent between the two.
func pickPlain(title string, items []string) (string, error) {
	fmt.Fprintln(os.Stderr, "\n"+titleStyle.Render("▸ "+title))
	for i, item := range items {
		fmt.Fprintf(os.Stderr, "  %s %s\n", dimStyle.Render(fmt.Sprintf("[%d]", i+1)), item)
	}
	fmt.Fprint(os.Stderr, "  "+dimStyle.Render(fmt.Sprintf("Pick 1-%d (q to cancel): ", len(items))))
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	choice := strings.TrimSpace(line)
	if choice == "" || strings.EqualFold(choice, "q") {
		return "", fmt.Errorf("cancelled")
	}
	n, err := strconv.Atoi(choice)
	if err != nil || n < 1 || n > len(items) {
		return "", fmt.Errorf("invalid selection %q", choice)
	}
	return items[n-1], nil
}
