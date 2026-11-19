package syncspec

import (
	"strings"
	"testing"
)

// saveFromDevice makes a change on a device session branch and lets the stop
// hook push it, exactly as after an answer in a real session.
func saveFromDevice(w *world, dir, file, line string, env ...string) {
	w.t.Helper()
	appendLine(w.t, dir, file, line)
	if res := w.sync(dir, "stop", "{}", env...); res.code != 0 {
		w.t.Fatalf("stop hook exit %d: %s", res.code, res.stderr)
	}
}

func TestMergeToMain_MergesSessionBranchAndDeletesIt(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/inbox-1")
	saveFromDevice(w, phone, "MEMORY.md", "- dentist on Tuesday at 3pm")

	res := w.mergeToMain()

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := w.remoteFile("main", "MEMORY.md"); !strings.Contains(got, "dentist on Tuesday") {
		t.Errorf("main MEMORY.md = %q, want the phone's line", got)
	}
	if w.remoteHasBranch("claude/inbox-1") {
		t.Error("session branch still exists after merge")
	}
}

func TestMergeToMain_DoesNothing_WhenThereAreNoSessionBranches(t *testing.T) {
	w := newWorld(t)
	before := w.remoteFile("main", "MEMORY.md")

	res := w.mergeToMain()

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := w.remoteFile("main", "MEMORY.md"); got != before {
		t.Errorf("main changed to %q with nothing to merge", got)
	}
}

func TestMergeToMain_KeepsBothLines_WhenTwoDevicesAppendToSameFile(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/a-1")
	pc := w.device("pc", "claude/b-1")
	saveFromDevice(w, ipad, "MEMORY.md", "- line written on the iPad")
	saveFromDevice(w, pc, "MEMORY.md", "- line written on the PC")

	res := w.mergeToMain()

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	got := w.remoteFile("main", "MEMORY.md")
	if !strings.Contains(got, "written on the iPad") || !strings.Contains(got, "written on the PC") {
		t.Errorf("main MEMORY.md = %q, want both lines", got)
	}
	if w.remoteHasBranch("claude/a-1") || w.remoteHasBranch("claude/b-1") {
		t.Error("a session branch survived a clean merge")
	}
}

func TestMergeToMain_LatestDeviceWins_WhenBinaryFileConflicts(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/z-budget")
	pc := w.device("pc", "claude/a-budget")
	writeFile(t, ipad, "budget.xlsx", "\x00ipad\x00")
	if res := w.sync(ipad, "stop", "{}", "GIT_COMMITTER_DATE=2026-11-18T10:00:00Z"); res.code != 0 {
		t.Fatal(res.stderr)
	}
	writeFile(t, pc, "budget.xlsx", "\x00pc, saved later\x00")
	if res := w.sync(pc, "stop", "{}", "GIT_COMMITTER_DATE=2026-11-18T11:00:00Z"); res.code != 0 {
		t.Fatal(res.stderr)
	}

	res := w.mergeToMain()

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := w.remoteFile("main", "budget.xlsx"); got != "\x00pc, saved later\x00" {
		t.Errorf("main budget.xlsx = %q, want the version saved last", got)
	}
}

func TestMergeToMain_KeepsBranch_WhenItReceivesChangesDuringTheRun(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/busy-1")
	saveFromDevice(w, phone, "MEMORY.md", "- first answer")
	appendLine(t, phone, "MEMORY.md", "- second answer")
	git(t, w.env, phone, "commit", "--quiet", "-am", "second answer")
	git(t, w.env, phone, "push", "--quiet", "origin", "HEAD:refs/heads/pending")
	// The phone pushes its second answer right when the Action updates main.
	w.remoteHook("post-receive", `#!/bin/sh
while read old new ref; do
  [ "$ref" = refs/heads/main ] && git update-ref refs/heads/claude/busy-1 refs/heads/pending
done
exit 0
`)

	res := w.mergeToMain()

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if !w.remoteHasBranch("claude/busy-1") {
		t.Fatal("deleted a branch that had unmerged changes")
	}
	w.remoteHook("post-receive", "#!/bin/sh\nexit 0\n")
	w.mergeToMain()
	if got := w.remoteFile("main", "MEMORY.md"); !strings.Contains(got, "second answer") {
		t.Errorf("main MEMORY.md = %q, want the second answer after the next run", got)
	}
}

func TestMergeToMain_RetriesPush_WhenMainMovesDuringTheRun(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/retry-1")
	saveFromDevice(w, phone, "MEMORY.md", "- from the phone session")
	pc := w.device("pc", "main")
	appendLine(t, pc, "MEMORY.md", "- pushed straight to main by the PC")
	git(t, w.env, pc, "commit", "--quiet", "-am", "pc")
	git(t, w.env, pc, "push", "--quiet", "origin", "HEAD:refs/heads/pc-pending")
	// The first push to main loses a race against the PC; later pushes go through.
	w.remoteHook("update", `#!/bin/sh
[ "$1" = refs/heads/main ] || exit 0
[ -f raced ] && exit 0
touch raced
git update-ref refs/heads/main refs/heads/pc-pending
exit 1
`)

	res := w.mergeToMain()

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	got := w.remoteFile("main", "MEMORY.md")
	if !strings.Contains(got, "from the phone session") || !strings.Contains(got, "by the PC") {
		t.Errorf("main MEMORY.md = %q, want both the phone and the PC lines", got)
	}
}

func TestMergeToMain_SkipsUnmergeableBranch_AndMergesTheOthers(t *testing.T) {
	w := newWorld(t)
	seedFile := w.device("seeder", "main")
	writeFile(t, seedFile, "notes.txt", "groceries\n")
	git(t, w.env, seedFile, "add", "-A")
	git(t, w.env, seedFile, "commit", "--quiet", "-m", "notes")
	git(t, w.env, seedFile, "push", "--quiet", "origin", "main")

	phone := w.device("phone", "claude/delete-1")
	if err := removeFile(phone, "notes.txt"); err != nil {
		t.Fatal(err)
	}
	if res := w.sync(phone, "stop", "{}"); res.code != 0 {
		t.Fatal(res.stderr)
	}
	pc := w.device("pc", "main")
	writeFile(t, pc, "notes.txt", "groceries\nmilk\n")
	git(t, w.env, pc, "commit", "--quiet", "-am", "pc edits notes")
	git(t, w.env, pc, "push", "--quiet", "origin", "main")
	ipad := w.device("ipad", "claude/fine-1")
	saveFromDevice(w, ipad, "MEMORY.md", "- unrelated iPad note")

	res := w.mergeToMain()

	if res.code == 0 || !strings.Contains(res.stdout, "claude/delete-1 could not be saved to main") {
		t.Errorf("got exit %d, stdout %q; want a failed run naming the stuck branch", res.code, res.stdout)
	}
	if got := w.remoteFile("main", "MEMORY.md"); !strings.Contains(got, "unrelated iPad note") {
		t.Errorf("main MEMORY.md = %q, want the iPad note merged despite the other branch", got)
	}
	if !w.remoteHasBranch("claude/delete-1") {
		t.Error("deleted the branch that could not be merged; its changes would be lost")
	}
}

func TestMergeToMain_FailsAndKeepsBranches_WhenMainNeverAcceptsThePush(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/stuck-1")
	saveFromDevice(w, phone, "MEMORY.md", "- a line that must not be lost")
	w.remoteHook("update", "#!/bin/sh\n[ \"$1\" = refs/heads/main ] && exit 1\nexit 0\n")

	res := w.mergeToMain("WORKSPACE_RETRY_DELAY=0")

	if res.code == 0 || !strings.Contains(res.stdout+res.stderr, "claude/stuck-1 could not be saved to main") {
		t.Errorf("got exit %d, stderr %q; want a failed run that says why", res.code, res.stderr)
	}
	if !w.remoteHasBranch("claude/stuck-1") {
		t.Error("deleted the session branch although main never received its changes")
	}
}

// githubRefusesWorkflowEdits mimics GitHub refusing a GITHUB_TOKEN push that
// changes .github/workflows (the token has no "workflows" permission).
const githubRefusesWorkflowEdits = `#!/bin/sh
[ "$1" = refs/heads/main ] || exit 0
if git diff --name-only "$2" "$3" | grep -q '^\.github/workflows/'; then
  echo "refusing to allow a GitHub App to create or update workflow without workflows permission" >&2
  exit 1
fi
exit 0
`

func TestMergeToMain_SavesOtherBranches_WhenGitHubRefusesOne(t *testing.T) {
	w := newWorld(t)
	pc := w.device("pc", "claude/a-workflow")
	writeFile(t, pc, ".github/workflows/extra.yml", "name: extra\n")
	if res := w.sync(pc, "stop", "{}", "GIT_COMMITTER_DATE=2026-11-18T10:00:00Z"); res.code != 0 {
		t.Fatal(res.stderr)
	}
	ipad := w.device("ipad", "claude/b-notes")
	saveFromDevice(w, ipad, "MEMORY.md", "- note that must reach main", "GIT_COMMITTER_DATE=2026-11-18T11:00:00Z")
	w.remoteHook("update", githubRefusesWorkflowEdits)

	res := w.mergeToMain("WORKSPACE_RETRY_DELAY=0")

	if got := w.remoteFile("main", "MEMORY.md"); !strings.Contains(got, "must reach main") {
		t.Errorf("main MEMORY.md = %q, want the iPad note despite the refused branch", got)
	}
	if !w.remoteHasBranch("claude/a-workflow") {
		t.Error("deleted the refused branch; its changes would be lost")
	}
	if w.remoteHasBranch("claude/b-notes") {
		t.Error("kept a branch that was saved to main")
	}
	if res.code == 0 || !strings.Contains(res.stdout+res.stderr, "claude/a-workflow could not be saved to main") {
		t.Errorf("got exit %d, output %q; want a failed run naming the stuck branch", res.code, res.stdout+res.stderr)
	}
	if strings.Contains(res.stdout, "trying again") {
		t.Errorf("output %q: mistook GitHub refusing the push for main moving, and retried for nothing", res.stdout)
	}
}
