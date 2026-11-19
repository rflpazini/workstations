package syncspec

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Line coverage for the bash scripts, enforced with:
//
//	SYNCSPEC_COVERAGE=1 go test ./...
//
// Every bash run is traced (see traceBash) and each script must reach 80%.
const minCoverage = 80.0

func TestMain(m *testing.M) {
	if os.Getenv("SYNCSPEC_COVERAGE") == "" {
		os.Exit(m.Run())
	}
	dir, err := os.MkdirTemp("", "syncspec-trace")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	trace := filepath.Join(dir, "trace.log")
	os.Setenv("SYNCSPEC_TRACE", trace)
	code := m.Run()
	if code == 0 {
		code = reportCoverage(os.Stdout, trace, syncScript(), mergeScript())
	}
	os.RemoveAll(dir)
	os.Exit(code)
}

var (
	structuralLine = regexp.MustCompile(`^(\}|fi|done|esac|;;|else|then|do|\{|EOF|done <<EOF|[\w-]+\(\) \{|case .* in)$`)
	caseLabel      = regexp.MustCompile(`^[\w|-]+\)$`)
	traceLine      = regexp.MustCompile(`^\++(/[^:]+):(\d+): `)
)

// executableLines returns the lines of a bash script that xtrace reports when
// they run: no blanks, comments, heredoc bodies, case labels, declarations or
// lone keywords.
func executableLines(src string) map[int]string {
	lines := map[int]string{}
	inHeredoc := false
	for i, raw := range strings.Split(src, "\n") {
		s := strings.TrimSpace(raw)
		if inHeredoc {
			inHeredoc = s != "EOF"
			continue
		}
		if strings.HasSuffix(s, "<<EOF") {
			inHeredoc = true
		}
		if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "local ") ||
			caseLabel.MatchString(s) || structuralLine.MatchString(s) {
			continue
		}
		lines[i+1] = s
	}
	return lines
}

func hitLines(r io.Reader) map[string]map[int]bool {
	hits := map[string]map[int]bool{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		m := traceLine.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[2])
		if hits[m[1]] == nil {
			hits[m[1]] = map[int]bool{}
		}
		hits[m[1]][n] = true
	}
	return hits
}

func reportCoverage(out io.Writer, trace string, scripts ...string) int {
	f, err := os.Open(trace)
	if err != nil {
		fmt.Fprintln(out, "no bash trace recorded:", err)
		return 1
	}
	defer f.Close()
	hits := hitLines(f)

	code := 0
	fmt.Fprintf(out, "\n%-40s %6s %5s %9s  %s\n", "File", "Lines", "Hit", "Coverage", "Status")
	for _, script := range scripts {
		src, err := os.ReadFile(script)
		if err != nil {
			fmt.Fprintln(out, err)
			return 1
		}
		exe := executableLines(string(src))
		var missed []int
		for n := range exe {
			if !hits[script][n] {
				missed = append(missed, n)
			}
		}
		sort.Ints(missed)
		pct := 100 * float64(len(exe)-len(missed)) / float64(len(exe))
		status := "PASS"
		if pct < minCoverage {
			status, code = "FAIL", 1
		}
		rel, _ := filepath.Rel(repoRoot, script)
		fmt.Fprintf(out, "%-40s %6d %5d %8.1f%%  %s\n", rel, len(exe), len(exe)-len(missed), pct, status)
		for _, n := range missed {
			fmt.Fprintf(out, "    missed line %d: %s\n", n, exe[n])
		}
	}
	return code
}

func TestExecutableLines_CountsOnlyCommandsThatTraceReports(t *testing.T) {
	src := `#!/usr/bin/env bash
# comment
set -u

greet() {
  local name="$1"
  if [ -n "$name" ]; then
    echo "hi $name"
  fi
}

case "$1" in
  start)
    greet "$2"
    ;;
esac
cat <<EOF
heredoc body
EOF
`
	got := executableLines(src)

	want := []int{3, 7, 8, 14, 17}
	if len(got) != len(want) {
		t.Fatalf("got lines %v, want %v", got, want)
	}
	for _, n := range want {
		if _, ok := got[n]; !ok {
			t.Errorf("line %d missing from %v", n, got)
		}
	}
}
