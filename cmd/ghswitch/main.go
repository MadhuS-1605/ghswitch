// Command ghswitch is the Go port of ghswitch.sh — a GitHub-aware git
// toolkit and multi-account identity switcher.
package main

import (
	"fmt"
	"os"

	"github.com/MadhuS-1605/ghswitch/internal/commands"
)

// version is set via -ldflags "-X main.version=..." at release build time.
var version = "dev"

func main() {
	if err := commands.Root(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
