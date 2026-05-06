#!/usr/bin/env bash
# ============================================================
#  ghswitch — GitHub Account Switcher
#  Compatible: bash 3.2+ (macOS default), Linux, Windows Git Bash/WSL
#  Manages: git config, SSH keys, GPG signing, gh CLI
# ============================================================

# NO set -e / set -u — breaks bash 3.2 and subshell return codes

# ── Colors ───────────────────────────────────────────────────
_init_colors() {
  if [ -t 1 ] && [ "${TERM:-dumb}" != "dumb" ]; then
    # Real ESC chars (ANSI-C quoting) so colors work in printf format
    # strings AND inside %s arguments. Plain '\033...' would only be
    # interpreted in the format string, not in arguments.
    BOLD=$'\033[1m';  DIM=$'\033[2m';    RESET=$'\033[0m'
    GREEN=$'\033[38;5;82m'; CYAN=$'\033[38;5;117m'
    YELLOW=$'\033[38;5;220m'; RED=$'\033[38;5;203m'
    MAGENTA=$'\033[38;5;183m'; WHITE=$'\033[97m'
    BG_SEL=$'\033[48;5;235m'
  else
    BOLD=''; DIM=''; RESET=''; GREEN=''; CYAN=''
    YELLOW=''; RED=''; MAGENTA=''; WHITE=''; BG_SEL=''
  fi
}
_init_colors

# ── Paths ────────────────────────────────────────────────────
CONFIG_DIR="${HOME}/.config/ghswitch"
CONFIG_FILE="${CONFIG_DIR}/profiles.conf"
ACTIVE_FILE="${CONFIG_DIR}/active"
REPOS_FILE="${CONFIG_DIR}/repos.list"

# ── OS detection ─────────────────────────────────────────────
case "$(uname -s 2>/dev/null)" in
  Darwin)             OS=macos   ;;
  Linux)              OS=linux   ;;
  MINGW*|MSYS*|CYGWIN*) OS=windows ;;
  *)                  OS=unknown ;;
esac

# ── Capability flags ─────────────────────────────────────────
HAS_GH=false; HAS_GPG=false; HAS_AGENT=false
command -v gh        >/dev/null 2>&1 && HAS_GH=true
command -v gpg       >/dev/null 2>&1 && HAS_GPG=true
command -v ssh-agent >/dev/null 2>&1 && HAS_AGENT=true

# ── UI helpers ───────────────────────────────────────────────
header() {
  printf '\n'
  printf "${BOLD}${CYAN}╔══════════════════════════════════════════╗${RESET}\n"
  printf "${BOLD}${CYAN}║${RESET}  ${WHITE}${BOLD}◉ ghswitch${RESET}  ${DIM}GitHub Account Switcher${RESET}   ${BOLD}${CYAN}║${RESET}\n"
  printf "${BOLD}${CYAN}╚══════════════════════════════════════════╝${RESET}\n\n"
}
section() { printf "\n${BOLD}${MAGENTA}▸ %s${RESET}\n" "$1"; }
ok()      { printf "  ${GREEN}✓${RESET} %s\n" "$1"; }
info()    { printf "  ${CYAN}ℹ${RESET} %s\n" "$1"; }
warn()    { printf "  ${YELLOW}⚠${RESET} %s\n" "$1"; }
err()     { printf "  ${RED}✗${RESET} %s\n" "$1"; }
ask()     { printf "  ${BOLD}%s${RESET} " "$1"; }

# ── Config init ──────────────────────────────────────────────
init_config() {
  mkdir -p "${CONFIG_DIR}"
  [ -f "${CONFIG_FILE}" ] || touch "${CONFIG_FILE}"
  [ -f "${ACTIVE_FILE}" ] || printf '' > "${ACTIVE_FILE}"
}

# ── Profile helpers ──────────────────────────────────────────
get_profiles() {
  grep '^\[' "${CONFIG_FILE}" 2>/dev/null | sed 's/^\[//;s/\]$//' || true
}

get_field() {
  awk -v prof="[$1]" -v fld="$2" '
    $0 == prof        { found=1; next }
    found && /^\[/    { exit }
    found             { if (index($0, fld "=") == 1) { sub(/^[^=]+=/, ""); print; exit } }
  ' "${CONFIG_FILE}"
}

profile_exists() {
  grep -q "^\[$1\]" "${CONFIG_FILE}" 2>/dev/null
}

get_active() {
  tr -d '[:space:]' < "${ACTIVE_FILE}" 2>/dev/null || true
}

# ── Arrow-key menu ───────────────────────────────────────────
# Usage: chosen=$(arrow_menu item1 item2 ...)
arrow_menu() {
  local count=$# selected=0 active i label suffix key rest cur label_text

  if [ -n "${MENU_HIGHLIGHT+x}" ]; then
    active="$MENU_HIGHLIGHT"
  else
    active=$(get_active)
  fi
  label_text="${MENU_HIGHLIGHT_LABEL:-active}"

  # Store items in indexed vars (bash 3.2 safe, no mapfile)
  i=0
  for item in "$@"; do
    eval "MENU_ITEM_${i}=\"\$item\""
    i=$((i + 1))
  done

  # Pre-select active profile
  for i in $(seq 0 $((count - 1))); do
    eval "cur=\$MENU_ITEM_${i}"
    if [ "$cur" = "$active" ]; then selected=$i; break; fi
  done

  tput civis 2>/dev/null || true
  trap 'tput cnorm 2>/dev/null || true' EXIT INT TERM HUP

  while true; do
    for i in $(seq 0 $((count - 1))); do
      eval "label=\$MENU_ITEM_${i}"
      suffix=''
      if [ -n "$active" ] && [ "$label" = "$active" ]; then
        suffix="  ${GREEN}${BOLD}● ${label_text}${RESET}"
      fi
      if [ "$i" -eq "$selected" ]; then
        printf "  ${BG_SEL}${CYAN}${BOLD}  › %-20s${RESET}" "$label"
        printf "%b\n" "$suffix"
      else
        printf "    ${DIM}%-22s${RESET}" "$label"
        printf "%b\n" "$suffix"
      fi
    done

    IFS= read -r -s -n1 key
    case "$key" in
      $'\x1b')
        IFS= read -r -s -n2 -t 0.1 rest 2>/dev/null || rest=''
        key="${key}${rest}"
        ;;
    esac

    case "$key" in
      $'\x1b[A'|k)
        [ "$selected" -gt 0 ] && selected=$((selected - 1)) ;;
      $'\x1b[B'|j)
        [ "$selected" -lt $((count - 1)) ] && selected=$((selected + 1)) ;;
      '')
        tput cnorm 2>/dev/null || true
        eval "printf '%s\n' \"\$MENU_ITEM_${selected}\""
        return 0 ;;
      q|$'\x03')
        tput cnorm 2>/dev/null || true
        return 1 ;;
    esac

    tput cuu "$count" 2>/dev/null || printf '\033[%dA' "$count"
  done
}

# ── Apply profile ────────────────────────────────────────────
apply_profile() {
  local profile="$1" scope="${2:-global}"
  local name email ssh_key gpg_key gh_host expanded_key

  if ! profile_exists "$profile"; then
    err "Profile '$profile' not found."
    return 1
  fi

  name=$(get_field "$profile" "name")
  email=$(get_field "$profile" "email")
  ssh_key=$(get_field "$profile" "ssh_key")
  gpg_key=$(get_field "$profile" "gpg_key")
  gh_token=$(get_field "$profile" "gh_token")
  gh_host=$(get_field "$profile" "gh_host")
  gh_host="${gh_host:-github.com}"

  section "Applying profile: ${BOLD}${profile}${RESET} (${scope})"

  if [ -n "$name" ]; then
    git config --"$scope" user.name "$name"
    ok "user.name  = $name"
  fi
  if [ -n "$email" ]; then
    git config --"$scope" user.email "$email"
    ok "user.email = $email"
  fi

  if [ -n "$ssh_key" ]; then
    expanded_key=$(eval echo "$ssh_key")
    if [ -f "$expanded_key" ]; then
      git config --"$scope" core.sshCommand "ssh -i $expanded_key -o IdentitiesOnly=yes"
      ok "SSH key   = $expanded_key"
      if $HAS_AGENT && [ -n "${SSH_AUTH_SOCK:-}" ]; then
        ssh-add "$expanded_key" 2>/dev/null && ok "SSH key added to agent" || true
      fi
    else
      warn "SSH key file not found: $expanded_key"
    fi
  fi

  if [ -n "$gpg_key" ]; then
    if $HAS_GPG && gpg --list-secret-keys "$gpg_key" >/dev/null 2>&1; then
      git config --"$scope" user.signingKey "$gpg_key"
      git config --"$scope" commit.gpgSign true
      ok "GPG key   = $gpg_key"
    else
      warn "GPG key not in keyring: $gpg_key"
    fi
  fi

  if [ -n "$gh_token" ]; then
    git config --"$scope" credential."https://${gh_host}".helper "!f() { echo \"username=git\"; echo \"password=${gh_token}\"; }; f"
    ok "git token = configured via credential.helper"
    if $HAS_GH; then
      if printf "%s" "$gh_token" | gh auth login --with-token --hostname "$gh_host" >/dev/null 2>&1; then
        ok "gh token  = authenticated for $gh_host"
        # `gh auth login --with-token` adds/refreshes the token but doesn't
        # change the *active* account when multiple are logged in for one
        # host. Flip active so subsequent gh commands use this profile.
        # Silently no-ops on gh < 2.40 (no `auth switch` subcommand).
        if [ -n "$name" ]; then
          if gh auth switch --hostname "$gh_host" --user "$name" >/dev/null 2>&1; then
            ok "gh active = $name"
          else
            warn "gh active = '$name' not switched (need gh ≥ 2.40 + matching user)"
          fi
        fi
      else
        warn "gh token  = failed to authenticate gh CLI"
      fi
    else
      info "gh CLI missing, token applied only to git HTTPS"
    fi
  elif $HAS_GH; then
    # If name is set, try to switch gh CLI active account
    if [ -n "$name" ] && gh auth switch --hostname "$gh_host" --user "$name" >/dev/null 2>&1; then
      ok "gh CLI    = switched to account $name"
    elif gh auth status --hostname "$gh_host" >/dev/null 2>&1; then
      ok "gh CLI    = authenticated on $gh_host"
    else
      warn "gh CLI not authed for $gh_host — run: gh auth login --hostname $gh_host"
    fi
  fi

  if [ "$scope" = "global" ]; then
    printf '%s' "$profile" > "${ACTIVE_FILE}"
    ok "Active profile → $profile"
  fi

  if [ "$scope" = "local" ]; then
    _register_repo 2>/dev/null || true
  fi

  printf "\n  ${GREEN}${BOLD}✓ Done!${RESET}\n"
}

# ── Add profile ──────────────────────────────────────────────
add_profile() {
  local pname pname_full pemail pssh pgpg phost apply_now

  section "Add new profile"

  ask "Profile name (e.g. work, personal):"; read -r pname
  pname=$(printf '%s' "$pname" | tr ' ' '-' | tr '[:upper:]' '[:lower:]')

  if [ -z "$pname" ]; then err "Name cannot be empty."; return 1; fi

  if profile_exists "$pname"; then
    warn "Profile '$pname' already exists. Use 'ghswitch edit' to modify."
    return 1
  fi

  ask "Github Username:";                                            read -r pname_full
  ask "Email:";                                                      read -r pemail
  ask "SSH key path (blank to skip, e.g. ~/.ssh/id_ed25519_work):";  read -r pssh
  ask "GPG key ID (blank to skip):";                                 read -r pgpg
  ask "GitHub Token (blank to skip):";                               read -r ptoken
  phost="github.com"

  printf '\n[%s]\nname=%s\nemail=%s\n' "$pname" "$pname_full" "$pemail" >> "${CONFIG_FILE}"
  [ -n "$pssh" ] && printf 'ssh_key=%s\n' "$pssh" >> "${CONFIG_FILE}"
  [ -n "$pgpg" ] && printf 'gpg_key=%s\n' "$pgpg" >> "${CONFIG_FILE}"
  [ -n "$ptoken" ] && printf 'gh_token=%s\n' "$ptoken" >> "${CONFIG_FILE}"
  printf 'gh_host=%s\n' "$phost" >> "${CONFIG_FILE}"

  ok "Profile '$pname' saved."

  ask "Apply this profile globally now? (y/N):"; read -r apply_now
  case "$apply_now" in [Yy]) apply_profile "$pname" "global" ;; esac
}

# ── List profiles ─────────────────────────────────────────────
list_profiles() {
  local active profiles p n e s g h mark

  active=$(get_active)
  profiles=$(get_profiles)

  section "Configured profiles"

  if [ -z "$profiles" ]; then
    warn "No profiles yet. Run: ghswitch add"
    return
  fi

  while IFS= read -r p; do
    [ -z "$p" ] && continue
    n=$(get_field "$p" "name");    e=$(get_field "$p" "email")
    s=$(get_field "$p" "ssh_key"); g=$(get_field "$p" "gpg_key")
    t=$(get_field "$p" "gh_token"); h=$(get_field "$p" "gh_host")
    mark=''
    [ "$p" = "$active" ] && mark="  ${GREEN}${BOLD}← active${RESET}"

    printf "\n  ${BOLD}${CYAN}[%s]${RESET}" "$p"; printf "%b\n" "$mark"
    printf "    ${DIM}name  :${RESET} %s\n" "$n"
    printf "    ${DIM}email :${RESET} %s\n" "$e"
    [ -n "$s" ] && printf "    ${DIM}ssh   :${RESET} %s\n" "$s"
    [ -n "$g" ] && printf "    ${DIM}gpg   :${RESET} %s\n" "$g"
    [ -n "$t" ] && printf "    ${DIM}token :${RESET} [configured]\n"
    [ -n "$h" ] && printf "    ${DIM}host  :${RESET} %s\n" "$h"
  done <<PROFILES
$profiles
PROFILES
  printf '\n'
}

# ── Status ───────────────────────────────────────────────────
show_status() {
  local gname gemail ggpg gssh active

  section "Current git identity (global)"
  gname=$(git config --global user.name       2>/dev/null || printf '(not set)')
  gemail=$(git config --global user.email     2>/dev/null || printf '(not set)')
  ggpg=$(git config --global user.signingKey  2>/dev/null || printf '(not set)')
  gssh=$(git config --global core.sshCommand  2>/dev/null || printf '(default)')
  active=$(get_active)

  printf "  ${DIM}Name     :${RESET} ${BOLD}%s${RESET}\n" "$gname"
  printf "  ${DIM}Email    :${RESET} ${BOLD}%s${RESET}\n" "$gemail"
  printf "  ${DIM}GPG Key  :${RESET} %s\n" "$ggpg"
  printf "  ${DIM}SSH Cmd  :${RESET} ${DIM}%s${RESET}\n" "$gssh"
  [ -n "$active" ] && printf "  ${DIM}ghswitch :${RESET} ${GREEN}${BOLD}%s${RESET} (active)\n" "$active"

  if in_repo; then
    local lname lemail lssh
    lname=$(git config --local user.name 2>/dev/null)
    lemail=$(git config --local user.email 2>/dev/null)
    lssh=$(git config --local core.sshCommand 2>/dev/null)
    if [ -n "$lname" ] || [ -n "$lemail" ] || [ -n "$lssh" ]; then
      section "Local git identity (repo override)"
      [ -n "$lname" ] && printf "  ${DIM}Name     :${RESET} ${BOLD}%s${RESET}\n" "$lname"
      [ -n "$lemail" ] && printf "  ${DIM}Email    :${RESET} ${BOLD}%s${RESET}\n" "$lemail"
      [ -n "$lssh" ]   && printf "  ${DIM}SSH Cmd  :${RESET} ${DIM}%s${RESET}\n" "$lssh"
    fi
  fi

  if $HAS_GH; then
    section "gh CLI status"
    gh auth status 2>&1 | sed 's/^/  /' || true
  fi
  printf '\n'
}

# ── Build args array from newline string (bash 3.2 safe) ─────
_profiles_to_args() {
  # populates global MENU_ARGS and MENU_COUNT
  MENU_COUNT=0
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    eval "MENU_ARGS_${MENU_COUNT}=\"\$line\""
    MENU_COUNT=$((MENU_COUNT + 1))
  done <<PEOF
$1
PEOF
}

_call_menu_from_args() {
  local i args_str=''
  local tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  arrow_menu "${tmp_args[@]}"
}

# ── Interactive switcher ──────────────────────────────────────
interactive_switch() {
  local profiles chosen

  profiles=$(get_profiles)
  if [ -z "$profiles" ]; then
    warn "No profiles yet."
    info "Run: ghswitch add"
    return
  fi

  section "Switch global profile"
  printf "  ${DIM}↑↓ / j k to navigate · Enter to select · q to quit${RESET}\n\n"

  _profiles_to_args "$profiles"
  chosen=$(_call_menu_from_args) || { printf '\n'; info "Cancelled."; return; }
  printf '\n'
  apply_profile "$chosen" "global"
}

# ── Local (per-repo) ─────────────────────────────────────────
apply_local() {
  local profiles chosen repo_root

  profiles=$(get_profiles)
  if [ -z "$profiles" ]; then
    warn "No profiles yet. Run: ghswitch add"
    return
  fi

  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    err "Not inside a git repository."
    return 1
  fi

  repo_root=$(git rev-parse --show-toplevel)
  section "Apply profile to this repo only"
  info "Repo: $repo_root"
  printf "  ${DIM}↑↓ / j k · Enter · q${RESET}\n\n"

  _profiles_to_args "$profiles"
  chosen=$(_call_menu_from_args) || { printf '\n'; info "Cancelled."; return; }
  printf '\n'
  apply_profile "$chosen" "local"
}

# ── Delete profile ────────────────────────────────────────────
delete_profile() {
  local profiles chosen tmp active confirm

  profiles=$(get_profiles)
  if [ -z "$profiles" ]; then
    warn "No profiles found."
    return
  fi

  section "Delete a profile"
  printf "  ${DIM}↑↓ / j k · Enter · q to cancel${RESET}\n\n"

  _profiles_to_args "$profiles"
  chosen=$(_call_menu_from_args) || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  ask "Delete '$chosen'? This cannot be undone. (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      tmp=$(mktemp)
      awk -v prof="[$chosen]" '
        $0 == prof     { skip=1; next }
        skip && /^\[/  { skip=0 }
        !skip          { print }
      ' "${CONFIG_FILE}" > "$tmp"
      mv "$tmp" "${CONFIG_FILE}"
      ok "Profile '$chosen' deleted."
      active=$(get_active)
      [ "$active" = "$chosen" ] && printf '' > "${ACTIVE_FILE}"
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Unset global identity ─────────────────────────────────────
unset_global() {
  local confirm
  section "Clear global git identity"
  ask "Remove global user.name, email, SSH command, GPG config? (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      git config --global --unset user.name       2>/dev/null && ok "Cleared user.name"       || true
      git config --global --unset user.email      2>/dev/null && ok "Cleared user.email"      || true
      git config --global --unset core.sshCommand 2>/dev/null && ok "Cleared core.sshCommand" || true
      git config --global --unset user.signingKey 2>/dev/null && ok "Cleared user.signingKey" || true
      git config --global --unset commit.gpgSign  2>/dev/null && ok "Cleared commit.gpgSign"  || true
      
      active=$(get_active)
      gh_host="github.com"
      [ -n "$active" ] && h=$(get_field "$active" "gh_host") && [ -n "$h" ] && gh_host="$h"
      git config --global --unset-all credential."https://${gh_host}".helper 2>/dev/null && ok "Cleared credential.helper for ${gh_host}" || true
      
      printf '' > "${ACTIVE_FILE}"
      ok "Global identity cleared."
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Unset local identity ──────────────────────────────────────
unset_local() {
  local confirm
  require_repo || return 1
  section "Clear local git identity (repo override)"
  ask "Remove local user.name, email, and SSH command? (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      git config --local --unset user.name       2>/dev/null && ok "Cleared local user.name"       || true
      git config --local --unset user.email      2>/dev/null && ok "Cleared local user.email"      || true
      git config --local --unset core.sshCommand 2>/dev/null && ok "Cleared local core.sshCommand" || true
      ok "Local identity cleared. Global settings will now apply."
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Edit config file ──────────────────────────────────────────
edit_config() {
  local ed="${EDITOR:-${VISUAL:-vi}}"
  info "Opening ${CONFIG_FILE} in ${ed}"
  "$ed" "${CONFIG_FILE}"
}

# ============================================================
#  Git toolkit — branch / commit / sync / history / stash / PR
# ============================================================

# ── Git context helpers ──────────────────────────────────────
in_repo()          { git rev-parse --is-inside-work-tree >/dev/null 2>&1; }
require_repo()     { in_repo || { err "Not inside a git repository."; return 1; }; }
current_branch()   { git rev-parse --abbrev-ref HEAD 2>/dev/null; }
repo_root()        { git rev-parse --show-toplevel 2>/dev/null; }
upstream_of()      { git rev-parse --abbrev-ref --symbolic-full-name "${1:-HEAD}@{u}" 2>/dev/null; }
dirty_count()      { git status --porcelain 2>/dev/null | grep -c . | tr -d ' '; }

ahead_behind() {
  # echoes "behind ahead" (number of commits) — empty if no upstream
  local up
  up=$(upstream_of) || return 1
  [ -z "$up" ] && return 1
  git rev-list --left-right --count "${up}...HEAD" 2>/dev/null
}

default_branch() {
  local d
  d=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')
  printf '%s' "${d:-main}"
}

# Run a generic arrow_menu without the profile "active" marker.
# Args: highlight_value highlight_label item1 item2 ...
generic_menu() {
  local hl="$1" lbl="$2"; shift 2
  MENU_HIGHLIGHT="$hl"
  MENU_HIGHLIGHT_LABEL="$lbl"
  arrow_menu "$@"
  local rc=$?
  unset MENU_HIGHLIGHT MENU_HIGHLIGHT_LABEL
  return $rc
}

# ── Branch operations ────────────────────────────────────────
list_branches() {
  git for-each-ref --format='%(refname:short)' refs/heads 2>/dev/null
}

switch_branch() {
  require_repo || return 1
  local branches chosen current i
  branches=$(list_branches)
  if [ -z "$branches" ]; then warn "No branches found."; return 1; fi

  current=$(current_branch)
  section "Switch branch"
  info "Current: ${BOLD}${current}${RESET}"
  printf "  ${DIM}↑↓ / j k · Enter · q to cancel${RESET}\n\n"

  _profiles_to_args "$branches"
  if [ "$MENU_COUNT" = "0" ]; then warn "No branches."; return 1; fi

  local tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "$current" "current" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  if [ "$chosen" = "$current" ]; then info "Already on '$chosen'."; return; fi
  if git checkout "$chosen"; then
    ok "Switched to ${BOLD}${chosen}${RESET}"
  else
    err "Checkout failed."
    return 1
  fi
}

new_branch() {
  require_repo || return 1
  local name="$1"
  if [ -z "$name" ]; then ask "New branch name:"; read -r name; fi
  if [ -z "$name" ]; then err "Name cannot be empty."; return 1; fi

  section "Create branch ${BOLD}${name}${RESET}"
  if git checkout -b "$name"; then
    ok "On ${BOLD}${name}${RESET} (branched from $(git rev-parse --short HEAD))"
  else
    err "Branch creation failed."
    return 1
  fi
}

rename_branch() {
  require_repo || return 1
  local newname="$1" cur
  cur=$(current_branch)
  if [ -z "$newname" ]; then ask "Rename '${cur}' to:"; read -r newname; fi
  if [ -z "$newname" ]; then err "Name cannot be empty."; return 1; fi

  section "Rename branch"
  if git branch -m "$newname"; then
    ok "Renamed ${cur} → ${BOLD}${newname}${RESET}"
    if git rev-parse "@{u}" >/dev/null 2>&1 || git rev-parse "origin/${cur}" >/dev/null 2>&1; then
      info "Was pushed — to update remote run:"
      info "  git push origin -u ${newname} && git push origin --delete ${cur}"
    fi
  else
    err "Rename failed."
    return 1
  fi
}

clean_branches() {
  require_repo || return 1
  local def merged confirm b
  def=$(default_branch)

  section "Delete merged branches"
  info "Comparing against: ${BOLD}${def}${RESET}"

  merged=$(git branch --merged "$def" 2>/dev/null \
    | grep -v '^\*' \
    | sed 's/^[[:space:]]*//' \
    | grep -v "^${def}\$" \
    | grep -v '^$')

  if [ -z "$merged" ]; then ok "No merged branches to clean."; return; fi

  printf "\n  ${BOLD}Will delete:${RESET}\n"
  printf '%s\n' "$merged" | sed 's/^/    /'
  printf '\n'

  ask "Delete these branches? (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      while IFS= read -r b; do
        [ -z "$b" ] && continue
        git branch -d "$b" && ok "Deleted $b" || warn "Skipped $b"
      done <<MERGED
$merged
MERGED
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Commit operations ────────────────────────────────────────
quick_commit() {
  require_repo || return 1
  local msg="$1" dirty
  dirty=$(dirty_count)
  if [ "$dirty" = "0" ]; then warn "Working tree clean — nothing to commit."; return; fi

  if [ -z "$msg" ]; then ask "Commit message:"; read -r msg; fi
  if [ -z "$msg" ]; then err "Message cannot be empty."; return 1; fi

  section "Quick commit"
  git add -A || { err "git add failed."; return 1; }
  if git commit -m "$msg"; then
    ok "Committed: $(git rev-parse --short HEAD)"
  else
    err "Commit failed."
    return 1
  fi
}

amend_commit() {
  require_repo || return 1
  section "Amend last commit"
  git add -A
  if git commit --amend --no-edit; then
    ok "Amended (message unchanged): $(git rev-parse --short HEAD)"
    if git rev-parse "@{u}" >/dev/null 2>&1; then
      warn "Already pushed — use: git push --force-with-lease"
    fi
  else
    err "Amend failed."
    return 1
  fi
}

undo_commit() {
  require_repo || return 1
  local last confirm
  last=$(git log -1 --pretty=format:'%h %s' 2>/dev/null)
  if [ -z "$last" ]; then err "No commits to undo."; return 1; fi

  section "Undo last commit (keep changes)"
  info "Last commit: ${BOLD}${last}${RESET}"
  ask "Soft reset HEAD~1? Files stay staged. (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      git reset --soft HEAD~1 && ok "Undone. Changes preserved in staging."
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Sync operations ──────────────────────────────────────────
sync_repo() {
  require_repo || return 1
  local cur up
  cur=$(current_branch)
  up=$(upstream_of)

  section "Sync ${BOLD}${cur}${RESET}"

  if [ -z "$up" ]; then
    info "No upstream — pushing with -u to origin/${cur}"
    git push -u origin "$cur" || { err "Push failed."; return 1; }
    ok "Pushed."
    return
  fi

  info "Pulling (rebase)..."
  if ! git pull --rebase; then
    err "Pull failed (conflicts?). Resolve, then run sync again."
    return 1
  fi

  info "Pushing..."
  if git push; then
    ok "Synced."
  else
    err "Push failed."
    return 1
  fi
}

pull_branch() {
  require_repo || return 1
  local cur up
  cur=$(current_branch)
  up=$(upstream_of)
  section "Pull ${BOLD}${cur}${RESET}"
  if [ -z "$up" ]; then
    warn "No upstream set."
    info "Set with: git branch --set-upstream-to=origin/${cur}"
    return 1
  fi
  git pull --rebase
}

push_branch() {
  require_repo || return 1
  local cur up
  cur=$(current_branch)
  up=$(upstream_of)
  section "Push ${BOLD}${cur}${RESET}"
  if [ -z "$up" ]; then
    info "First push — using -u origin/${cur}"
    git push -u origin "$cur"
  else
    git push
  fi
}

fetch_all() {
  require_repo || return 1
  section "Fetch all remotes (--prune)"
  git fetch --all --prune && ok "Fetch complete."
}

# ── History ──────────────────────────────────────────────────
pretty_log() {
  require_repo || return 1
  local n="${1:-20}"
  case "$n" in
    ''|*[!0-9]*) n=20 ;;
  esac
  section "Last ${n} commits"
  git log --graph --decorate --abbrev-commit --date=short \
    --pretty=format:"%C(yellow)%h%C(reset) %C(cyan)%ad%C(reset) %C(bold green)%an%C(reset) %s%C(magenta)%d%C(reset)" \
    -n "$n" 2>/dev/null
  printf '\n\n'
}

show_diff() {
  require_repo || return 1
  section "Working tree diff vs HEAD"
  if git diff --quiet HEAD 2>/dev/null && [ "$(dirty_count)" = "0" ]; then
    ok "No changes."
    return
  fi
  git -c color.ui=always diff --stat HEAD 2>/dev/null
  printf '\n'
  git -c color.ui=always diff HEAD 2>/dev/null
}

show_who() {
  require_repo || return 1
  local file="$1"
  if [ -z "$file" ]; then
    section "Top contributors"
    git shortlog -sn --no-merges HEAD 2>/dev/null | head -20 | sed 's/^/  /'
    return
  fi
  if [ ! -f "$file" ]; then err "File not found: $file"; return 1; fi
  section "Blame: ${file}"
  git -c color.ui=always blame --date=short -- "$file" 2>/dev/null
}

# ── Stash menu ───────────────────────────────────────────────
stash_menu() {
  require_repo || return 1
  local action stashes chosen msg confirm ref tmp_args i

  section "Stash"
  printf "  ${DIM}↑↓ / j k · Enter · q${RESET}\n\n"

  _profiles_to_args "$(printf 'save\nlist\npop\napply\ndrop\nclear\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  action=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$action" in
    save)
      ask "Stash message (Enter for none):"; read -r msg
      if [ -n "$msg" ]; then
        git stash push -u -m "$msg" && ok "Stashed."
      else
        git stash push -u && ok "Stashed."
      fi
      ;;
    list)
      stashes=$(git stash list 2>/dev/null)
      if [ -z "$stashes" ]; then info "No stashes."; else printf '%s\n' "$stashes"; fi
      ;;
    pop|apply|drop)
      stashes=$(git stash list --pretty=format:'%gd %s' 2>/dev/null)
      if [ -z "$stashes" ]; then warn "No stashes."; return; fi
      _profiles_to_args "$stashes"
      tmp_args=()
      for i in $(seq 0 $((MENU_COUNT - 1))); do
        eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
      done
      chosen=$(generic_menu "" "" "${tmp_args[@]}") || { info "Cancelled."; return; }
      ref=$(printf '%s' "$chosen" | awk '{print $1}')
      git stash "$action" "$ref"
      ;;
    clear)
      ask "Drop ALL stashes? (y/N):"; read -r confirm
      case "$confirm" in
        [Yy]) git stash clear && ok "All stashes cleared." ;;
        *)    info "Cancelled." ;;
      esac
      ;;
  esac
}

# ── GitHub (gh CLI wrappers) ─────────────────────────────────
require_gh() {
  if ! $HAS_GH; then
    err "gh CLI not installed."
    info "Install: https://cli.github.com/"
    return 1
  fi
}

create_pr() {
  require_repo || return 1
  require_gh   || return 1
  section "Create pull request"
  gh pr create
}

list_prs() {
  require_repo || return 1
  require_gh   || return 1
  section "Open pull requests"
  gh pr list
}

# ── Repo dashboard ───────────────────────────────────────────
repo_info() {
  require_repo || return 1
  local cur up ab dirty remote root active behind ahead

  cur=$(current_branch)
  up=$(upstream_of)
  ab=$(ahead_behind)
  dirty=$(dirty_count)
  remote=$(git config --get remote.origin.url 2>/dev/null)
  root=$(repo_root)
  active=$(get_active)

  section "Repo dashboard"
  printf "  ${DIM}Path     :${RESET} %s\n" "$root"
  printf "  ${DIM}Branch   :${RESET} ${BOLD}${CYAN}%s${RESET}\n" "$cur"
  if [ -n "$up" ]; then
    printf "  ${DIM}Upstream :${RESET} %s\n" "$up"
    if [ -n "$ab" ]; then
      behind=$(printf '%s' "$ab" | awk '{print $1}')
      ahead=$(printf '%s' "$ab" | awk '{print $2}')
      printf "  ${DIM}Sync     :${RESET} ${GREEN}%s ahead${RESET} · ${YELLOW}%s behind${RESET}\n" "$ahead" "$behind"
    fi
  else
    printf "  ${DIM}Upstream :${RESET} ${YELLOW}(none)${RESET}\n"
  fi
  printf "  ${DIM}Remote   :${RESET} %s\n" "${remote:-(none)}"
  if [ "$dirty" = "0" ]; then
    printf "  ${DIM}Status   :${RESET} ${GREEN}clean${RESET}\n"
  else
    printf "  ${DIM}Status   :${RESET} ${YELLOW}%s changed file(s)${RESET}\n" "$dirty"
  fi
  [ -n "$active" ] && printf "  ${DIM}Profile  :${RESET} ${GREEN}${BOLD}%s${RESET}\n" "$active"
  printf '\n'
}

# ── Doctor ───────────────────────────────────────────────────
doctor() {
  local active ssh_key expanded perms gh_host gpg_key cur_email prof_email remote
  local issues=0

  section "ghswitch doctor"

  # Tools
  if command -v git >/dev/null 2>&1; then
    ok "git installed: $(git --version | awk '{print $3}')"
  else
    err "git missing"; issues=$((issues + 1))
  fi
  if $HAS_GH; then
    ok "gh CLI installed: $(gh --version 2>/dev/null | head -1 | awk '{print $3}')"
  else
    warn "gh CLI missing — pr / prs commands will not work"
  fi
  if $HAS_GPG; then
    ok "gpg installed"
  else
    warn "gpg missing — signed commits disabled"
  fi
  if $HAS_AGENT && [ -n "${SSH_AUTH_SOCK:-}" ]; then
    ok "ssh-agent active (socket: $SSH_AUTH_SOCK)"
  else
    warn "ssh-agent not running or SSH_AUTH_SOCK not set"
  fi

  # Active profile
  active=$(get_active)
  if [ -z "$active" ]; then
    section "Active profile"
    info "(none) — run 'ghswitch' to pick one"
  else
    section "Active profile: ${BOLD}${active}${RESET}"

    ssh_key=$(get_field "$active" "ssh_key")
    if [ -n "$ssh_key" ]; then
      expanded=$(eval echo "$ssh_key")
      if [ -f "$expanded" ]; then
        ok "SSH key exists: $expanded"
        perms=$(stat -f '%A' "$expanded" 2>/dev/null || stat -c '%a' "$expanded" 2>/dev/null)
        if [ "$perms" = "600" ]; then
          ok "SSH key permissions: 600"
        else
          err "SSH key permissions: $perms (should be 600)"
          info "  fix: chmod 600 $expanded"
          issues=$((issues + 1))
        fi
        if $HAS_AGENT && [ -n "${SSH_AUTH_SOCK:-}" ]; then
          if ssh-add -l 2>/dev/null | grep -q "$expanded"; then
            ok "SSH key loaded in agent"
          else
            warn "SSH key not in agent"
            info "  fix: ssh-add $expanded"
          fi
        fi
      else
        err "SSH key file missing: $expanded"
        issues=$((issues + 1))
      fi
    fi

    gh_host=$(get_field "$active" "gh_host"); gh_host="${gh_host:-github.com}"
    if $HAS_GH; then
      if gh auth status --hostname "$gh_host" >/dev/null 2>&1; then
        ok "gh authenticated for $gh_host"
        local exp_line exp_epoch now_epoch days_left scopes_line
        exp_line=$(GH_HOST="$gh_host" gh api -i /user 2>/dev/null \
                   | awk 'BEGIN{IGNORECASE=1} /^github-authentication-token-expiration:/ { sub(/^[^:]+:[[:space:]]*/, ""); print; exit }' \
                   | tr -d '\r\n')
        if [ -n "$exp_line" ]; then
          exp_epoch=$(date -j -f "%Y-%m-%d %H:%M:%S %Z" "$exp_line" +%s 2>/dev/null \
                    || date -j -f "%Y-%m-%d %H:%M:%S UTC" "$exp_line" +%s 2>/dev/null \
                    || date -d "$exp_line" +%s 2>/dev/null)
          now_epoch=$(date +%s)
          if [ -n "$exp_epoch" ]; then
            days_left=$(( (exp_epoch - now_epoch) / 86400 ))
            if [ "$days_left" -lt 0 ]; then
              err "gh token EXPIRED ($exp_line)"
              info "  fix: ghswitch rotate $active"
              issues=$((issues + 1))
            elif [ "$days_left" -lt 7 ]; then
              err "gh token expires in $days_left day(s) — $exp_line"
              info "  fix: ghswitch rotate $active"
              issues=$((issues + 1))
            elif [ "$days_left" -lt 30 ]; then
              warn "gh token expires in $days_left days ($exp_line)"
            else
              ok "gh token valid · $days_left days remaining"
            fi
          fi
        else
          info "gh token has no expiry header (classic PAT or OAuth)"
        fi
        scopes_line=$(GH_HOST="$gh_host" gh api -i /user 2>/dev/null \
                      | awk 'BEGIN{IGNORECASE=1} /^x-oauth-scopes:/ { sub(/^[^:]+:[[:space:]]*/, ""); print; exit }' \
                      | tr -d '\r\n')
        [ -n "$scopes_line" ] && info "scopes: $scopes_line"
      else
        err "gh NOT authenticated for $gh_host"
        info "  fix: gh auth login --hostname $gh_host"
        issues=$((issues + 1))
      fi
    fi

    gpg_key=$(get_field "$active" "gpg_key")
    if [ -n "$gpg_key" ]; then
      if $HAS_GPG && gpg --list-secret-keys "$gpg_key" >/dev/null 2>&1; then
        ok "GPG key in keyring: $gpg_key"
      else
        err "GPG key NOT in keyring: $gpg_key"
        issues=$((issues + 1))
      fi
    fi
  fi

  # Repo-specific
  if in_repo; then
    section "Current repo"
    remote=$(git config --get remote.origin.url 2>/dev/null)
    cur_email=$(git config user.email 2>/dev/null)
    info "Remote     : ${remote:-(none)}"
    info "user.email : ${cur_email:-(not set)}"
    if [ -n "$active" ]; then
      prof_email=$(get_field "$active" "email")
      if [ -n "$prof_email" ] && [ "$cur_email" != "$prof_email" ]; then
        warn "user.email mismatch: repo='$cur_email' vs profile='$prof_email'"
        info "  fix: ghswitch local   (or 'ghswitch auto' to detect)"
        issues=$((issues + 1))
      fi
    fi
  fi

  printf '\n'
  if [ "$issues" = "0" ]; then
    printf "  ${GREEN}${BOLD}✓ All checks passed${RESET}\n\n"
  else
    printf "  ${YELLOW}${BOLD}⚠ %d issue(s) found${RESET}\n\n" "$issues"
  fi
}

# ── Auto-detect profile ──────────────────────────────────────
# Lookup order:
#   1. .ghswitch.yml in repo root (key: profile)
#   2. user.email match against profiles
auto_profile() {
  require_repo || return 1
  local cur_email profiles best p p_email remote confirm preset_file preset root
  local quiet=0
  case "${1:-}" in --quiet|-q) quiet=1 ;; esac

  root=$(repo_root)
  preset_file="${root}/.ghswitch.yml"
  preset=""
  if [ -f "$preset_file" ]; then
    preset=$(awk -F: '/^[[:space:]]*profile[[:space:]]*:/ {
      sub(/^[^:]+:[[:space:]]*/,"")
      gsub(/^["'\'']|["'\''][[:space:]]*$/,"")
      print
      exit
    }' "$preset_file" 2>/dev/null)
  fi

  cur_email=$(git config user.email 2>/dev/null)
  remote=$(git config --get remote.origin.url 2>/dev/null)

  if [ "$quiet" = "0" ]; then
    section "Auto-detect profile"
    [ -n "$preset" ] && info "preset     : .ghswitch.yml → ${preset}"
    info "user.email : ${cur_email:-(not set)}"
    info "remote     : ${remote:-(none)}"
  fi

  if [ -n "$preset" ] && profile_exists "$preset"; then
    best="$preset"
  else
    [ -n "$preset" ] && [ "$quiet" = "0" ] && warn "Preset profile '$preset' not found locally"
    best=""
    profiles=$(get_profiles)
    while IFS= read -r p; do
      [ -z "$p" ] && continue
      p_email=$(get_field "$p" "email")
      if [ -n "$cur_email" ] && [ "$p_email" = "$cur_email" ]; then
        best="$p"; break
      fi
    done <<PEOF
$profiles
PEOF
  fi

  if [ -z "$best" ]; then
    [ "$quiet" = "0" ] && {
      warn "No matching profile (preset or user.email)"
      info "Use 'ghswitch local' to pick one manually"
    }
    return 1
  fi

  if [ "$quiet" = "1" ]; then
    apply_profile "$best" "local" >/dev/null
    ok "Auto-applied: $best"
    return 0
  fi

  ok "Match: ${BOLD}${best}${RESET}"
  ask "Apply '$best' to this repo? (Y/n):"; read -r confirm
  case "$confirm" in
    [Nn]) info "Cancelled." ;;
    *)    apply_profile "$best" "local" ;;
  esac
}

# ── Bootstrap (clone + apply profile) ────────────────────────
bootstrap_repo() {
  local url="$1" profile="$2" dir profiles tmp_args i

  if [ -z "$url" ]; then
    err "Usage: ghswitch bootstrap <repo-url> [profile]"
    return 1
  fi

  section "Bootstrap repo"
  info "URL: $url"

  dir=$(basename "$url" .git)
  if [ -e "$dir" ]; then
    err "Directory '$dir' already exists."
    return 1
  fi

  if ! git clone "$url" "$dir"; then
    err "Clone failed."
    return 1
  fi
  ok "Cloned into ./$dir"

  if [ -z "$profile" ]; then
    profiles=$(get_profiles)
    if [ -n "$profiles" ]; then
      printf "\n"
      section "Pick profile for this repo"
      printf "  ${DIM}↑↓ / j k · Enter · q to skip${RESET}\n\n"
      _profiles_to_args "$profiles"
      tmp_args=()
      for i in $(seq 0 $((MENU_COUNT - 1))); do
        eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
      done
      profile=$(arrow_menu "${tmp_args[@]}") || profile=""
      printf '\n'
    fi
  fi

  if [ -n "$profile" ]; then
    if profile_exists "$profile"; then
      ( cd "$dir" && apply_profile "$profile" "local" )
    else
      warn "Profile '$profile' not found — skipping"
    fi
  fi

  printf "\n  ${BOLD}${GREEN}✓ Done.${RESET}  Run: ${CYAN}cd $dir${RESET}\n\n"
}

# ── Conventional commit helper ───────────────────────────────
conventional_commit() {
  require_repo || return 1
  local types type scope desc msg confirm dirty tmp_args i breaking

  dirty=$(dirty_count)
  if [ "$dirty" = "0" ]; then warn "Working tree clean — nothing to commit."; return; fi

  section "Conventional commit"
  printf "  ${DIM}Pick type, then enter scope and subject${RESET}\n\n"

  types="feat
fix
chore
docs
refactor
test
style
perf
build
ci
revert"

  _profiles_to_args "$types"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  type=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  ask "Scope (optional, e.g. auth, api):"; read -r scope
  ask "Subject (imperative, no period):"; read -r desc
  ask "Breaking change? (y/N):";          read -r breaking

  if [ -z "$desc" ]; then err "Subject required."; return 1; fi

  local bang=""
  case "$breaking" in [Yy]) bang="!" ;; esac

  if [ -n "$scope" ]; then
    msg="${type}(${scope})${bang}: ${desc}"
  else
    msg="${type}${bang}: ${desc}"
  fi

  printf "\n  ${DIM}Message:${RESET} ${BOLD}%s${RESET}\n\n" "$msg"
  ask "Stage all + commit? (Y/n):"; read -r confirm
  case "$confirm" in
    [Nn]) info "Cancelled." ;;
    *)
      git add -A && git commit -m "$msg" && ok "Committed: $(git rev-parse --short HEAD)"
      ;;
  esac
}

# ── Hooks installer ──────────────────────────────────────────
install_hooks() {
  require_repo || return 1
  local hooks_dir hook_path confirm
  hooks_dir=$(git rev-parse --git-path hooks 2>/dev/null)
  hook_path="${hooks_dir}/pre-commit"

  section "Install pre-commit guard"
  info "Hook path: $hook_path"

  if [ -f "$hook_path" ] && ! grep -q 'ghswitch-guard' "$hook_path" 2>/dev/null; then
    warn "Existing pre-commit hook found (not from ghswitch)"
    ask "Overwrite? (y/N):"; read -r confirm
    case "$confirm" in [Yy]) ;; *) info "Cancelled."; return ;; esac
  fi

  cat > "$hook_path" <<'HOOK'
#!/usr/bin/env bash
# ghswitch-guard: blocks commit if user.email mismatches active profile
ACTIVE_FILE="${HOME}/.config/ghswitch/active"
[ -f "$ACTIVE_FILE" ] || exit 0
ACTIVE=$(tr -d '[:space:]' < "$ACTIVE_FILE" 2>/dev/null)
[ -z "$ACTIVE" ] && exit 0
CONFIG="${HOME}/.config/ghswitch/profiles.conf"
[ -f "$CONFIG" ] || exit 0
EXPECTED=$(awk -v p="[$ACTIVE]" '$0==p{f=1;next} f && /^\[/{exit} f && /^email=/{sub(/^email=/,"");print;exit}' "$CONFIG")
CURRENT=$(git config user.email 2>/dev/null)
if [ -n "$EXPECTED" ] && [ "$EXPECTED" != "$CURRENT" ]; then
  printf '\033[31m✗\033[0m ghswitch-guard: user.email is "%s" but active profile "%s" expects "%s"\n' "$CURRENT" "$ACTIVE" "$EXPECTED" >&2
  printf '   fix: ghswitch local   (or ghswitch auto)\n' >&2
  exit 1
fi
exit 0
HOOK
  chmod +x "$hook_path"
  ok "Installed pre-commit guard"
  info "Bypass with --no-verify if you really need to (not recommended)"
}

# ── Conflict resolver ────────────────────────────────────────
conflict_helper() {
  require_repo || return 1
  local files file ed action confirm remaining

  files=$(git diff --name-only --diff-filter=U 2>/dev/null)
  if [ -z "$files" ]; then ok "No merge conflicts."; return; fi

  ed="${EDITOR:-${VISUAL:-vi}}"
  section "Merge conflicts"
  printf '%s\n' "$files" | sed 's/^/  ✗ /'
  printf "\n  ${DIM}Editor: ${ed}${RESET}\n\n"

  while IFS= read -r file; do
    [ -z "$file" ] && continue
    printf "  ${BOLD}%s${RESET}\n" "$file"
    ask "  Open in editor? (Y / s=skip / q=quit):"; read -r action
    case "$action" in
      [Ss]) info "Skipped."; continue ;;
      [Qq]) info "Quit."; return ;;
      *)    "$ed" "$file" ;;
    esac
    ask "  Mark '$file' resolved? (y/N):"; read -r confirm
    case "$confirm" in
      [Yy]) git add "$file" && ok "Marked resolved." ;;
      *)    info "Left unresolved." ;;
    esac
  done <<CEOF
$files
CEOF

  remaining=$(git diff --name-only --diff-filter=U 2>/dev/null)
  if [ -z "$remaining" ]; then
    ok "All conflicts resolved."
    info "Now run: git commit   (or git rebase --continue)"
  else
    printf '\n'
    warn "Still unresolved:"
    printf '%s\n' "$remaining" | sed 's/^/    /'
  fi
}

# ── Tag menu ─────────────────────────────────────────────────
tag_menu() {
  require_repo || return 1
  local action name msg confirm tags chosen tmp_args i

  section "Tags"
  printf "  ${DIM}↑↓ / j k · Enter · q${RESET}\n\n"

  _profiles_to_args "$(printf 'list\ncreate\ndelete\npush-all\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  action=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$action" in
    list)
      tags=$(git tag --sort=-creatordate 2>/dev/null | head -50)
      if [ -z "$tags" ]; then info "No tags."; else printf '%s\n' "$tags" | sed 's/^/  /'; fi
      ;;
    create)
      ask "Tag name (e.g. v1.0.0):"; read -r name
      [ -z "$name" ] && { err "Name required."; return 1; }
      ask "Annotation message (Enter for lightweight tag):"; read -r msg
      if [ -n "$msg" ]; then
        git tag -a "$name" -m "$msg" && ok "Annotated tag created: $name"
      else
        git tag "$name" && ok "Lightweight tag created: $name"
      fi
      ask "Push '$name' to origin now? (y/N):"; read -r confirm
      case "$confirm" in [Yy]) git push origin "$name" && ok "Pushed." ;; esac
      ;;
    delete)
      tags=$(git tag --sort=-creatordate 2>/dev/null)
      if [ -z "$tags" ]; then warn "No tags."; return; fi
      _profiles_to_args "$tags"
      tmp_args=()
      for i in $(seq 0 $((MENU_COUNT - 1))); do
        eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
      done
      chosen=$(generic_menu "" "" "${tmp_args[@]}") || { info "Cancelled."; return; }
      git tag -d "$chosen" && ok "Deleted local tag: $chosen"
      ask "Also delete from origin? (y/N):"; read -r confirm
      case "$confirm" in
        [Yy]) git push origin ":refs/tags/$chosen" && ok "Deleted remote tag." ;;
      esac
      ;;
    push-all)
      git push --tags && ok "All tags pushed."
      ;;
  esac
}

# ── Worktree menu ────────────────────────────────────────────
worktree_menu() {
  require_repo || return 1
  local action existing chosen path branch tmp_args i

  section "Worktrees"
  printf "  ${DIM}↑↓ / j k · Enter · q${RESET}\n\n"

  _profiles_to_args "$(printf 'list\nadd\nremove\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  action=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$action" in
    list)
      git worktree list | sed 's/^/  /'
      ;;
    add)
      ask "Path for new worktree (e.g. ../proj-feature):"; read -r path
      [ -z "$path" ] && { err "Path required."; return 1; }
      ask "Branch (existing or new):"; read -r branch
      [ -z "$branch" ] && { err "Branch required."; return 1; }
      if git show-ref --verify --quiet "refs/heads/$branch"; then
        git worktree add "$path" "$branch" && ok "Added worktree at $path"
      else
        git worktree add -b "$branch" "$path" && ok "Created branch '$branch' + worktree at $path"
      fi
      ;;
    remove)
      existing=$(git worktree list --porcelain 2>/dev/null | awk '/^worktree / {print $2}' | tail -n +2)
      if [ -z "$existing" ]; then warn "No additional worktrees."; return; fi
      _profiles_to_args "$existing"
      tmp_args=()
      for i in $(seq 0 $((MENU_COUNT - 1))); do
        eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
      done
      chosen=$(generic_menu "" "" "${tmp_args[@]}") || { info "Cancelled."; return; }
      git worktree remove "$chosen" && ok "Removed: $chosen"
      ;;
  esac
}

# ── Profile export / import ──────────────────────────────────
export_profiles() {
  local out="${1:-ghswitch-profiles.conf}"
  if [ ! -f "$CONFIG_FILE" ]; then err "No config file to export."; return 1; fi

  section "Export profiles"
  awk '/^gh_token=/ {next} {print}' "$CONFIG_FILE" > "$out"
  ok "Wrote ${BOLD}${out}${RESET} (gh_token entries redacted)"
  info "Share with teammates — they'll add their own tokens."
}

import_profiles() {
  local src="$1" confirm line p skip imported=0
  if [ -z "$src" ]; then err "Usage: ghswitch import <file>"; return 1; fi
  if [ ! -f "$src" ]; then err "File not found: $src"; return 1; fi

  section "Import profiles from $src"
  ask "Merge into config? Existing profile names won't be overwritten. (y/N):"; read -r confirm
  case "$confirm" in [Yy]) ;; *) info "Cancelled."; return ;; esac

  skip=0
  while IFS= read -r line; do
    case "$line" in
      \[*\])
        p=$(printf '%s' "$line" | sed 's/^\[//;s/\]$//')
        if profile_exists "$p"; then
          warn "Skipped existing: $p"
          skip=1
        else
          printf '\n%s\n' "$line" >> "$CONFIG_FILE"
          imported=$((imported + 1))
          skip=0
          ok "Imported: $p"
        fi
        ;;
      *)
        [ "$skip" = "0" ] && [ -n "$line" ] && printf '%s\n' "$line" >> "$CONFIG_FILE"
        ;;
    esac
  done < "$src"

  printf '\n'
  ok "Imported $imported new profile(s)."
}

# ── Shell integration emitter ────────────────────────────────
shellinit() {
  cat <<'SHELL'
# ghswitch shell integration — eval "$(ghswitch shellinit)" in your rc file
_ghswitch_chpwd() {
  git rev-parse --is-inside-work-tree >/dev/null 2>&1 || return
  local active_file="${HOME}/.config/ghswitch/active"
  [ -f "$active_file" ] || return
  local active
  active=$(tr -d '[:space:]' < "$active_file" 2>/dev/null)
  [ -z "$active" ] && return
  local config="${HOME}/.config/ghswitch/profiles.conf"
  [ -f "$config" ] || return
  local expected current
  expected=$(awk -v p="[$active]" '$0==p{f=1;next} f && /^\[/{exit} f && /^email=/{sub(/^email=/,"");print;exit}' "$config")
  current=$(git config user.email 2>/dev/null)
  if [ -n "$expected" ] && [ -n "$current" ] && [ "$expected" != "$current" ]; then
    printf '\033[33m⚠ ghswitch:\033[0m user.email "%s" doesn'\''t match active profile "%s" (expects "%s")\n' "$current" "$active" "$expected"
    printf '  run: ghswitch auto   # or: ghswitch local\n'
  fi
}
if [ -n "${ZSH_VERSION:-}" ]; then
  autoload -Uz add-zsh-hook 2>/dev/null && add-zsh-hook chpwd _ghswitch_chpwd
elif [ -n "${BASH_VERSION:-}" ]; then
  case "${PROMPT_COMMAND:-}" in
    *_ghswitch_chpwd*) ;;
    *) PROMPT_COMMAND="_ghswitch_chpwd; ${PROMPT_COMMAND:-:}" ;;
  esac
fi
SHELL
}

# ── Fixup (interactive autosquash) ───────────────────────────
fixup_commit() {
  require_repo || return 1
  local commits chosen hash dirty staged tmp_args i rebase_target

  dirty=$(dirty_count)
  staged=1
  git diff --cached --quiet 2>/dev/null && staged=0
  if [ "$dirty" = "0" ] && [ "$staged" = "0" ]; then
    warn "Nothing to fixup — make some changes first."
    return 1
  fi

  section "Fixup commit"
  printf "  ${DIM}Pick the commit to fix up. ↑↓ / j k · Enter · q${RESET}\n\n"

  commits=$(git log -20 --pretty=format:'%h %s' 2>/dev/null)
  if [ -z "$commits" ]; then err "No commits to choose from."; return 1; fi

  _profiles_to_args "$commits"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  hash=$(printf '%s' "$chosen" | awk '{print $1}')

  info "Staging working tree..."
  git add -A

  if ! git commit --fixup="$hash"; then
    err "Fixup commit failed."
    return 1
  fi
  ok "Created fixup commit"

  if git rev-parse "${hash}~1" >/dev/null 2>&1; then
    rebase_target="${hash}~1"
  else
    rebase_target="--root"
  fi

  info "Running rebase --autosquash..."
  if GIT_SEQUENCE_EDITOR=true git rebase -i --autosquash "$rebase_target"; then
    ok "Rebased — fixup squashed into ${hash}"
  else
    err "Rebase hit conflicts. Resolve, then: git rebase --continue"
    return 1
  fi
}

# ── Recent branches ──────────────────────────────────────────
recent_branches() {
  require_repo || return 1
  local branches chosen current tmp_args i

  current=$(current_branch)
  branches=$(git for-each-ref --count=10 --sort=-committerdate refs/heads --format='%(refname:short)' 2>/dev/null)

  if [ -z "$branches" ]; then warn "No branches."; return 1; fi

  section "Recent branches"
  info "Current: ${BOLD}${current}${RESET}"
  printf "  ${DIM}↑↓ / j k · Enter · q${RESET}\n\n"

  _profiles_to_args "$branches"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "$current" "current" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  if [ "$chosen" = "$current" ]; then info "Already on '$chosen'."; return; fi
  git checkout "$chosen" && ok "Switched to ${BOLD}${chosen}${RESET}"
}

# ── WIP stash helpers ────────────────────────────────────────
wip_save() {
  require_repo || return 1
  local cur dirty msg
  cur=$(current_branch)
  dirty=$(dirty_count)
  if [ "$dirty" = "0" ]; then warn "Nothing to stash."; return; fi

  msg="WIP @ ${cur} · ${dirty} file(s) · $(date '+%H:%M %d-%b')"
  section "Save WIP"
  if git stash push -u -m "$msg"; then
    ok "Stashed: ${msg}"
  else
    err "Stash failed."
    return 1
  fi
}

wip_pop() {
  require_repo || return 1
  section "Pop most recent stash"
  if [ -z "$(git stash list 2>/dev/null)" ]; then
    warn "No stashes to pop."
    return
  fi
  git stash pop
}

# ── Repo refresh / cleanup ───────────────────────────────────
fresh_repo() {
  require_repo || return 1
  local gone b deleted=0

  section "Fresh repo cleanup"

  info "Fetching with --prune..."
  git fetch --all --prune

  info "Cleaning gone tracking branches..."
  gone=$(git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads 2>/dev/null \
        | grep '\[gone\]' | awk '{print $1}')
  if [ -n "$gone" ]; then
    while IFS= read -r b; do
      [ -z "$b" ] && continue
      if git branch -D "$b" >/dev/null 2>&1; then
        ok "Deleted gone branch: $b"
        deleted=$((deleted + 1))
      fi
    done <<GONE
$gone
GONE
  fi
  [ "$deleted" = "0" ] && info "No gone branches."

  info "Running git gc --auto..."
  git gc --auto >/dev/null 2>&1

  ok "Repo refreshed."
}

# ── Enriched PR list ─────────────────────────────────────────
list_prs() {
  require_repo || return 1
  require_gh   || return 1
  local prs num title branch checks review draft cm rm dm

  section "Open pull requests"

  prs=$(gh pr list \
    --json number,title,headRefName,statusCheckRollup,reviewDecision,isDraft \
    --template '{{range .}}{{.number}}{{"\t"}}{{.title}}{{"\t"}}{{.headRefName}}{{"\t"}}{{range .statusCheckRollup}}{{.state}},{{end}}{{"\t"}}{{.reviewDecision}}{{"\t"}}{{.isDraft}}{{"\n"}}{{end}}' \
    2>/dev/null)

  if [ -z "$prs" ]; then info "No open PRs."; return; fi

  printf "  ${BOLD}%-6s %-45s %-22s %-9s %s${RESET}\n" "#" "TITLE" "BRANCH" "CHECKS" "REVIEW"

  while IFS=$'\t' read -r num title branch checks review draft; do
    [ -z "$num" ] && continue
    case "$checks" in
      *FAILURE*|*ERROR*)                cm="${RED}✗ fail${RESET} " ;;
      *PENDING*|*IN_PROGRESS*|*QUEUED*) cm="${YELLOW}⏳ run${RESET}  " ;;
      *SUCCESS*)                        cm="${GREEN}✓ pass${RESET} " ;;
      *)                                cm="${DIM}-${RESET}      " ;;
    esac
    case "$review" in
      APPROVED)          rm="${GREEN}approved${RESET}" ;;
      CHANGES_REQUESTED) rm="${RED}changes${RESET}" ;;
      REVIEW_REQUIRED)   rm="${YELLOW}required${RESET}" ;;
      *)                 rm="${DIM}-${RESET}" ;;
    esac
    dm=""
    [ "$draft" = "true" ] && dm=" ${DIM}[draft]${RESET}"
    printf "  ${CYAN}#%-5s${RESET} %-45.45s ${DIM}%-22.22s${RESET} %b %b%b\n" \
      "$num" "$title" "$branch" "$cm" "$rm" "$dm"
  done <<PRS
$prs
PRS
  printf '\n'
}

# ── PR checkout / status ─────────────────────────────────────
pr_checkout() {
  require_repo || return 1
  require_gh   || return 1
  local num="$1" prs chosen tmp_args i

  if [ -z "$num" ]; then
    prs=$(gh pr list --json number,title,headRefName \
          --template '{{range .}}{{.number}} · {{.title}} ({{.headRefName}}){{"\n"}}{{end}}' 2>/dev/null)
    if [ -z "$prs" ]; then warn "No open PRs."; return; fi

    section "Pick PR to check out"
    printf "  ${DIM}↑↓ / j k · Enter · q${RESET}\n\n"
    _profiles_to_args "$prs"
    tmp_args=()
    for i in $(seq 0 $((MENU_COUNT - 1))); do
      eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
    done
    chosen=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
    printf '\n'
    num=$(printf '%s' "$chosen" | awk '{print $1}')
  fi

  section "Checkout PR #${num}"
  gh pr checkout "$num"
}

pr_status() {
  require_repo || return 1
  require_gh   || return 1
  section "PR status"
  gh pr status
}

# ── Interactive status ───────────────────────────────────────
interactive_status() {
  require_repo || return 1
  local count selected=0 i line status name status_disp key rest msg confirm
  local repo_path

  repo_path=$(repo_root)

  tput civis 2>/dev/null || true
  trap 'tput cnorm 2>/dev/null || true' EXIT INT TERM HUP

  while true; do
    count=0
    while IFS= read -r line; do
      [ -z "$line" ] && continue
      eval "ISTAT_FILE_${count}=\"\$line\""
      count=$((count + 1))
    done <<ISEOF
$(git -C "$repo_path" status --porcelain 2>/dev/null)
ISEOF

    printf '\033[2J\033[H'
    printf "${BOLD}${CYAN}╔══════════════════════════════════════════╗${RESET}\n"
    printf "${BOLD}${CYAN}║${RESET}  ${WHITE}${BOLD}◉ ghswitch status${RESET}  ${DIM}interactive       ${RESET}${BOLD}${CYAN}║${RESET}\n"
    printf "${BOLD}${CYAN}╚══════════════════════════════════════════╝${RESET}\n"
    printf "  ${DIM}↑↓/jk move · ${BOLD}s${RESET}${DIM} stage · ${BOLD}u${RESET}${DIM} unstage · ${BOLD}d${RESET}${DIM} diff · ${BOLD}x${RESET}${DIM} discard · ${BOLD}c${RESET}${DIM} commit · ${BOLD}q${RESET}${DIM} quit${RESET}\n\n"

    if [ "$count" = "0" ]; then
      printf "  ${GREEN}✓${RESET} Working tree clean.\n\n"
      printf "  ${DIM}Press q to exit${RESET}\n"
    fi

    [ "$selected" -ge "$count" ] && [ "$count" -gt 0 ] && selected=$((count - 1))
    [ "$selected" -lt 0 ] && selected=0

    if [ "$count" -gt 0 ]; then
      for i in $(seq 0 $((count - 1))); do
        eval "line=\$ISTAT_FILE_${i}"
        status="${line:0:2}"
        name="${line:3}"
        case "$status" in
          '??')                  status_disp="${RED}??${RESET}" ;;
          ' M'|' D'|' T')        status_disp="${YELLOW}${status}${RESET}" ;;
          'M '|'A '|'D '|'R '|'T ') status_disp="${GREEN}${status}${RESET}" ;;
          'MM'|'AM'|'AD'|'RM')   status_disp="${MAGENTA}${status}${RESET}" ;;
          'UU'|'AA'|'DD'|*U*|*u*) status_disp="${RED}${status}${RESET}" ;;
          *)                     status_disp="${DIM}${status}${RESET}" ;;
        esac
        if [ "$i" -eq "$selected" ]; then
          printf "  ${BG_SEL}${BOLD} › %b  %-50.50s ${RESET}\n" "$status_disp" "$name"
        else
          printf "    %b  %s\n" "$status_disp" "$name"
        fi
      done
    fi

    IFS= read -r -s -n1 key
    case "$key" in
      $'\x1b')
        IFS= read -r -s -n2 -t 0.1 rest 2>/dev/null || rest=''
        key="${key}${rest}"
        ;;
    esac

    case "$key" in
      $'\x1b[A'|k) [ "$selected" -gt 0 ] && selected=$((selected - 1)) ;;
      $'\x1b[B'|j) [ "$selected" -lt $((count - 1)) ] && selected=$((selected + 1)) ;;
      s|' ')
        [ "$count" = "0" ] && continue
        eval "line=\$ISTAT_FILE_${selected}"
        name="${line:3}"
        case "$name" in *' -> '*) name="${name##* -> }" ;; esac
        git -C "$repo_path" add -- "$name" 2>/dev/null
        ;;
      u)
        [ "$count" = "0" ] && continue
        eval "line=\$ISTAT_FILE_${selected}"
        name="${line:3}"
        case "$name" in *' -> '*) name="${name##* -> }" ;; esac
        git -C "$repo_path" restore --staged -- "$name" 2>/dev/null \
          || git -C "$repo_path" reset HEAD -- "$name" >/dev/null 2>&1
        ;;
      d)
        [ "$count" = "0" ] && continue
        eval "line=\$ISTAT_FILE_${selected}"
        name="${line:3}"
        status="${line:0:2}"
        case "$name" in *' -> '*) name="${name##* -> }" ;; esac
        tput cnorm 2>/dev/null || true
        printf '\033[2J\033[H'
        if [ "$status" = '??' ]; then
          printf "${DIM}--- untracked: %s ---${RESET}\n" "$name"
          cat "${repo_path}/${name}" 2>/dev/null | head -200
          printf "\n${DIM}(showing first 200 lines)${RESET}\n"
        else
          git -C "$repo_path" -c color.ui=always diff HEAD -- "$name" | ${PAGER:-less -R}
        fi
        printf "\n${DIM}Press any key to return${RESET}"
        IFS= read -r -s -n1 _ </dev/tty
        tput civis 2>/dev/null || true
        ;;
      x)
        [ "$count" = "0" ] && continue
        eval "line=\$ISTAT_FILE_${selected}"
        name="${line:3}"
        status="${line:0:2}"
        case "$name" in *' -> '*) name="${name##* -> }" ;; esac
        tput cnorm 2>/dev/null || true
        printf "\n  ${RED}${BOLD}Discard '%s'? This cannot be undone. (y/N):${RESET} " "$name"
        IFS= read -r confirm </dev/tty
        case "$confirm" in
          [Yy])
            if [ "$status" = '??' ]; then
              rm -f -- "${repo_path}/${name}" 2>/dev/null
            else
              git -C "$repo_path" checkout -- "$name" 2>/dev/null \
                || git -C "$repo_path" restore -- "$name" 2>/dev/null
            fi
            ;;
        esac
        tput civis 2>/dev/null || true
        ;;
      c)
        tput cnorm 2>/dev/null || true
        printf "\n  ${BOLD}Commit message (empty to cancel):${RESET} "
        IFS= read -r msg </dev/tty
        if [ -n "$msg" ]; then
          git -C "$repo_path" commit -m "$msg"
          printf "\n  ${DIM}Press any key to continue${RESET}"
          IFS= read -r -s -n1 _ </dev/tty
        fi
        tput civis 2>/dev/null || true
        ;;
      q|$'\x03')
        tput cnorm 2>/dev/null || true
        printf '\n'
        return 0
        ;;
    esac
  done
}

# ── Commit signing setup ─────────────────────────────────────
setup_signing() {
  require_gh || return 1
  local choice tmp_args i

  section "Commit signing setup"
  printf "  ${DIM}Pick signing method · ↑↓ · Enter · q${RESET}\n\n"

  _profiles_to_args "$(printf 'SSH (recommended)\nGPG\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  choice=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$choice" in
    'SSH (recommended)') _setup_ssh_signing ;;
    'GPG')               _setup_gpg_signing ;;
  esac
}

_setup_ssh_signing() {
  local email keypath allowed pubkey
  ask "Email for the signing key:"; read -r email
  if [ -z "$email" ]; then err "Email required."; return 1; fi

  keypath="${HOME}/.ssh/id_ed25519_signing"
  if [ -f "$keypath" ]; then
    warn "Key already exists at $keypath — reusing"
  else
    section "Generate SSH signing key"
    if ! ssh-keygen -t ed25519 -C "$email (signing)" -f "$keypath" -N ""; then
      err "Keygen failed."
      return 1
    fi
    ok "Generated $keypath"
  fi

  section "Upload to GitHub as signing key"
  if gh ssh-key add "${keypath}.pub" --title "ghswitch signing key" --type signing 2>&1; then
    ok "Uploaded as signing key"
  else
    warn "Upload failed — maybe already added"
  fi

  section "Configure git for SSH signing"
  git config --global gpg.format ssh
  git config --global user.signingKey "$keypath"
  git config --global commit.gpgsign true
  ok "git config updated (global)"

  allowed="${HOME}/.ssh/allowed_signers"
  pubkey=$(cat "${keypath}.pub" 2>/dev/null)
  if ! grep -q -F "$pubkey" "$allowed" 2>/dev/null; then
    printf '%s namespaces="git" %s\n' "$email" "$pubkey" >> "$allowed"
    ok "Added to $allowed"
  fi
  git config --global gpg.ssh.allowedSignersFile "$allowed"

  printf '\n  ${BOLD}${GREEN}✓ Done.${RESET}  Test with: ${CYAN}git commit -S --allow-empty -m "test"${RESET}\n\n'
}

_setup_gpg_signing() {
  if ! $HAS_GPG; then err "gpg not installed."; return 1; fi
  local email keyid confirm
  ask "Email on existing GPG key (or blank to generate new):"; read -r email

  if [ -z "$email" ]; then
    section "Generate GPG key"
    info "gpg --full-generate-key will run interactively."
    ask "Continue? (y/N):"; read -r confirm
    case "$confirm" in [Yy]) gpg --full-generate-key ;; *) info "Cancelled."; return ;; esac
    ask "Email of the key you just created:"; read -r email
    [ -z "$email" ] && { err "Email required."; return 1; }
  fi

  keyid=$(gpg --list-secret-keys --keyid-format LONG "$email" 2>/dev/null \
          | awk '/^sec/ {split($2, a, "/"); print a[2]; exit}')
  if [ -z "$keyid" ]; then err "Could not find a secret key for $email."; return 1; fi
  ok "Key ID: $keyid"

  section "Upload to GitHub"
  if gpg --armor --export "$keyid" | gh gpg-key add - 2>&1; then
    ok "Uploaded"
  else
    warn "Upload failed — maybe already added"
  fi

  section "Configure git"
  git config --global gpg.format openpgp
  git config --global user.signingKey "$keyid"
  git config --global commit.gpgsign true
  ok "git config updated."

  printf '\n  ${BOLD}${GREEN}✓ Done.${RESET}  Test with: ${CYAN}git commit -S --allow-empty -m "test"${RESET}\n\n'
}

# ── Browser open helpers ─────────────────────────────────────
_open_url() {
  local url="$1"
  case "$OS" in
    macos)   open "$url" >/dev/null 2>&1 ;;
    linux)   xdg-open "$url" >/dev/null 2>&1 ;;
    windows) start "$url" >/dev/null 2>&1 ;;
    *)       printf '%s\n' "$url" ;;
  esac
}

_remote_to_url() {
  local r="$1" host owner_repo
  case "$r" in
    git@*)
      host=$(printf '%s' "$r" | sed -E 's#^git@([^:]+):.*#\1#')
      owner_repo=$(printf '%s' "$r" | sed -E 's#^git@[^:]+:(.+)$#\1#' | sed 's/\.git$//')
      ;;
    https://*)
      host=$(printf '%s' "$r" | sed -E 's#^https://([^/]+)/.*#\1#')
      owner_repo=$(printf '%s' "$r" | sed -E 's#^https://[^/]+/(.+)$#\1#' | sed 's/\.git$//')
      ;;
    ssh://git@*)
      host=$(printf '%s' "$r" | sed -E 's#^ssh://git@([^:/]+).*#\1#')
      owner_repo=$(printf '%s' "$r" | sed -E 's#^ssh://git@[^:/]+[:/](.+)$#\1#' | sed 's/\.git$//')
      ;;
    *)
      return 1 ;;
  esac
  [ -z "$host" ] || [ -z "$owner_repo" ] && return 1
  printf 'https://%s/%s' "$host" "$owner_repo"
}

open_in_browser() {
  require_repo || return 1
  local target="$1" remote base cur url
  remote=$(git config --get remote.origin.url 2>/dev/null)
  if [ -z "$remote" ]; then err "No remote 'origin' set."; return 1; fi
  base=$(_remote_to_url "$remote") || { err "Could not parse remote: $remote"; return 1; }

  cur=$(current_branch)
  case "$target" in
    ""|repo|home) url="$base" ;;
    branch)       url="${base}/tree/${cur}" ;;
    commits)      url="${base}/commits/${cur}" ;;
    pr)
      if $HAS_GH && gh pr view --web >/dev/null 2>&1; then return 0; fi
      url="${base}/pulls" ;;
    prs|pulls)    url="${base}/pulls" ;;
    issues)       url="${base}/issues" ;;
    actions|ci)   url="${base}/actions" ;;
    settings)     url="${base}/settings" ;;
    *)
      err "Unknown target. Try: repo | branch | commits | pr | prs | issues | actions | settings"
      return 1 ;;
  esac

  section "Open in browser"
  info "$url"
  _open_url "$url"
}

# ── Reflog browser ───────────────────────────────────────────
reflog_browser() {
  require_repo || return 1
  local entries chosen sha tmp_args i action sub_args bname confirm

  entries=$(git reflog --pretty=format:'%h %gd %gs' -50 2>/dev/null)
  if [ -z "$entries" ]; then warn "No reflog entries."; return; fi

  section "Reflog"
  printf "  ${DIM}Pick an entry to act on · ↑↓ · Enter · q${RESET}\n\n"
  _profiles_to_args "$entries"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'
  sha=$(printf '%s' "$chosen" | awk '{print $1}')

  section "Selected: ${BOLD}${chosen}${RESET}"
  printf "  ${DIM}Pick action · ↑↓ · Enter · q${RESET}\n\n"
  _profiles_to_args "$(printf 'show diff\ncheckout (detached)\nreset --hard (DESTRUCTIVE)\ncherry-pick onto current\ncreate branch from this\n')"
  sub_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "sub_args[\$i]=\$MENU_ARGS_${i}"
  done
  action=$(generic_menu "" "" "${sub_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$action" in
    "show diff")
      git -c color.ui=always show "$sha" | ${PAGER:-less -R}
      ;;
    "checkout (detached)")
      git checkout "$sha" && ok "Checked out (detached) at $sha"
      ;;
    "reset --hard (DESTRUCTIVE)")
      ask "Reset HEAD to $sha and discard local changes? (y/N):"; read -r confirm
      case "$confirm" in
        [Yy]) git reset --hard "$sha" && ok "Reset to $sha" ;;
        *)    info "Cancelled." ;;
      esac
      ;;
    "cherry-pick onto current")
      if git cherry-pick "$sha"; then
        ok "Cherry-picked $sha"
      else
        err "Conflicts. Resolve, then: git cherry-pick --continue"
      fi
      ;;
    "create branch from this")
      ask "New branch name:"; read -r bname
      [ -z "$bname" ] && { err "Name required."; return 1; }
      git checkout -b "$bname" "$sha" && ok "Created '$bname' at $sha"
      ;;
  esac
}

# ── Revert ───────────────────────────────────────────────────
revert_commit() {
  require_repo || return 1
  local commits chosen sha tmp_args i confirm

  section "Revert a commit"
  printf "  ${DIM}Pick the commit to revert · ↑↓ · Enter · q${RESET}\n\n"

  commits=$(git log -30 --pretty=format:'%h %s' 2>/dev/null)
  if [ -z "$commits" ]; then err "No commits."; return 1; fi

  _profiles_to_args "$commits"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'
  sha=$(printf '%s' "$chosen" | awk '{print $1}')

  ask "Create revert commit for ${sha}? (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      if git revert --no-edit "$sha"; then
        ok "Reverted $sha"
      else
        err "Revert hit conflicts. Resolve, then: git revert --continue"
      fi
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Issue list ───────────────────────────────────────────────
list_issues() {
  require_repo || return 1
  require_gh   || return 1
  local issues num title labels author state

  section "Open issues"

  issues=$(gh issue list \
    --json number,title,labels,author,state \
    --template '{{range .}}{{.number}}{{"\t"}}{{.title}}{{"\t"}}{{range .labels}}{{.name}},{{end}}{{"\t"}}{{.author.login}}{{"\t"}}{{.state}}{{"\n"}}{{end}}' \
    2>/dev/null)

  if [ -z "$issues" ]; then info "No open issues."; return; fi

  printf "  ${BOLD}%-6s %-50s %-20s %s${RESET}\n" "#" "TITLE" "LABELS" "AUTHOR"
  while IFS=$'\t' read -r num title labels author state; do
    [ -z "$num" ] && continue
    labels="${labels%,}"
    printf "  ${CYAN}#%-5s${RESET} %-50.50s ${YELLOW}%-20.20s${RESET} ${DIM}%s${RESET}\n" \
      "$num" "$title" "${labels:--}" "$author"
  done <<ISSUES
$issues
ISSUES
  printf '\n'
}

# ── PR review flow ───────────────────────────────────────────
pr_review() {
  require_repo || return 1
  require_gh   || return 1
  local prs chosen num action body tmp_args i

  prs=$(gh pr list --json number,title,headRefName \
        --template '{{range .}}{{.number}} · {{.title}} ({{.headRefName}}){{"\n"}}{{end}}' 2>/dev/null)
  if [ -z "$prs" ]; then warn "No open PRs."; return; fi

  section "Review a PR"
  printf "  ${DIM}Pick PR · ↑↓ · Enter · q${RESET}\n\n"
  _profiles_to_args "$prs"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'
  num=$(printf '%s' "$chosen" | awk '{print $1}')

  section "PR #${num}"
  printf "  ${DIM}Pick action · ↑↓ · Enter · q${RESET}\n\n"
  _profiles_to_args "$(printf 'view\ndiff\napprove\nrequest-changes\ncomment\nopen in browser\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  action=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$action" in
    view) gh pr view "$num" ;;
    diff) gh pr diff "$num" | ${PAGER:-less -R} ;;
    approve)
      ask "Body (optional):"; read -r body
      if [ -n "$body" ]; then
        gh pr review "$num" --approve --body "$body" && ok "Approved PR #$num"
      else
        gh pr review "$num" --approve && ok "Approved PR #$num"
      fi
      ;;
    request-changes)
      ask "Body (required):"; read -r body
      if [ -z "$body" ]; then err "Body required."; return 1; fi
      gh pr review "$num" --request-changes --body "$body" && ok "Requested changes on PR #$num"
      ;;
    comment)
      ask "Body:"; read -r body
      if [ -z "$body" ]; then err "Body required."; return 1; fi
      gh pr review "$num" --comment --body "$body" && ok "Comment posted on PR #$num"
      ;;
    "open in browser") gh pr view "$num" --web ;;
  esac
}

# ── Notifications inbox ──────────────────────────────────────
list_inbox() {
  require_gh || return 1
  local out

  section "Notifications · review-requested + mentions + assigned"

  printf "\n  ${BOLD}${MAGENTA}▸ PRs awaiting your review${RESET}\n"
  out=$(gh search prs --review-requested=@me --state=open \
        --json number,title,repository,url \
        --template '{{range .}}  {{.repository.nameWithOwner}}#{{.number}} · {{.title}}{{"\n"}}{{end}}' 2>/dev/null)
  if [ -z "$out" ]; then printf "  ${DIM}(none)${RESET}\n"; else printf '%s' "$out"; fi

  printf "\n  ${BOLD}${MAGENTA}▸ PRs mentioning you${RESET}\n"
  out=$(gh search prs --mentions=@me --state=open \
        --json number,title,repository \
        --template '{{range .}}  {{.repository.nameWithOwner}}#{{.number}} · {{.title}}{{"\n"}}{{end}}' 2>/dev/null)
  if [ -z "$out" ]; then printf "  ${DIM}(none)${RESET}\n"; else printf '%s' "$out"; fi

  printf "\n  ${BOLD}${MAGENTA}▸ Issues assigned to you${RESET}\n"
  out=$(gh search issues --assignee=@me --state=open \
        --json number,title,repository \
        --template '{{range .}}  {{.repository.nameWithOwner}}#{{.number}} · {{.title}}{{"\n"}}{{end}}' 2>/dev/null)
  if [ -z "$out" ]; then printf "  ${DIM}(none)${RESET}\n"; else printf '%s' "$out"; fi

  printf "\n  ${BOLD}${MAGENTA}▸ Issues mentioning you${RESET}\n"
  out=$(gh search issues --mentions=@me --state=open \
        --json number,title,repository \
        --template '{{range .}}  {{.repository.nameWithOwner}}#{{.number}} · {{.title}}{{"\n"}}{{end}}' 2>/dev/null)
  if [ -z "$out" ]; then printf "  ${DIM}(none)${RESET}\n"; else printf '%s' "$out"; fi
  printf '\n'
}

# ── Token rotation ───────────────────────────────────────────
rotate_token() {
  local profile="$1" gh_host new_token tmp confirm settings_url

  if [ -z "$profile" ]; then profile=$(get_active); fi
  if [ -z "$profile" ]; then
    err "Usage: ghswitch rotate <profile>  (or set an active profile first)"
    return 1
  fi
  if ! profile_exists "$profile"; then
    err "Profile '$profile' not found."
    return 1
  fi

  gh_host=$(get_field "$profile" "gh_host"); gh_host="${gh_host:-github.com}"
  settings_url="https://${gh_host}/settings/tokens"

  section "Rotate token for ${BOLD}${profile}${RESET}"
  info "Settings page: $settings_url"
  ask "Open in browser? (Y/n):"; read -r confirm
  case "$confirm" in
    [Nn]) ;;
    *)    _open_url "$settings_url" ;;
  esac

  ask "Paste the new token (blank to cancel):"; read -r new_token
  if [ -z "$new_token" ]; then info "Cancelled."; return; fi

  tmp=$(mktemp)
  awk -v prof="[$profile]" -v tok="$new_token" '
    BEGIN { in_section=0; replaced=0 }
    {
      if ($0 == prof) { in_section=1; print; next }
      if (in_section && $0 ~ /^\[/) {
        if (!replaced) print "gh_token=" tok
        in_section=0; replaced=0
        print; next
      }
      if (in_section && $0 ~ /^gh_token=/) {
        print "gh_token=" tok
        replaced=1; next
      }
      print
    }
    END { if (in_section && !replaced) print "gh_token=" tok }
  ' "$CONFIG_FILE" > "$tmp" && mv "$tmp" "$CONFIG_FILE"
  ok "Token updated in profiles.conf"

  if [ "$profile" = "$(get_active)" ]; then
    ask "Re-apply profile to refresh gh auth? (Y/n):"; read -r confirm
    case "$confirm" in [Nn]) ;; *) apply_profile "$profile" "global" ;; esac
  else
    info "Run 'ghswitch $profile' to activate with the new token"
  fi
}

# ── Per-profile commit stats ─────────────────────────────────
commit_stats() {
  require_repo || return 1
  local days="${1:-30}" profiles p email count total=0 root_disp
  case "$days" in ''|*[!0-9]*) days=30 ;; esac

  root_disp=$(repo_root | sed "s|$HOME|~|")
  section "Commit stats · last ${days} days · ${BOLD}${root_disp}${RESET}"

  profiles=$(get_profiles)
  if [ -z "$profiles" ]; then warn "No profiles."; return; fi

  printf "\n"
  while IFS= read -r p; do
    [ -z "$p" ] && continue
    email=$(get_field "$p" "email")
    [ -z "$email" ] && continue
    count=$(git log --since="${days} days ago" --author="$email" --pretty=oneline 2>/dev/null | grep -c .)
    if [ "$count" -gt 0 ]; then
      printf "  ${BOLD}%-20s${RESET}  ${GREEN}%5s${RESET} commits  ${DIM}<%s>${RESET}\n" "$p" "$count" "$email"
      total=$((total + count))
    else
      printf "  ${DIM}%-20s      0 commits  <%s>${RESET}\n" "$p" "$email"
    fi
  done <<PEOF
$profiles
PEOF
  printf "  ${DIM}─────────────────────────────────────────────${RESET}\n"
  printf "  ${BOLD}%-20s  %5s${RESET} commits matched\n\n" "total" "$total"
}

# ── Repo registry ────────────────────────────────────────────
_register_repo() {
  local root="${1:-}"
  if [ -z "$root" ]; then root=$(git rev-parse --show-toplevel 2>/dev/null); fi
  [ -z "$root" ] && return 1
  [ -f "$REPOS_FILE" ] || touch "$REPOS_FILE"
  if ! grep -Fxq "$root" "$REPOS_FILE" 2>/dev/null; then
    printf '%s\n' "$root" >> "$REPOS_FILE"
  fi
}

_purge_repos() {
  [ -f "$REPOS_FILE" ] || return 0
  local tmp r
  tmp=$(mktemp)
  while IFS= read -r r; do
    [ -z "$r" ] && continue
    if [ -d "$r/.git" ] || [ -f "$r/.git" ]; then
      printf '%s\n' "$r" >> "$tmp"
    fi
  done < "$REPOS_FILE"
  mv "$tmp" "$REPOS_FILE"
}

_profile_for_email() {
  local target="$1" p p_email
  [ -z "$target" ] && return 1
  for p in $(get_profiles); do
    p_email=$(get_field "$p" "email")
    if [ "$p_email" = "$target" ]; then
      printf '%s' "$p"
      return 0
    fi
  done
  return 1
}

repos_command() {
  local sub="${1:-list}"; shift 2>/dev/null || true
  local target confirm r br d email pname dlbl

  case "$sub" in
    add)
      require_repo || return 1
      _register_repo
      ok "Added: $(repo_root)"
      ;;
    rm|remove)
      target="${1:-}"
      if [ -z "$target" ]; then target=$(git rev-parse --show-toplevel 2>/dev/null); fi
      if [ -z "$target" ]; then err "Provide a path or run inside a repo."; return 1; fi
      if [ ! -f "$REPOS_FILE" ]; then warn "No registry yet."; return; fi
      grep -Fxv "$target" "$REPOS_FILE" > "${REPOS_FILE}.tmp" || true
      mv "${REPOS_FILE}.tmp" "$REPOS_FILE"
      ok "Removed: $target"
      ;;
    clear)
      ask "Clear all repos from registry? (y/N):"; read -r confirm
      case "$confirm" in
        [Yy]) printf '' > "$REPOS_FILE"; ok "Cleared." ;;
        *)    info "Cancelled." ;;
      esac
      ;;
    ""|list|ls)
      _purge_repos
      if [ ! -s "$REPOS_FILE" ] 2>/dev/null; then
        warn "No repos registered."
        info "Auto-registers when you run 'ghswitch local' or 'ghswitch auto' inside a repo."
        info "Add manually with: ghswitch repos add"
        return
      fi
      section "Registered repos"
      printf "  ${BOLD}%-46s %-20s %-7s %s${RESET}\n" "PATH" "BRANCH" "DIRTY" "PROFILE"
      while IFS= read -r r; do
        [ -z "$r" ] && continue
        br=$(git -C "$r" rev-parse --abbrev-ref HEAD 2>/dev/null)
        d=$(git -C "$r" status --porcelain 2>/dev/null | grep -c .)
        email=$(git -C "$r" config user.email 2>/dev/null)
        pname=$(_profile_for_email "$email")
        if [ "$d" = "0" ]; then dlbl="${GREEN}clean${RESET}"; else dlbl="${YELLOW}${d}${RESET}    "; fi
        printf "  %-46.46s ${CYAN}%-20.20s${RESET} %b ${MAGENTA}%s${RESET}\n" \
          "$(printf '%s' "$r" | sed "s|$HOME|~|")" "${br:--}" "$dlbl" "${pname:--}"
      done < "$REPOS_FILE"
      printf '\n'
      ;;
    *)
      err "Unknown subcommand: $sub"
      info "Try: list | add | rm [path] | clear"
      return 1
      ;;
  esac
}

sync_all_repos() {
  _purge_repos
  if [ ! -s "$REPOS_FILE" ] 2>/dev/null; then
    warn "No repos registered. Use 'ghswitch repos add'."
    return 1
  fi

  section "Sync all registered repos"
  local r br up dirty ok_n=0 fail=0 skip=0
  while IFS= read -r r; do
    [ -z "$r" ] && continue
    printf "\n  ${BOLD}${CYAN}▸ %s${RESET}\n" "$(printf '%s' "$r" | sed "s|$HOME|~|")"
    br=$(git -C "$r" rev-parse --abbrev-ref HEAD 2>/dev/null)
    up=$(git -C "$r" rev-parse --abbrev-ref --symbolic-full-name "@{u}" 2>/dev/null)
    if [ -z "$up" ]; then
      warn "${br}: no upstream — skip"
      skip=$((skip + 1))
      continue
    fi
    dirty=$(git -C "$r" status --porcelain 2>/dev/null | grep -c .)
    if [ "$dirty" != "0" ]; then
      warn "${br}: ${dirty} dirty file(s) — skip"
      skip=$((skip + 1))
      continue
    fi
    if git -C "$r" pull --rebase 2>&1 | sed 's/^/    /'; then
      if git -C "$r" push 2>&1 | sed 's/^/    /'; then
        ok "synced"
        ok_n=$((ok_n + 1))
      else
        err "push failed"
        fail=$((fail + 1))
      fi
    else
      err "pull failed"
      fail=$((fail + 1))
    fi
  done < "$REPOS_FILE"

  printf "\n  ${BOLD}Summary:${RESET} ${GREEN}%d synced${RESET} · ${YELLOW}%d skipped${RESET} · ${RED}%d failed${RESET}\n\n" \
    "$ok_n" "$skip" "$fail"
}

# ── Release wizard ───────────────────────────────────────────
release_wizard() {
  require_repo || return 1
  require_gh   || return 1
  local last_tag bump base new_tag commits notes draft confirm
  local maj min pat tmp_args i

  section "Release wizard"

  last_tag=$(git describe --tags --abbrev=0 2>/dev/null)
  if [ -n "$last_tag" ]; then
    info "Last tag: ${BOLD}${last_tag}${RESET}"
  else
    info "No previous tag — first release"
    last_tag=""
  fi

  printf "\n  ${DIM}Pick version bump · ↑↓ · Enter · q${RESET}\n\n"
  _profiles_to_args "$(printf 'patch (x.y.Z+1)\nminor (x.Y+1.0)\nmajor (X+1.0.0)\ncustom\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  bump=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  base="${last_tag#v}"
  base="${base:-0.0.0}"
  maj=$(printf '%s' "$base" | awk -F. '{print ($1 ~ /^[0-9]+$/) ? $1 : 0}')
  min=$(printf '%s' "$base" | awk -F. '{print ($2 ~ /^[0-9]+$/) ? $2 : 0}')
  pat=$(printf '%s' "$base" | awk -F. '{print ($3 ~ /^[0-9]+$/) ? $3 : 0}')

  case "$bump" in
    "patch (x.y.Z+1)") new_tag="v${maj}.${min}.$((pat + 1))" ;;
    "minor (x.Y+1.0)") new_tag="v${maj}.$((min + 1)).0" ;;
    "major (X+1.0.0)") new_tag="v$((maj + 1)).0.0" ;;
    custom)
      ask "Tag (e.g. v1.2.3):"; read -r new_tag
      [ -z "$new_tag" ] && { err "Tag required."; return 1; }
      ;;
  esac

  info "New tag: ${BOLD}${new_tag}${RESET}"

  if [ -n "$last_tag" ]; then
    commits=$(git log "${last_tag}..HEAD" --pretty=format:'- %s (%h)' --no-merges 2>/dev/null)
  else
    commits=$(git log --pretty=format:'- %s (%h)' --no-merges 2>/dev/null | head -50)
  fi

  if [ -z "$commits" ]; then
    warn "No commits since ${last_tag:-repo start}."
    ask "Create tag anyway? (y/N):"; read -r confirm
    case "$confirm" in [Yy]) ;; *) info "Cancelled."; return ;; esac
  fi

  notes=$(printf '## Changes\n\n%s\n' "$commits")

  printf "\n  ${BOLD}Release notes preview:${RESET}\n"
  printf '%s\n' "$notes" | sed 's/^/    /'
  printf '\n'

  ask "Mark as draft release? (Y/n):"; read -r draft
  case "$draft" in [Nn]) draft="" ;; *) draft="--draft" ;; esac

  ask "Tag locally, push, and create release on origin? (y/N):"; read -r confirm
  case "$confirm" in
    [Yy])
      git tag -a "$new_tag" -m "$new_tag" || { err "Tag failed."; return 1; }
      git push origin "$new_tag" || { err "Push tag failed."; return 1; }
      if printf '%s' "$notes" | gh release create "$new_tag" --title "$new_tag" --notes-file - $draft; then
        ok "Release created: $new_tag"
      else
        err "gh release create failed (tag is pushed)"
        return 1
      fi
      ;;
    *) info "Cancelled." ;;
  esac
}

# ── Cherry-pick wizard ───────────────────────────────────────
cherry_pick_wizard() {
  require_repo || return 1
  local branches branch chosen sha tmp_args i commits cur

  cur=$(current_branch)
  branches=$(list_branches | grep -v "^${cur}\$")
  if [ -z "$branches" ]; then warn "No other branches to pick from."; return 1; fi

  section "Cherry-pick from another branch"
  info "Onto: ${BOLD}${cur}${RESET}"
  printf "  ${DIM}Pick source branch · ↑↓ · Enter · q${RESET}\n\n"

  _profiles_to_args "$branches"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  branch=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  commits=$(git log -20 --pretty=format:'%h %s' "$branch" 2>/dev/null)
  if [ -z "$commits" ]; then err "No commits on $branch."; return 1; fi

  section "Pick commit from ${BOLD}${branch}${RESET}"
  printf "  ${DIM}↑↓ · Enter · q${RESET}\n\n"
  _profiles_to_args "$commits"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  chosen=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'
  sha=$(printf '%s' "$chosen" | awk '{print $1}')

  if git cherry-pick "$sha"; then
    ok "Cherry-picked $sha onto $cur"
  else
    err "Cherry-pick hit conflicts. Resolve, then: git cherry-pick --continue"
    return 1
  fi
}

# ── Rebase wizard ────────────────────────────────────────────
rebase_wizard() {
  require_repo || return 1
  local choice tmp_args i n branches branch up cur

  cur=$(current_branch)

  section "Interactive rebase"
  info "On: ${BOLD}${cur}${RESET}"
  printf "  ${DIM}Pick mode · ↑↓ · Enter · q${RESET}\n\n"

  _profiles_to_args "$(printf 'rebase last N commits (interactive)\nrebase onto another branch\nrebase onto upstream\n')"
  tmp_args=()
  for i in $(seq 0 $((MENU_COUNT - 1))); do
    eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
  done
  choice=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
  printf '\n'

  case "$choice" in
    "rebase last N commits (interactive)")
      ask "How many commits back? (default 5):"; read -r n
      n="${n:-5}"
      case "$n" in *[!0-9]*|'') err "Need a positive integer."; return 1 ;; esac
      git rebase -i "HEAD~${n}"
      ;;
    "rebase onto another branch")
      branches=$(list_branches | grep -v "^${cur}\$")
      if [ -z "$branches" ]; then err "No other branches."; return 1; fi
      printf "  ${DIM}Pick base branch · ↑↓ · Enter · q${RESET}\n\n"
      _profiles_to_args "$branches"
      tmp_args=()
      for i in $(seq 0 $((MENU_COUNT - 1))); do
        eval "tmp_args[\$i]=\$MENU_ARGS_${i}"
      done
      branch=$(generic_menu "" "" "${tmp_args[@]}") || { printf '\n'; info "Cancelled."; return; }
      printf '\n'
      git rebase "$branch"
      ;;
    "rebase onto upstream")
      up=$(upstream_of)
      if [ -z "$up" ]; then err "No upstream set for $cur."; return 1; fi
      info "Fetching, then rebasing onto $up"
      git fetch && git rebase "$up"
      ;;
  esac
}

# ── Usage ────────────────────────────────────────────────────
usage() {
  printf "${BOLD}Usage:${RESET}  ghswitch ${CYAN}[command] [args...]${RESET}\n\n"

  printf "${BOLD}${MAGENTA}▸ Profiles${RESET}\n"
  printf "  ${CYAN}%-14s${RESET}  %s\n" \
    "(none)"        "Interactive arrow-key profile switcher" \
    "add"           "Add a new profile" \
    "list | ls"     "Show all profiles" \
    "local"         "Apply profile to current repo only" \
    "status"        "Show current git identity + gh auth" \
    "unset [local]" "Clear global (or local) git identity" \
    "edit"          "Open profiles.conf in \$EDITOR" \
    "delete | rm"   "Remove a profile" \
    "<name>"        "Switch directly, e.g. ghswitch work"

  printf "\n${BOLD}${MAGENTA}▸ Branches${RESET}\n"
  printf "  ${CYAN}%-14s${RESET}  %s\n" \
    "branch | br"   "Interactive branch switcher (all branches)" \
    "recent"        "Switch to one of last 10 active branches" \
    "new [name]"    "Create + checkout new branch" \
    "rename [name]" "Rename current branch" \
    "clean"         "Delete merged local branches" \
    "fresh"         "Fetch --prune + drop gone branches + gc"

  printf "\n${BOLD}${MAGENTA}▸ Commits${RESET}\n"
  printf "  ${CYAN}%-14s${RESET}  %s\n" \
    "commit | c"    "Stage all + commit (msg as arg or prompt)" \
    "cc"            "Conventional commit wizard (feat/fix/...)" \
    "fixup"         "Pick a past commit + autosquash fixup" \
    "amend"         "Stage all + amend last commit (no edit)" \
    "undo"          "Soft reset HEAD~1 (keep changes staged)" \
    "wip"           "Quick stash with auto-message" \
    "wip-pop"       "Pop the most recent stash" \
    "st"            "Interactive status (lazygit-style picker)"

  printf "\n${BOLD}${MAGENTA}▸ History${RESET}\n"
  printf "  ${CYAN}%-16s${RESET}  %s\n" \
    "log | l [N]"     "Pretty graph log (default 20)" \
    "diff | d"        "Show working-tree diff vs HEAD" \
    "who [file]"      "Top contributors (or blame a file)" \
    "reflog"          "Interactive reflog browser (recover lost commits)" \
    "revert"          "Pick a commit + create revert" \
    "cherry-pick"     "Pick commit from another branch" \
    "rebase"          "Interactive rebase wizard"

  printf "\n${BOLD}${MAGENTA}▸ Sync${RESET}\n"
  printf "  ${CYAN}%-16s${RESET}  %s\n" \
    "sync"            "Pull --rebase, then push" \
    "pull"            "Pull --rebase from upstream" \
    "push"            "Push current branch (-u on first push)" \
    "fetch"           "Fetch all remotes (--prune)" \
    "sync-all"        "Run sync across every registered repo"

  printf "\n${BOLD}${MAGENTA}▸ Stash · Tags · Worktrees${RESET}\n"
  printf "  ${CYAN}%-16s${RESET}  %s\n" \
    "stash"           "Menu: save / list / pop / apply / drop / clear" \
    "tag"             "Menu: list / create / delete / push-all" \
    "worktree | wt"   "Menu: list / add / remove"

  printf "\n${BOLD}${MAGENTA}▸ GitHub${RESET}\n"
  printf "  ${CYAN}%-16s${RESET}  %s\n" \
    "pr"              "Open a pull request via gh CLI" \
    "pr-list | prs"   "List PRs with checks + review status" \
    "pr-checkout [N]" "Check out a PR (menu if no number given)" \
    "pr-status"       "Status of PR for current branch" \
    "review"          "Pick a PR + approve / request-changes / comment" \
    "issues"          "List open issues" \
    "inbox"           "Review-requested + mentions + assigned" \
    "open [target]"   "Open repo/branch/pr/issues/actions in browser" \
    "release"         "Tag + push + draft GitHub release" \
    "conflict"        "Walk through merge conflicts in \$EDITOR"

  printf "\n${BOLD}${MAGENTA}▸ Repo & Diagnostics${RESET}\n"
  printf "  ${CYAN}%-16s${RESET}  %s\n" \
    "repo | info"     "Repo dashboard (branch, sync, dirty, profile)" \
    "doctor"          "Diagnose SSH / GPG / gh / identity / token expiry" \
    "auto"            "Auto-detect + apply matching profile to repo" \
    "stats [days]"    "Per-profile commit count in this repo (default 30d)" \
    "repos [sub]"     "Multi-repo registry: list | add | rm | clear" \
    "rotate [profile]" "Guided token rotation (opens settings page)"

  printf "\n${BOLD}${MAGENTA}▸ Team Setup${RESET}\n"
  printf "  ${CYAN}%-14s${RESET}  %s\n" \
    "bootstrap URL [profile]" "Clone repo + apply local profile" \
    "hooks-install" "Install pre-commit guard against wrong identity" \
    "signed"        "Wizard: SSH or GPG commit signing setup" \
    "shellinit"     "Emit shell hook (eval \"\$(ghswitch shellinit)\")" \
    "export [file]" "Export profiles.conf (tokens redacted)" \
    "import <file>" "Merge profiles from a shared file" \
    "help | -h"     "This help text"

  printf "\n${BOLD}Per-repo preset:${RESET}  drop ${DIM}.ghswitch.yml${RESET} in repo root with ${DIM}profile: <name>${RESET}\n"

  printf "\n${BOLD}Config file:${RESET}  %s\n\n" "${CONFIG_FILE}"
}

# ── Main ─────────────────────────────────────────────────────
main() {
  if ! command -v git >/dev/null 2>&1; then
    err "git is not installed or not in PATH."; exit 1
  fi

  init_config

  local cmd="${1:-}"

  # Skip the banner for commands whose output is consumed by other tools
  # or that take over the screen themselves.
  case "$cmd" in
    shellinit|st) ;;
    *) header ;;
  esac

  case "$cmd" in
    "")              interactive_switch ;;

    # Profiles
    add)             add_profile        ;;
    list|ls)         list_profiles      ;;
    local)           apply_local        ;;
    status|whoami)   show_status        ;;
    unset|reset)
      if [ "$2" = "--local" ] || [ "$2" = "local" ]; then
        unset_local
      else
        unset_global
      fi
      ;;
    edit)            edit_config        ;;
    delete|rm)       delete_profile     ;;

    # Branches
    branch|br)       switch_branch       ;;
    recent)          recent_branches     ;;
    new)             new_branch    "$2"  ;;
    rename|mv)       rename_branch "$2"  ;;
    clean)           clean_branches      ;;
    fresh|nuke)      fresh_repo          ;;

    # Commits
    commit|c)        quick_commit  "$2"  ;;
    cc)              conventional_commit ;;
    fixup)           fixup_commit        ;;
    amend)           amend_commit        ;;
    undo)            undo_commit         ;;
    wip)             wip_save            ;;
    wip-pop)         wip_pop             ;;
    st)              interactive_status  ;;

    # Sync
    sync)            sync_repo           ;;
    pull)            pull_branch         ;;
    push)            push_branch         ;;
    fetch)           fetch_all           ;;

    # History
    log|l)           pretty_log    "$2"  ;;
    diff|d)          show_diff           ;;
    who)             show_who      "$2"  ;;
    reflog)          reflog_browser      ;;
    revert)          revert_commit       ;;
    cherry-pick|cp)  cherry_pick_wizard  ;;
    rebase)          rebase_wizard       ;;

    # Sync (extra)
    sync-all)        sync_all_repos      ;;

    # Stash · Tags · Worktrees
    stash)           stash_menu          ;;
    tag|tags)        tag_menu            ;;
    worktree|wt)     worktree_menu       ;;

    # GitHub & conflicts
    pr)              create_pr           ;;
    pr-list|prs)     list_prs            ;;
    pr-checkout|prco) pr_checkout  "$2"  ;;
    pr-status)       pr_status           ;;
    review)          pr_review           ;;
    issues)          list_issues         ;;
    inbox|notifications) list_inbox      ;;
    open)            open_in_browser "$2" ;;
    release)         release_wizard      ;;
    conflict|merge)  conflict_helper     ;;

    # Repo & diagnostics
    repo|info)       repo_info           ;;
    doctor)          doctor              ;;
    auto)            auto_profile  "$2"  ;;
    stats)           commit_stats  "$2"  ;;
    repos)           repos_command "$2" "$3" ;;
    rotate)          rotate_token  "$2"  ;;

    # Team setup
    bootstrap)       bootstrap_repo "$2" "$3" ;;
    hooks-install)   install_hooks       ;;
    signed)          setup_signing       ;;
    shellinit)       shellinit           ;;
    export)          export_profiles "$2" ;;
    import)          import_profiles "$2" ;;

    help|--help|-h)  usage               ;;
    *)
      if profile_exists "$cmd"; then
        apply_profile "$cmd" "global"
      else
        err "Unknown command or profile: '$cmd'"
        printf '\n'; usage; exit 1
      fi
      ;;
  esac
}

main "$@"