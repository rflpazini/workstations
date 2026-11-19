#!/usr/bin/env bash
# Keeps this workspace in sync with GitHub so every device sees the same files.
# Claude Code runs it from .claude/settings.json:
#   start   when a session opens: bring in what other devices saved
#   prompt  before each message: same, at most once per interval
#   stop    after each answer: save everything and push it
# It never blocks a session. Outside a git repo with a remote it does nothing.
set -u

mode="${1:-}"
cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null || exit 0
command -v git >/dev/null 2>&1 || exit 0
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || exit 0
git remote get-url origin >/dev/null 2>&1 || exit 0

# Inside the template repo itself, never auto-save: changes there are reviewed.
upstream=$(sed -n 's/.*"upstream"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' .claude/workspace.json 2>/dev/null)
origin=$(git config --get remote.origin.url)
origin=${origin%.git}
origin=${origin#*github.com[:/]}
[ -n "$upstream" ] && [ "$origin" = "$upstream" ] && exit 0

base="${WORKSPACE_BASE_BRANCH:-main}"
interval="${WORKSPACE_SYNC_INTERVAL:-60}"
stamp="$(git rev-parse --git-dir)/workspace-sync-fetched"

ensure_identity() {
  git config user.email >/dev/null 2>&1 || git config user.email "workspace@users.noreply.github.com"
  git config user.name >/dev/null 2>&1 || git config user.name "Workspace"
}

save() {
  ensure_identity
  git add -A
  git diff --cached --quiet || git commit --quiet -m "workspace: auto-save"
}

# New work = commits on HEAD that neither main nor this branch's last push has.
# Commits only brought in from main do not count.
has_unpushed_commits() {
  set -- "origin/$base"
  git rev-parse --verify --quiet '@{u}' >/dev/null && set -- "$@" '@{u}'
  [ -n "$(git rev-list HEAD --not "$@" 2>/dev/null)" ]
}

# Saves local work first, so a merge never trips over unsaved files.
bring_in_main() {
  save
  if ! git fetch --quiet origin "$base" 2>/dev/null; then
    echo "Workspace sync: could not reach GitHub. Tell the user changes from their other devices may be missing in this session."
    return
  fi
  date +%s >"$stamp"
  if ! git merge --quiet --no-edit "origin/$base" >/dev/null 2>&1; then
    git merge --abort >/dev/null 2>&1
    echo "Workspace sync: could not bring in the latest changes from other devices because the same file changed on both sides. Tell the user, and keep working with this session's version."
    return
  fi
  [ "$mode" = start ] && echo "Workspace sync: up to date with the latest changes from all devices."
}

fetched_recently() {
  local last now
  last=$(cat "$stamp" 2>/dev/null || echo 0)
  now=$(date +%s)
  [ $((now - last)) -lt "$interval" ]
}

# Exit 2 hands the message back to Claude so it never tells the user "saved"
# when the push failed. Unpushed commits are retried after the next answer.
push_or_warn() {
  local hook_input=""
  has_unpushed_commits || return 0
  git push --quiet -u origin "$(git symbolic-ref --short HEAD)" 2>/dev/null && return 0
  [ -t 0 ] || hook_input=$(cat)
  grep -Eq '"stop_hook_active"[[:space:]]*:[[:space:]]*true' <<<"$hook_input" && return 0
  echo "Workspace sync: the latest changes are NOT saved to GitHub yet (push failed). Tell the user in one sentence that this change is only on this device for now and will be saved again after the next answer." >&2
  exit 2
}

case "$mode" in
  start)
    bring_in_main
    ;;
  prompt)
    fetched_recently || bring_in_main
    ;;
  stop)
    save
    push_or_warn
    ;;
esac
exit 0
