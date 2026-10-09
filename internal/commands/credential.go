package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/MadhuS-1605/ghswitch/internal/config"
	"github.com/MadhuS-1605/ghswitch/internal/gitutil"
	"github.com/spf13/cobra"
)

// installCredentialHelper points git's HTTPS credentials for host at
// `ghswitch __credential`, which answers with whichever profile is active
// in the calling shell — so HTTPS remotes follow per-terminal identity the
// same way SSH ones do. The empty first value resets inherited helpers
// (osxkeychain etc.) for this host, the same trick `gh auth setup-git` uses,
// so a cached credential for another account can't win.
func installCredentialHelper(host string) {
	bin := "ghswitch"
	if _, err := exec.LookPath(bin); err != nil {
		if exe, err := os.Executable(); err == nil {
			bin = exe
		}
	}
	key := fmt.Sprintf("credential.https://%s.helper", host)
	want := "!" + bin + " __credential"
	if cur, _ := gitutil.Run("config", "--global", "--get-all", key); cur == want {
		return
	}
	_, _ = gitutil.Run("config", "--global", "--replace-all", key, "")
	_, _ = gitutil.Run("config", "--global", "--add", key, want)
}

// readCredentialHost parses git's credential-helper input (key=value lines,
// blank-line terminated) and returns the host for an https request.
func readCredentialHost(r io.Reader) string {
	var protocol, host string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			break
		}
		switch k {
		case "protocol":
			protocol = v
		case "host":
			host = v
		}
	}
	if protocol != "https" {
		return ""
	}
	return host
}

// profileServes reports whether p (primary or a multi-host entry) is meant
// for host — a work profile must not hand its token to some other host.
func profileServes(p config.Profile, host string) bool {
	if ghHost(p) == host {
		return true
	}
	for _, h := range p.Hosts {
		if h.Host == host {
			return true
		}
	}
	return false
}

// __credential implements git's credential-helper protocol. It prints
// nothing (so git falls through to its prompt) unless the active profile
// serves the requested host and has a token — from the keychain, or from
// gh's own keyring for the profile's gh account.
func newHiddenCredentialCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "__credential <get|store|erase>",
		Hidden:        true,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "get" {
				return nil
			}
			host := readCredentialHost(os.Stdin)
			p, found := config.Get(activeProfileName())
			if host == "" || !found || !profileServes(p, host) {
				return nil
			}
			p = p.ForHost(host)
			token := p.Token()
			if token == "" && hasCmd("gh") {
				out, err := ghCmd("auth", "token", "--hostname", host, "--user", p.Login()).Output()
				if err == nil {
					token = strings.TrimSpace(string(out))
				}
			}
			if token == "" {
				return nil
			}
			fmt.Printf("username=%s\npassword=%s\n", p.Login(), token)
			return nil
		},
	}
}
