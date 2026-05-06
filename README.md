# ◉ ghswitch

> **GitHub-aware git toolkit for the terminal** — switch identities, manage branches, commit, sync, stash, and open PRs from one tiny script. GitKraken-style UX, zero dependencies.

```
╔══════════════════════════════════════════╗
║  ◉ ghswitch  GitHub Account Switcher   ║
╚══════════════════════════════════════════╝

▸ Switch global profile
  ↑↓ / j k to navigate · Enter to select · q to quit

  ██  › work                  ● active
      personal
      client-acme
```

---

## Features

- **Arrow-key TUI** — navigate profiles, branches, commits, reflog, stashes like GitKraken
- **Profile switcher** — git config, SSH keys, GPG signing, and gh-CLI auth all swap together
- **Per-repo override** — apply a profile locally without touching global config
- **Daily git ops** — branch / commit / sync / push / pull / stash / log / diff in one tool
- **History surgery** — interactive `reflog`, `revert`, `cherry-pick`, `rebase` wizards
- **Smart sync** — `ghswitch sync` does pull --rebase + push, with auto `-u` on first push
- **Multi-repo** — `ghswitch repos` registry + `ghswitch sync-all` across every registered repo
- **GitHub depth** — PRs, issues, `review` flow, `inbox`, `open` in browser, `release` wizard
- **Repo dashboard** — `ghswitch repo` prints branch, ahead/behind, dirty count, active profile
- **Token hygiene** — `doctor` flags expiring fine-grained PATs; `rotate` walks you through replacement
- **Per-profile stats** — `ghswitch stats` shows commit counts per profile in the current repo
- **bash 3.2+ compatible** — macOS (default bash), Linux, Windows Git Bash / WSL
- **Zero dependencies** — just bash + git. Optional: `gh` for PRs, `gpg` for signed commits.

---

## Installation

```bash
# 1. Download
curl -O https://raw.githubusercontent.com/MadhuS-1605/Github-Profile-Switcher/main/ghswitch.sh

# 2. Make executable
chmod +x ghswitch.sh

# 3. Install globally
sudo cp ghswitch.sh /usr/local/bin/ghswitch

# 4. Verify
ghswitch help
```

**Linux (no sudo)**
```bash
mkdir -p ~/.local/bin
cp ghswitch.sh ~/.local/bin/ghswitch
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

**Windows (Git Bash)**
```bash
mkdir -p ~/bin
cp ghswitch.sh ~/bin/ghswitch
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

> **Important:** Always use `git@github.com:...` (SSH) remote URLs — not HTTPS — so ghswitch's key management takes effect.

---

## Quick Start

```bash
# Add your accounts
ghswitch add          # prompts: name, email, SSH key, GPG key, gh token, gh host

# See all profiles
ghswitch list

# Launch the TUI switcher
ghswitch

# Or switch directly by name
ghswitch work
ghswitch personal
```

---

## Commands

### Profiles

| Command | Description |
|---|---|
| `ghswitch` | Interactive arrow-key profile switcher |
| `ghswitch add` | Add a new profile (guided prompts) |
| `ghswitch list` / `ls` | List all profiles with active marker |
| `ghswitch local` | Apply profile to current repo only (`git --local`) |
| `ghswitch status` / `whoami` | Current git identity + gh-CLI auth |
| `ghswitch unset` / `reset` | Clear global git identity |
| `ghswitch edit` | Open `profiles.conf` in `$EDITOR` |
| `ghswitch delete` / `rm` | Remove a profile |
| `ghswitch <name>` | Switch directly, e.g. `ghswitch work` |

### Branches

| Command | Description |
|---|---|
| `ghswitch branch` / `br` | Arrow-key switcher over all branches |
| `ghswitch recent` | Switcher over the 10 most recently active branches |
| `ghswitch new [name]` | Create + checkout a new branch from current |
| `ghswitch rename [name]` / `mv` | Rename current branch (warns if pushed) |
| `ghswitch clean` | Delete local branches merged into the default branch |
| `ghswitch fresh` / `nuke` | `fetch --prune` + drop `gone` tracking branches + `gc --auto` |

### Commits

| Command | Description |
|---|---|
| `ghswitch commit "msg"` / `c` | `git add -A` + commit (prompts if no message) |
| `ghswitch cc` | Conventional commit wizard — pick `feat`/`fix`/etc., scope, subject, breaking flag |
| `ghswitch fixup` | Pick a past commit from a menu, create a fixup, auto-squash via interactive rebase |
| `ghswitch amend` | Add staged + amend last commit, message unchanged |
| `ghswitch undo` | Soft reset HEAD~1, keeping changes staged |
| `ghswitch wip` | One-shot stash with auto-message: `WIP @ <branch> · N file(s) · HH:MM` |
| `ghswitch wip-pop` | Pop the most recent stash |
| `ghswitch st` | **Interactive status** — lazygit-style picker (j/k navigate, s stage, u unstage, d diff, x discard, c commit) |

### Sync

| Command | Description |
|---|---|
| `ghswitch sync` | Pull `--rebase` then push (auto `-u` on first push) |
| `ghswitch pull` | Pull `--rebase` from upstream |
| `ghswitch push` | Push current branch (auto `-u` on first push) |
| `ghswitch fetch` | `git fetch --all --prune` |
| `ghswitch sync-all` | Run sync across every repo in the registry; reports synced/skipped/failed |

### History

| Command | Description |
|---|---|
| `ghswitch log [N]` / `l` | Pretty graph log, default 20 commits |
| `ghswitch diff` / `d` | Working-tree diff vs HEAD with stat |
| `ghswitch who [file]` | Top contributors (or `git blame` a file) |
| `ghswitch reflog` | Interactive reflog browser → show diff / checkout / reset / cherry-pick / branch-from |
| `ghswitch revert` | Pick a commit from the last 30, generate a revert commit |
| `ghswitch cherry-pick` / `cp` | Pick a source branch, then a commit; cherry-pick onto current branch |
| `ghswitch rebase` | Interactive rebase wizard: last N commits / onto branch / onto upstream |

### Stash · Tags · Worktrees

| Command | Description |
|---|---|
| `ghswitch stash` | Menu: save / list / pop / apply / drop / clear |
| `ghswitch tag` | Menu: list / create (annotated or lightweight) / delete (local + remote) / push-all |
| `ghswitch worktree` / `wt` | Menu: list / add (existing or new branch) / remove |

### GitHub

| Command | Description |
|---|---|
| `ghswitch pr` | Open a pull request via `gh` (uses active profile's token) |
| `ghswitch prs` / `pr-list` | Rich PR list with check status (✓/✗/⏳) + review decision |
| `ghswitch pr-checkout [N]` / `prco` | Check out PR by number, or pick from menu |
| `ghswitch pr-status` | `gh pr status` for the current branch |
| `ghswitch review` | Pick a PR + action (view / diff / approve / request-changes / comment / web) |
| `ghswitch issues` | List open issues with number, title, labels, author |
| `ghswitch inbox` / `notifications` | Review-requested + mentioning-you + assigned, across all repos |
| `ghswitch open [target]` | Open repo / `branch` / `pr` / `issues` / `actions` / `settings` in browser (honors `gh_host`) |
| `ghswitch release` | Wizard: bump (patch/minor/major/custom) → tag → push → `gh release create` with notes |
| `ghswitch conflict` / `merge` | Walk through unmerged files in `$EDITOR`, mark resolved |

### Repo & Diagnostics

| Command | Description |
|---|---|
| `ghswitch repo` / `info` | Dashboard: branch, upstream, ahead/behind, dirty count, profile |
| `ghswitch doctor` | Audit ssh-agent, key perms, gh auth, **token expiry + scopes**, GPG keyring, identity-vs-profile mismatch |
| `ghswitch auto` | Detect a matching profile (by `user.email`) and apply locally |
| `ghswitch stats [days]` | Per-profile commit count in the current repo (default 30 days) |
| `ghswitch repos [sub]` | Multi-repo registry — `list` (default) / `add` / `rm [path]` / `clear`. Auto-registers on `local` + `auto` |
| `ghswitch rotate [profile]` | Open settings page → paste new token → rewrite `profiles.conf` → re-auth `gh` |

### Team Setup

| Command | Description |
|---|---|
| `ghswitch bootstrap <url> [profile]` | `git clone` + apply chosen profile locally |
| `ghswitch hooks-install` | Install a pre-commit guard that blocks commits with the wrong identity |
| `ghswitch signed` | Wizard: generates an SSH or GPG signing key, uploads to GitHub via `gh`, configures `commit.gpgsign` |
| `ghswitch shellinit` | Emit a zsh/bash hook — `eval "$(ghswitch shellinit)"` in your rc to warn on `cd` |
| `ghswitch export [file]` | Export `profiles.conf` with tokens redacted, ready to share |
| `ghswitch import <file>` | Merge profiles from a shared file (skips existing names) |
| `ghswitch help` / `-h` | Show help |

### Per-repo preset (`.ghswitch.yml`)

Drop a `.ghswitch.yml` in the repo root and `ghswitch auto` will apply that profile automatically — no email-matching needed:

```yaml
profile: client-acme
```

Combined with `eval "$(ghswitch shellinit)"`, every `cd` into the repo verifies your `user.email` matches the declared profile and warns if it doesn't. The repo *declares* its identity instead of you remembering.

---

## Team Onboarding

Share profiles with your team without leaking tokens:

```bash
# Lead exports — tokens stripped
ghswitch export team-profiles.conf
git add team-profiles.conf && git commit -m "share team git profiles"

# Each teammate imports + adds their own token
ghswitch import team-profiles.conf
ghswitch edit            # paste your own gh_token under each profile
ghswitch doctor          # verify everything resolves
```

**Identity guardrails for every repo:**

```bash
# Fail commits that use the wrong email for the active profile
cd ~/work/acme-repo
ghswitch local           # set repo to "client-acme" profile
ghswitch hooks-install   # pre-commit guard blocks identity drift

# Or warn on every `cd` instead of failing on commit
echo 'eval "$(ghswitch shellinit)"' >> ~/.zshrc
```

**One-shot new-laptop / new-repo setup:**

```bash
ghswitch bootstrap git@github.com:acme/website.git client-acme
cd website   # already configured with the right identity
```

---

## Daily Workflow Examples

```bash
# Morning: refresh + see what's open
ghswitch fresh         # fetch --prune + drop gone branches + gc
ghswitch repo
ghswitch prs           # list PRs with check + review status

# Pick up a teammate's PR for review
ghswitch pr-checkout   # menu of open PRs

# Start a feature
ghswitch new feature/login
# ... edit files ...
ghswitch st            # interactive status: stage/unstage with arrows
ghswitch cc            # conventional commit wizard
ghswitch sync          # pulls + pushes in one shot
ghswitch pr            # open PR

# Forgot something — fix a past commit cleanly
# (edit the file you forgot)
ghswitch fixup         # pick the commit, auto-squash via rebase

# Context switch
ghswitch wip           # quick stash of WIP
ghswitch recent        # jump to a different recent branch
# ... do other work ...
ghswitch recent        # jump back
ghswitch wip-pop       # restore your work

# Switch profile on a different repo
cd ~/work/acme
ghswitch auto          # reads .ghswitch.yml or matches user.email

# Multi-repo: keep every repo in sync
ghswitch repos         # list all registered repos (auto-tracked from `local`/`auto`)
ghswitch sync-all      # pull --rebase + push across every clean repo

# Recover from a bad rebase / lost commit
ghswitch reflog        # pick the entry, then: show diff / checkout / branch-from / reset

# Cut a release with auto-generated notes
ghswitch release       # bump (patch/minor/major) → tag → push → gh release create

# Token about to expire? doctor warns; rotate replaces it
ghswitch doctor        # surfaces "expires in N days"
ghswitch rotate work   # opens settings page, prompts for new token, re-auths gh
```

---

## What It Manages

| Layer | Config Key | Effect |
|---|---|---|
| Git identity | `user.name` / `user.email` | Commit author, global or per-repo |
| SSH key | `core.sshCommand` | Forces specific key with `IdentitiesOnly=yes` |
| SSH agent | *(runtime)* | Adds key to agent if socket is active |
| GPG signing | `user.signingKey` + `commit.gpgSign` | Signed commits with your key |
| GitHub Token | `credential.helper` / `gh` CLI | Auth for Git over HTTPS and logs into `gh` |
| gh CLI | *(runtime check)* | Verifies `gh auth status` for the configured host |

---

## Config File

Located at `~/.config/ghswitch/profiles.conf`. Created automatically on first run.

```ini
[work]
name=Jane Doe
email=jane@company.com
ssh_key=~/.ssh/id_ed25519_work
gpg_key=ABCD1234
gh_token=ghp_...
gh_host=github.com

[personal]
name=Jane Doe
email=jane@gmail.com
ssh_key=~/.ssh/id_ed25519_personal
gh_host=github.com

[client-acme]
name=Jane Doe
email=jane@acme.io
ssh_key=~/.ssh/id_ed25519_acme
gh_host=github.com
```

All fields except `name` and `email` are optional.

---

## SSH Key Setup

Generate a separate key per account and add each public key to the respective GitHub account under **Settings → SSH and GPG keys**.

```bash
# Work
ssh-keygen -t ed25519 -C "you@company.com" -f ~/.ssh/id_ed25519_work

# Personal
ssh-keygen -t ed25519 -C "you@gmail.com" -f ~/.ssh/id_ed25519_personal

# Copy public key to clipboard (macOS)
cat ~/.ssh/id_ed25519_work.pub | pbcopy
```

Fix permissions if you see SSH warnings:
```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519_work      # private key
chmod 644 ~/.ssh/id_ed25519_work.pub  # public key
```

---

## Per-repo Override

Apply a profile to a single repo without changing your global identity:

```bash
cd ~/projects/client-website
ghswitch local          # pick "client-acme" from the menu
git config user.email   # confirms local override is active
```

---

## Troubleshooting

**Blank output / no prompts**
```bash
bash -x ghswitch add 2>&1 | head -40
# If line 2 shows "set -euo pipefail" — old version is still installed
sudo cp ghswitch.sh $(which ghswitch)
```

**403 on git push**
```bash
# Switch remote from HTTPS to SSH
git remote set-url origin git@github.com:USERNAME/REPO.git

# Clear stale macOS Keychain credentials
git credential-osxkeychain erase <<EOF
protocol=https
host=github.com
EOF
```

**Wrong key being used**
```bash
ghswitch status         # check active profile and sshCommand
ghswitch edit           # fix ssh_key path — must point to private key, not .pub
```

**gh CLI warning on switch**
```bash
gh auth login --hostname github.com
```

---

## Compatibility

| Platform | Shell | Status |
|---|---|---|
| macOS (12+) | bash 3.2 (system) | ✅ Tested |
| macOS (12+) | zsh (via bash shebang) | ✅ Works |
| Ubuntu / Debian | bash 5.x | ✅ Tested |
| Windows | Git Bash (MINGW) | ✅ Works |
| Windows | WSL2 | ✅ Works |

---

## License

MIT