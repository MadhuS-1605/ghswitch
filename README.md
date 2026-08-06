# ◉ ghswitch

> **No more committing with the wrong account.** A single Go binary that switches your entire GitHub identity — commit author, SSH key, GPG signing, `gh` auth — per terminal, instantly. Plus a full GitHub-aware git toolkit: branches, commits, sync, history, stashes, PRs, releases.

```
╔══════════════════════════════════════════╗
║  ◉ ghswitch  GitHub Account Switcher   ║
╚══════════════════════════════════════════╝

▸ Switch global profile
  ↑↓/j k or mouse · enter/click to select · q to quit

  › work                  ● active
    personal
    client-acme
```

---

## Why

Every "account switcher" that just flips `gh auth switch` is solving the
wrong problem — API tokens are one piece of your GitHub identity. Commit
author, SSH key, and GPG signing have to move together, or you end up
committing to the wrong repo under the wrong name. `ghswitch` switches all
of it at once.

And unlike a tool that mutates `~/.gitconfig`, `ghswitch` switches your
identity **per terminal** — open two shells, run two different accounts in
parallel, with zero risk of one clobbering the other.

## Features

- **Session isolation** — `ghswitch work` in one terminal and `ghswitch personal` in another are fully independent; each shell resolves its own identity via exported env vars, not a shared global file
- **One switch, everything moves** — git identity, SSH key, GPG signing, and `gh` CLI auth all swap together
- **Device-flow login** — `ghswitch login` authenticates through GitHub's OAuth device flow (same mechanism `gh auth login` uses), no manual token pasting required
- **`reauthor`** — already committed under the wrong identity? Rewrite author/committer on a commit range to match the active profile in one command
- **Arrow-key + mouse TUI** — every menu (profiles, branches, stashes, PRs) supports both
- **Per-repo pin** — `ghswitch local` applies a profile to one repo without touching global/session identity; `.ghswitch.yml` auto-declares it
- **Full git toolkit** — branch / commit / sync / push / pull / stash / log / diff, conventional-commit wizard, fixup + autosquash, interactive rebase, cherry-pick, reauthor
- **GitHub depth** — PRs, issues, review flow, inbox, release wizard, open-in-browser
- **Team features** — export/import profiles (tokens redacted), multi-repo registry + `sync-all`, per-profile commit stats
- **Diagnostics** — `doctor` audits SSH key perms, `gh` auth, GPG keyring, token expiry, and identity/profile mismatches
- **Single static binary** — macOS, Linux, Windows; no runtime dependencies beyond `git` (optional: `gh` for GitHub features, `gpg` for signed commits)

---

## Installation

**Homebrew**
```bash
brew install MadhuS-1605/ghswitch/ghswitch
```

**curl**
```bash
curl -fsSL https://raw.githubusercontent.com/MadhuS-1605/ghswitch/main/scripts/install.sh | bash
```

**go install**
```bash
go install github.com/MadhuS-1605/ghswitch/cmd/ghswitch@latest
```

**PowerShell (Windows)**
```powershell
irm https://raw.githubusercontent.com/MadhuS-1605/ghswitch/main/scripts/install.ps1 | iex
```

Then enable session isolation by adding this to your shell rc (`~/.zshrc`, `~/.bashrc`):
```bash
eval "$(ghswitch shellinit)"
```

> **Important:** Use `git@github.com:...` (SSH) remote URLs, not HTTPS, so ghswitch's SSH key management takes effect.

---

## Quick start

```bash
ghswitch login work        # device-flow login, saves token to profile "work"
ghswitch add                # or add a profile manually (name, email, SSH key, GPG key...)
ghswitch list                # see all profiles
ghswitch                     # launch the interactive switcher
ghswitch work                 # or switch directly by name
```

Open a second terminal and run `ghswitch personal` — the first terminal
keeps using `work`, unaffected.

---

## How session isolation works

`ghswitch <profile>` doesn't write to `~/.gitconfig`. It prints
`export GIT_AUTHOR_NAME=... GIT_AUTHOR_EMAIL=... GIT_SSH_COMMAND=...` to
stdout. The shell function installed by `shellinit` evals that output into
the *current* shell only — git reads those env vars in preference to
`user.name`/`user.email`/`core.sshCommand`, so each terminal gets an
independent identity with nothing shared. A best-effort mirror is also
written to `git config --global` as a fallback for tools that only read
gitconfig, but the env vars are the source of truth.

`ghswitch local` is unaffected by any of this — it writes `git config
--local`, which is already repo-scoped.

---

## Commands

### Identity

| Command | Description |
|---|---|
| `ghswitch` | Interactive switcher (arrow keys + mouse) |
| `ghswitch <name>` / `switch <name>` | Switch directly, session-scoped |
| `ghswitch <name> --ttl 4h` | Switch with a time-to-live; warns (doesn't silently revert) once it passes |
| `ghswitch back` | Switch back to whichever profile was active before the last switch |
| `ghswitch audit [N]` | Show recent switches (timestamp, profile, cwd) — default last 20 |
| `ghswitch completion <shell>` | Generate a shell-completion script (bash/zsh/fish/powershell) |
| `ghswitch login [profile]` | Device-flow GitHub login, saves token to a profile |
| `ghswitch add` | Add a profile (guided prompts) |
| `ghswitch list` / `ls` | List all profiles with active marker |
| `ghswitch local` | Apply a profile to the current repo only |
| `ghswitch auto` | Detect a matching profile (`.ghswitch.yml` or `user.email`) and apply locally |
| `ghswitch status` / `whoami` | Current identity (session + global + local) + `gh` auth |
| `ghswitch unset` / `reset` | Clear global git identity |
| `ghswitch edit` | Open `profiles.conf` in `$EDITOR` |
| `ghswitch delete` / `rm` | Remove a profile |
| `ghswitch shellinit` | Emit the shell function that enables session isolation |
| `ghswitch reauthor` | Rewrite author/committer on past commits to match the active profile |
| `ghswitch doctor` | Audit SSH keys, `gh` auth + token expiry, GPG keyring, identity mismatches |
| `ghswitch rotate [profile]` | Open token settings page, paste new token, re-auth `gh` |
| `ghswitch hooks-install` | Install a pre-commit guard that blocks commits with the wrong identity |

### Branches

| Command | Description |
|---|---|
| `ghswitch branch` / `br` | Arrow-key/mouse switcher over all branches |
| `ghswitch recent` | Switcher over the 10 most recently active branches |
| `ghswitch new [name]` | Create + checkout a new branch |
| `ghswitch rename [name]` / `mv` | Rename current branch |
| `ghswitch clean` | Delete local branches merged into the default branch |
| `ghswitch fresh` / `nuke` | `fetch --prune` + drop gone branches + `gc --auto` |

### Commits

| Command | Description |
|---|---|
| `ghswitch commit "msg"` / `c` | `git add -A` + commit |
| `ghswitch cc` | Conventional-commit wizard |
| `ghswitch fixup` | Pick a commit, create a fixup, auto-squash via rebase |
| `ghswitch amend` | Add staged + amend last commit |
| `ghswitch undo` | Soft reset HEAD~1, keeping changes staged |
| `ghswitch wip` / `wip-pop` | One-shot stash / restore |

### Sync & history

| Command | Description |
|---|---|
| `ghswitch sync` | Pull `--rebase` then push |
| `ghswitch pull` / `push` / `fetch` | Individual sync steps |
| `ghswitch log [N]` / `l` | Pretty graph log |
| `ghswitch diff` / `d` | Working-tree diff vs HEAD |
| `ghswitch who [file]` | Top contributors or `git blame` |
| `ghswitch revert` | Pick a commit, generate a revert |
| `ghswitch cherry-pick` / `cp` | Pick a branch + commit, cherry-pick onto current |
| `ghswitch rebase` | Interactive rebase wizard |
| `ghswitch conflict` / `merge` | Walk through unmerged files |

### Stash · tags · worktrees

| Command | Description |
|---|---|
| `ghswitch stash` | save / list / pop / apply / drop / clear |
| `ghswitch tag` | list / create / delete / push-all |
| `ghswitch worktree` / `wt` | list / add / remove |

### GitHub

| Command | Description |
|---|---|
| `ghswitch pr` | Open a pull request via `gh` |
| `ghswitch prs` / `pr-list` | PR list with review status |
| `ghswitch pr-checkout [N]` / `prco` | Check out a PR |
| `ghswitch pr-status` | Status for the current branch |
| `ghswitch review` | Pick a PR + action (view/diff/approve/request-changes/comment) |
| `ghswitch issues` | List open issues |
| `ghswitch inbox` / `notifications` | Review-requested + assigned + mentions, across repos |
| `ghswitch open [target]` | Open repo/branch/pr/issues/actions/settings in browser |
| `ghswitch release` | Bump → tag → push → `gh release create` wizard |

### Team & multi-repo

| Command | Description |
|---|---|
| `ghswitch bootstrap <url> [profile]` | Clone + apply a profile locally |
| `ghswitch export [file]` / `import <file>` | Share profiles with tokens redacted |
| `ghswitch repos [list\|add\|rm\|clear]` | Multi-repo registry (auto-tracked on `local`/`auto`) |
| `ghswitch sync-all` | Sync every registered repo |
| `ghswitch stats [days]` | Per-profile commit counts in the current repo |

### Per-repo preset (`.ghswitch.yml`)

```yaml
profile: client-acme
```

`ghswitch auto` applies that profile automatically — no email-matching
needed. The repo declares its identity instead of you remembering it.

---

## Config file

Located at `~/.config/ghswitch/profiles.conf`, created automatically on
first run:

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
```

All fields except `name` and `email` are optional. `gh_token` is populated
automatically by `ghswitch login`.

---

## SSH key setup

```bash
ssh-keygen -t ed25519 -C "you@company.com" -f ~/.ssh/id_ed25519_work
ssh-keygen -t ed25519 -C "you@gmail.com" -f ~/.ssh/id_ed25519_personal
```

Add each public key to the matching GitHub account under **Settings → SSH
and GPG keys**, then fix permissions if you see warnings:

```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519_work
chmod 644 ~/.ssh/id_ed25519_work.pub
```

---

## Compatibility

macOS, Linux, Windows — single static Go binary, no interpreter/runtime
dependency beyond `git` itself.

---

## Claude Code skill

A skill lives at [`claude-skill/ghswitch`](claude-skill/ghswitch) so Claude
Code and other AI coding agents can drive `ghswitch` directly (check active
identity, switch profiles, run `reauthor`/`doctor`) instead of hand-editing
git config. Copy it into your `.claude/skills/` directory to enable it.

## License

MIT
