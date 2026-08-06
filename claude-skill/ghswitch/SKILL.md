---
name: ghswitch
description: Use when the user asks to check, switch, or fix their GitHub/git identity (account, SSH key, GPG signing, gh auth) in the current terminal — e.g. "which GitHub account am I on", "switch to my work account", "why did that commit go out under the wrong email", "fix my ghswitch setup".
---

# ghswitch

`ghswitch` is a CLI that switches an entire GitHub identity (git author, SSH
key, GPG signing, `gh` CLI auth) per terminal session. Use it instead of
manually editing `~/.gitconfig` or SSH config whenever the user wants to
inspect or change which GitHub account is active.

## Check the current identity

```bash
ghswitch status
```

Reports session identity, global git config, local repo override (if any),
and `gh` auth state. Run this first when the user asks "which account am I
on" or a commit looks like it went out under the wrong name.

## Switch identity

```bash
ghswitch <profile>          # session-scoped, this terminal only
ghswitch local <profile>    # pin this one repo, independent of session
ghswitch <profile> --ttl 4h # auto-warns (doesn't silently revert) after 4h
ghswitch back               # revert to the previously active profile
```

List available profiles first with `ghswitch list` if the profile name is
unknown. Never guess a profile name — ask the user or list first.

## Fix a commit made under the wrong identity

```bash
ghswitch reauthor
```

Interactive: rewrite author/committer on a commit range to match the
currently active profile. Use this instead of manual `git commit --amend` or
`git rebase` when the user says a commit has the wrong name/email.

## Diagnose setup problems

```bash
ghswitch doctor
```

Audits SSH key permissions, `gh` auth, GPG keyring, token expiry, and
identity/profile mismatches. Run this when switching doesn't seem to take
effect, or before troubleshooting SSH/GPG errors manually.

## Guardrails

- `ghswitch` is session-scoped by design — it exports env vars into the
  current shell, not the same as editing `~/.gitconfig`. If the user asks
  why two terminals show different identities, that's expected behavior,
  not a bug.
- Don't hand-edit `~/.config/ghswitch/profiles.conf` unless asked; prefer
  `ghswitch add` / `ghswitch edit`.
- Requires `git@github.com:...` (SSH) remotes for SSH key switching to take
  effect — HTTPS remotes won't pick up the per-profile SSH key.
