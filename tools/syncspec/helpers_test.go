// Package syncspec tests the scripts that keep a workspace in sync with GitHub:
// .claude/hooks/sync.sh (run by Claude Code hooks on every device) and
// .github/scripts/merge-to-main.sh (run by the sync-to-main GitHub Action).
// Every test builds real git repositories in a temp dir: a bare "GitHub" remote
// and one clone per device.
package syncspec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var repoRoot = mustAbs("../..")

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}
	return abs
}

func syncScript() string  { return filepath.Join(repoRoot, ".claude", "hooks", "sync.sh") }
func mergeScript() string { return filepath.Join(repoRoot, ".github", "scripts", "merge-to-main.sh") }

type result struct {
	stdout string
	stderr string
	code   int
}

// isolatedEnv keeps the developer's own git config (signing, credential
// helpers, default branch) out of the tests.
func isolatedEnv(t *testing.T) []string {
	t.Helper()
	home := t.TempDir()
	env := []string{
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + home,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + filepath.Join(home, "gitconfig"),
		"GIT_TERMINAL_PROMPT=0",
	}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") || strings.HasPrefix(kv, "TMPDIR=") {
			env = append(env, kv)
		}
	}
	return env
}

func run(t *testing.T, dir string, env []string, stdin string, name string, args ...string) result {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	traceBash(t, cmd, name)
	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("running %s %v: %v", name, args, err)
	}
	return result{stdout: out.String(), stderr: errOut.String(), code: code}
}

func git(t *testing.T, env []string, dir string, args ...string) string {
	t.Helper()
	res := run(t, dir, env, "", "git", args...)
	if res.code != 0 {
		t.Fatalf("git %v in %s failed (%d): %s", args, dir, res.code, res.stderr)
	}
	return strings.TrimSpace(res.stdout)
}

// world is one test's universe: a bare remote standing in for GitHub and the
// environment every command runs with.
type world struct {
	t      *testing.T
	env    []string
	base   string
	remote string
	runs   int
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{t: t, env: isolatedEnv(t), base: t.TempDir()}
	w.remote = filepath.Join(w.base, "remote.git")
	git(t, w.env, w.base, "init", "--quiet", "--bare", "--initial-branch=main", w.remote)

	seed := filepath.Join(w.base, "seed")
	git(t, w.env, w.base, "init", "--quiet", "--initial-branch=main", seed)
	identify(t, w.env, seed)
	attrs, err := os.ReadFile(filepath.Join(repoRoot, ".gitattributes"))
	if err != nil {
		t.Fatalf("reading the repo .gitattributes: %v", err)
	}
	writeFile(t, seed, ".gitattributes", string(attrs))
	ws, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "workspace.json"))
	if err != nil {
		t.Fatalf("reading .claude/workspace.json: %v", err)
	}
	writeFile(t, seed, ".claude/workspace.json", string(ws))
	writeFile(t, seed, "MEMORY.md", "# Memory\n\n")
	git(t, w.env, seed, "add", "-A")
	git(t, w.env, seed, "commit", "--quiet", "-m", "seed")
	git(t, w.env, seed, "remote", "add", "origin", w.remote)
	git(t, w.env, seed, "push", "--quiet", "origin", "main")
	return w
}

func identify(t *testing.T, env []string, dir string) {
	t.Helper()
	git(t, env, dir, "config", "user.name", "Test Device")
	git(t, env, dir, "config", "user.email", "device@example.com")
}

// device clones the remote the way a Claude Code session does and checks out
// the session branch (use "main" to stay on main).
func (w *world) device(name, branch string) string {
	w.t.Helper()
	dir := filepath.Join(w.base, name)
	git(w.t, w.env, w.base, "clone", "--quiet", w.remote, dir)
	identify(w.t, w.env, dir)
	if branch != "main" {
		git(w.t, w.env, dir, "checkout", "--quiet", "-b", branch)
	}
	return dir
}

func (w *world) sync(dir, mode, stdin string, extraEnv ...string) result {
	w.t.Helper()
	env := append(append([]string{}, w.env...), "CLAUDE_PROJECT_DIR="+dir)
	env = append(env, extraEnv...)
	return run(w.t, dir, env, stdin, "bash", syncScript(), mode)
}

// remoteFile reads a file as GitHub would show it on a branch. Returns "" when
// the branch or the file does not exist.
func (w *world) remoteFile(branch, path string) string {
	w.t.Helper()
	res := run(w.t, w.base, w.env, "", "git", "--git-dir", w.remote, "show", branch+":"+path)
	if res.code != 0 {
		return ""
	}
	return res.stdout
}

func (w *world) remoteHasBranch(branch string) bool {
	w.t.Helper()
	res := run(w.t, w.base, w.env, "", "git", "--git-dir", w.remote, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return res.code == 0
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendLine(t *testing.T, dir, name, line string) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// mergeToMain runs merge-to-main.sh the way the GitHub Action does: in a fresh
// full clone with no git identity configured.
func (w *world) mergeToMain(extraEnv ...string) result {
	w.t.Helper()
	w.runs++
	dir := filepath.Join(w.base, fmt.Sprintf("runner-%d", w.runs))
	git(w.t, w.env, w.base, "clone", "--quiet", w.remote, dir)
	git(w.t, w.env, dir, "config", "user.useConfigOnly", "true")
	env := append(append([]string{}, w.env...), extraEnv...)
	return run(w.t, dir, env, "", "bash", mergeScript())
}

// remoteHook installs a server-side git hook in the fake GitHub remote.
func (w *world) remoteHook(name, script string) {
	w.t.Helper()
	if err := os.WriteFile(filepath.Join(w.remote, "hooks", name), []byte(script), 0o755); err != nil {
		w.t.Fatal(err)
	}
}

func removeFile(dir, name string) error { return os.Remove(filepath.Join(dir, name)) }

// traceBash measures line coverage of the bash scripts. With SYNCSPEC_TRACE
// set to a file path, every bash command run by a test appends
// "+<script>:<line>:" to that file through a dedicated fd, so the scripts'
// stdout and stderr stay exactly as the assertions expect.
func traceBash(t *testing.T, cmd *exec.Cmd, name string) {
	t.Helper()
	path := os.Getenv("SYNCSPEC_TRACE")
	if path == "" || name != "bash" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("opening trace file: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	// BASH_ENV turns tracing on only in bash running a script; exporting
	// SHELLOPTS would leak into the sh processes git starts and break them.
	rc := filepath.Join(t.TempDir(), "trace.rc")
	if err := os.WriteFile(rc, []byte("set -x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd.ExtraFiles = []*os.File{f}
	cmd.Env = append(cmd.Env, "BASH_XTRACEFD=3", "BASH_ENV="+rc, `PS4=+${BASH_SOURCE:-}:${LINENO:-}: `)
}
