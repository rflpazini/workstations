#!/usr/bin/env bash
# Saves every claude/* session branch into main, one at a time and oldest first,
# then deletes it, so what any device saved reaches main without anyone opening
# a pull request. Run by .github/workflows/sync-to-main.yml; safe to rerun.
#   Markdown files keep both sides (.gitattributes: merge=union).
#   Any other file: the device that saved last wins.
#   A branch that cannot be saved (git cannot merge it, or GitHub refuses the
#   push) is kept, every other branch is still saved, and the run fails so
#   someone notices.
set -eu

base="${WORKSPACE_BASE_BRANCH:-main}"
prefix="${WORKSPACE_SESSION_PREFIX:-claude}"
delay="${WORKSPACE_RETRY_DELAY:-1}"

git config user.email >/dev/null 2>&1 || git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git config user.name >/dev/null 2>&1 || git config user.name "github-actions[bot]"

# Deletes the branch only if it still points where it was saved; a device may
# have pushed newer changes meanwhile, and the next run saves those.
remove_if_unchanged() {
  if git push --quiet --force-with-lease="refs/heads/$1:$2" origin --delete "$1" 2>/dev/null; then
    echo "Saved $1 to $base and removed it."
  else
    echo "Saved $1 to $base up to ${2:0:7} and kept it: it received newer changes."
  fi
}

# Merges one branch into the latest main and pushes. Retries when main moved
# under us; gives up when git cannot merge or GitHub refuses the push itself.
save_branch() {
  local branch="$1" sha="$2" attempt=1 before
  while :; do
    git fetch --quiet origin "$base" || return 1
    git checkout --quiet -B "$base" "origin/$base" || return 1
    before=$(git rev-parse "origin/$base")
    if ! git merge --quiet --no-edit -X theirs -m "workspace: save changes from $branch" "$sha" >/dev/null 2>&1; then
      git merge --abort >/dev/null 2>&1 || true
      return 1
    fi
    if git push --quiet origin "$base" 2>/dev/null; then
      remove_if_unchanged "$branch" "$sha"
      return 0
    fi
    git fetch --quiet origin "$base" || return 1
    if [ "$(git rev-parse "origin/$base")" = "$before" ] || [ "$attempt" -ge 5 ]; then
      return 1
    fi
    echo "$base moved during the run; trying again."
    sleep $((attempt * delay))
    attempt=$((attempt + 1))
  done
}

git fetch --quiet --prune origin "+refs/heads/*:refs/remotes/origin/*"
branches=$(git for-each-ref --sort=committerdate --format='%(refname:lstrip=3) %(objectname)' "refs/remotes/origin/$prefix")
if [ -z "$branches" ]; then
  echo "Nothing to merge."
  exit 0
fi

stuck=""
while read -r branch sha; do
  save_branch "$branch" "$sha" </dev/null || stuck="$stuck $branch"
done <<EOF
$branches
EOF

for branch in $stuck; do
  echo "::error::$branch could not be saved to $base. Its changes are kept on that branch."
done
[ -z "$stuck" ]
