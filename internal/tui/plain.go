package tui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// pickPlain is the non-interactive fallback for piped/CI stdout, where
// bubbletea can't take over the terminal.
func pickPlain(title string, items []string) (string, error) {
	fmt.Fprintf(os.Stderr, "\n▸ %s\n", title)
	for i, item := range items {
		fmt.Fprintf(os.Stderr, "  [%d] %s\n", i+1, item)
	}
	fmt.Fprintf(os.Stderr, "  Pick 1-%d (q to cancel): ", len(items))
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
