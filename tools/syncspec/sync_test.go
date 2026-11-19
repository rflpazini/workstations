package syncspec

import (
	"strings"
	"testing"
)

func TestStop_PushesSessionBranch_WhenFileChanged(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/inbox-1")
	appendLine(t, phone, "MEMORY.md", "- dentist on Tuesday at 3pm")

	res := w.sync(phone, "stop", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := w.remoteFile("claude/inbox-1", "MEMORY.md"); !strings.Contains(got, "dentist on Tuesday") {
		t.Errorf("remote branch MEMORY.md = %q, want the new line", got)
	}
}

func TestStop_DoesNotPush_WhenNothingChanged(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/idle-1")

	res := w.sync(phone, "stop", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if w.remoteHasBranch("claude/idle-1") {
		t.Error("pushed a branch although nothing changed")
	}
}

func TestSync_ExitsZeroQuietly_WhenFolderIsNotAGitRepo(t *testing.T) {
	for _, mode := range []string{"start", "prompt", "stop"} {
		t.Run(mode, func(t *testing.T) {
			w := newWorld(t)
			plain := t.TempDir()
			writeFile(t, plain, "MEMORY.md", "# Memory\n")

			res := w.sync(plain, mode, "{}")

			if res.code != 0 || res.stdout != "" || res.stderr != "" {
				t.Errorf("got exit %d, stdout %q, stderr %q; want exit 0 and no output", res.code, res.stdout, res.stderr)
			}
		})
	}
}

func TestSync_ExitsZeroQuietly_WhenRepoHasNoRemote(t *testing.T) {
	for _, mode := range []string{"start", "prompt", "stop"} {
		t.Run(mode, func(t *testing.T) {
			w := newWorld(t)
			local := t.TempDir()
			git(t, w.env, local, "init", "--quiet", "--initial-branch=main")
			identify(t, w.env, local)
			writeFile(t, local, "MEMORY.md", "# Memory\n")

			res := w.sync(local, mode, "{}")

			if res.code != 0 || res.stdout != "" || res.stderr != "" {
				t.Errorf("got exit %d, stdout %q, stderr %q; want exit 0 and no output", res.code, res.stdout, res.stderr)
			}
		})
	}
}

func TestStop_SavesChange_WhenGitIdentityIsMissing(t *testing.T) {
	w := newWorld(t)
	pc := w.device("pc", "claude/no-identity-1")
	git(t, w.env, pc, "config", "--unset", "user.name")
	git(t, w.env, pc, "config", "--unset", "user.email")
	// Like a container whose hostname has no domain: git cannot guess an identity.
	git(t, w.env, pc, "config", "user.useConfigOnly", "true")
	appendLine(t, pc, "MEMORY.md", "- prefers short answers")

	res := w.sync(pc, "stop", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := w.remoteFile("claude/no-identity-1", "MEMORY.md"); !strings.Contains(got, "short answers") {
		t.Errorf("remote MEMORY.md = %q, want the new line", got)
	}
}

// anotherDeviceSaves simulates a change from another device already merged
// into main by the sync-to-main Action.
func anotherDeviceSaves(w *world, name, file, line string) {
	w.t.Helper()
	other := w.device(name, "main")
	appendLine(w.t, other, file, line)
	git(w.t, w.env, other, "commit", "--quiet", "-am", "from "+name)
	git(w.t, w.env, other, "push", "--quiet", "origin", "main")
}

func TestStart_BringsInChanges_WhenAnotherDeviceSavedToMain(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/morning-1")
	anotherDeviceSaves(w, "phone", "MEMORY.md", "- vet appointment Friday")

	res := w.sync(ipad, "start", `{"source":"startup"}`)

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := readFile(t, ipad, "MEMORY.md"); !strings.Contains(got, "vet appointment Friday") {
		t.Errorf("iPad MEMORY.md = %q, want the phone's line", got)
	}
}

func TestPrompt_SkipsFetch_WhenFetchedLessThanIntervalAgo(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/chat-1")
	w.sync(ipad, "prompt", "{}")
	anotherDeviceSaves(w, "phone", "MEMORY.md", "- buy dog food")

	res := w.sync(ipad, "prompt", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := readFile(t, ipad, "MEMORY.md"); strings.Contains(got, "buy dog food") {
		t.Error("fetched again within the interval; want it skipped")
	}
}

func TestPrompt_BringsInChanges_WhenIntervalHasPassed(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/chat-2")
	w.sync(ipad, "prompt", "{}")
	anotherDeviceSaves(w, "phone", "MEMORY.md", "- buy dog food")

	res := w.sync(ipad, "prompt", "{}", "WORKSPACE_SYNC_INTERVAL=0")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if got := readFile(t, ipad, "MEMORY.md"); !strings.Contains(got, "buy dog food") {
		t.Errorf("iPad MEMORY.md = %q, want the phone's line", got)
	}
}

func TestStart_KeepsUnsavedWork_WhenBringingInChangesToSameFile(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/notes-1")
	appendLine(t, ipad, "MEMORY.md", "- unsaved note from the iPad")
	anotherDeviceSaves(w, "phone", "MEMORY.md", "- saved note from the phone")

	res := w.sync(ipad, "start", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	got := readFile(t, ipad, "MEMORY.md")
	if !strings.Contains(got, "unsaved note from the iPad") || !strings.Contains(got, "saved note from the phone") {
		t.Errorf("iPad MEMORY.md = %q, want both notes", got)
	}
}

func TestStart_WarnsWithoutBlocking_WhenBinaryFileConflicts(t *testing.T) {
	w := newWorld(t)
	ipad := w.device("ipad", "claude/budget-1")
	writeFile(t, ipad, "budget.xlsx", "\x00ipad version\x00")
	other := w.device("pc", "main")
	writeFile(t, other, "budget.xlsx", "\x00pc version\x00")
	git(t, w.env, other, "add", "-A")
	git(t, w.env, other, "commit", "--quiet", "-m", "pc budget")
	git(t, w.env, other, "push", "--quiet", "origin", "main")

	res := w.sync(ipad, "start", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if !strings.Contains(res.stdout, "could not bring in") {
		t.Errorf("stdout = %q, want a warning Claude can relay", res.stdout)
	}
	if run(t, ipad, w.env, "", "git", "rev-parse", "--verify", "--quiet", "MERGE_HEAD").code == 0 {
		t.Error("left the repo in the middle of a merge")
	}
	if got := readFile(t, ipad, "budget.xlsx"); got != "\x00ipad version\x00" {
		t.Errorf("budget.xlsx = %q, want the iPad's own version kept", got)
	}
}

func TestStart_TellsClaudeItSynced_WhenSessionOpens(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/hello-1")

	res := w.sync(phone, "start", `{"source":"startup"}`)

	if !strings.Contains(res.stdout, "Workspace sync:") {
		t.Errorf("stdout = %q, want a one-line sync status for Claude's context", res.stdout)
	}
}

func offline(w *world, dir string) {
	w.t.Helper()
	git(w.t, w.env, dir, "remote", "set-url", "origin", w.base+"/no-such-remote.git")
}

func TestStop_AsksClaudeToWarnUser_WhenPushFails(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/offline-1")
	appendLine(t, phone, "MEMORY.md", "- pay the electricity bill")
	offline(w, phone)

	res := w.sync(phone, "stop", `{"stop_hook_active": false}`)

	if res.code != 2 || !strings.Contains(res.stderr, "NOT saved") {
		t.Errorf("got exit %d, stderr %q; want exit 2 and a NOT saved warning", res.code, res.stderr)
	}
}

func TestStop_DoesNotLoop_WhenStopHookAlreadyActive(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/offline-2")
	appendLine(t, phone, "MEMORY.md", "- pay the electricity bill")
	offline(w, phone)

	res := w.sync(phone, "stop", `{"session_id":"x","stop_hook_active": true}`)

	if res.code != 0 {
		t.Errorf("exit %d; want 0 so Claude is not asked to continue forever", res.code)
	}
}

func TestStart_WarnsWithoutBlocking_WhenGitHubIsUnreachable(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/subway-1")
	offline(w, phone)

	res := w.sync(phone, "start", `{"source":"startup"}`)

	if res.code != 0 || !strings.Contains(res.stdout, "could not reach GitHub") {
		t.Errorf("got exit %d, stdout %q; want exit 0 and a could-not-reach warning", res.code, res.stdout)
	}
}

func TestStop_DoesNotPush_WhenSessionOnlyBroughtInOtherDevicesChanges(t *testing.T) {
	w := newWorld(t)
	phone := w.device("phone", "claude/pulled-1")
	saveFromDevice(w, phone, "MEMORY.md", "- line from the phone")
	w.mergeToMain()
	anotherDeviceSaves(w, "ipad", "MEMORY.md", "- line from the iPad")
	w.sync(phone, "start", "{}")

	res := w.sync(phone, "stop", "{}")

	if res.code != 0 {
		t.Fatalf("exit %d, stderr: %s", res.code, res.stderr)
	}
	if w.remoteHasBranch("claude/pulled-1") {
		t.Error("pushed again although the session had no new work; that wastes an Action run")
	}
}

func TestSync_DoesNothing_WhenRunInsideTheTemplateRepoItself(t *testing.T) {
	w := newWorld(t)
	dev := w.device("maintainer", "claude/template-edit-1")
	const upstream = "https://github.com/rflpazini/workstations.git"
	git(t, w.env, dev, "config", "remote.origin.url", upstream)
	git(t, w.env, dev, "config", "url."+w.remote+".insteadOf", upstream)
	appendLine(t, dev, "MEMORY.md", "- an edit to the template itself")

	res := w.sync(dev, "stop", "{}")

	if res.code != 0 || res.stdout != "" || res.stderr != "" {
		t.Errorf("got exit %d, stdout %q, stderr %q; want a silent no-op", res.code, res.stdout, res.stderr)
	}
	if w.remoteHasBranch("claude/template-edit-1") {
		t.Error("auto-saved inside the template repo; template changes must go through review")
	}
}
