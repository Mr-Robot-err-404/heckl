package reviewer

import (
	"fmt"
	"strings"
	"testing"
)

func ptr(n int) *int { return &n }

func show(n *int) string {
	if n == nil {
		return "<nil>"
	}
	return fmt.Sprint(*n)
}

func render(lines []diffLine) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, fmt.Sprintf("%s:%d:%s", l.Side, l.Number, l.Text))
	}
	return out
}

func assertLines(t *testing.T, got []diffLine, want []string) {
	t.Helper()
	have := render(got)
	if len(have) != len(want) {
		t.Fatalf("line count = %d, want %d\ngot:\n  %s", len(have), len(want), strings.Join(have, "\n  "))
	}
	for i := range want {
		if have[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, have[i], want[i])
		}
	}
}

const modifiedFile = `diff --git a/internal/app/server.go b/internal/app/server.go
index 1111111..2222222 100644
--- a/internal/app/server.go
+++ b/internal/app/server.go
@@ -10,6 +10,7 @@ func setup() {
 	mux := http.NewServeMux()
 	mux.Handle("/api", api)
-	log.Print("old")
+	log.Print("new")
+	log.Print("extra")
 	return mux
 }
`

func TestParseDiffIndexCounters(t *testing.T) {
	idx := parseDiffIndex(modifiedFile)

	lines, ok := idx["internal/app/server.go"]
	if !ok {
		t.Fatalf("file not indexed, got keys %v", keys(idx))
	}

	assertLines(t, lines, []string{
		"additions:10:\tmux := http.NewServeMux()",
		"additions:11:\tmux.Handle(\"/api\", api)",
		"deletions:12:\tlog.Print(\"old\")",
		"additions:12:\tlog.Print(\"new\")",
		"additions:13:\tlog.Print(\"extra\")",
		"additions:14:\treturn mux",
		"additions:15:}",
	})
}

func TestParseDiffIndexOmittedHunkCounts(t *testing.T) {
	diff := `diff --git a/one.txt b/one.txt
--- a/one.txt
+++ b/one.txt
@@ -1 +1 @@
-alpha
+beta
`
	assertLines(t, parseDiffIndex(diff)["one.txt"], []string{
		"deletions:1:alpha",
		"additions:1:beta",
	})
}

func TestParseDiffIndexNewFile(t *testing.T) {
	diff := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/new.go
@@ -0,0 +1,2 @@
+package main
+
`
	assertLines(t, parseDiffIndex(diff)["new.go"], []string{
		"additions:1:package main",
		"additions:2:",
	})
}

func TestParseDiffIndexDeletedFileIsNotIndexed(t *testing.T) {
	diff := `diff --git a/gone.go b/gone.go
deleted file mode 100644
--- a/gone.go
+++ /dev/null
@@ -1,2 +0,0 @@
-package main
-// bye
`
	idx := parseDiffIndex(diff)
	if len(idx) != 0 {
		t.Fatalf("deleted file produced index entries: %v", keys(idx))
	}

	c := idx.resolve(Concern{File: "gone.go", Line: ptr(1), Side: SideDeletions, Anchor: "package main"})
	if c.Line != nil || c.Side != "" {
		t.Errorf("want file-level degradation, got line=%s side=%q", show(c.Line), c.Side)
	}
}

func TestParseDiffIndexHunkHeaderTrailingContext(t *testing.T) {
	diff := `diff --git a/gen.go b/gen.go
--- a/gen.go
+++ b/gen.go
@@ -20,3 +20,3 @@ // +build ignore
 keep
-old
+new
`
	assertLines(t, parseDiffIndex(diff)["gen.go"], []string{
		"additions:20:keep",
		"deletions:21:old",
		"additions:21:new",
	})
}

func TestResolveAnchorWins(t *testing.T) {
	idx := parseDiffIndex(modifiedFile)

	c := idx.resolve(Concern{
		File:   "internal/app/server.go",
		Line:   ptr(99),
		Anchor: `log.Print("extra")`,
	})

	if c.Line == nil || *c.Line != 13 {
		t.Fatalf("line = %s, want 13", show(c.Line))
	}
	if c.Side != SideAdditions {
		t.Errorf("side = %q, want %q", c.Side, SideAdditions)
	}
}

func TestResolveAnchorPrefersProximity(t *testing.T) {
	diff := `diff --git a/dup.go b/dup.go
--- a/dup.go
+++ b/dup.go
@@ -1,2 +1,3 @@
+	total++
 	x := 1
 	y := 2
@@ -40,2 +41,3 @@
+	total++
 	z := 3
`
	idx := parseDiffIndex(diff)

	near := idx.resolve(Concern{File: "dup.go", Line: ptr(41), Anchor: "total++"})
	if near.Line == nil || *near.Line != 41 {
		t.Fatalf("line = %s, want 41 (nearest match to claim)", show(near.Line))
	}
}

func TestResolveShortDiffLineDoesNotHijackAnchor(t *testing.T) {
	diff := `diff --git a/hijack.go b/hijack.go
--- a/hijack.go
+++ b/hijack.go
@@ -1,0 +1,6 @@
+	}
+
+func run() error {
+	if err := doThing(); err != nil {
+		opts := Options{Retries: 3}
+	}
`
	idx := parseDiffIndex(diff)

	c := idx.resolve(Concern{File: "hijack.go", Anchor: `opts := Options{Retries: 3}`})
	if c.Line == nil || *c.Line != 5 {
		t.Fatalf("line = %s, want 5 - a bare %q line must not swallow a longer anchor", show(c.Line), "}")
	}
}

func TestResolveExactMatchBeatsSubstringMatch(t *testing.T) {
	diff := `diff --git a/rank.go b/rank.go
--- a/rank.go
+++ b/rank.go
@@ -1,0 +1,3 @@
+	wrapper(doThing())
+	other()
+	doThing()
`
	idx := parseDiffIndex(diff)

	c := idx.resolve(Concern{File: "rank.go", Anchor: "doThing()"})
	if c.Line == nil || *c.Line != 3 {
		t.Fatalf("line = %s, want 3 - exact match must outrank a containing line", show(c.Line))
	}
}

func TestResolveDegenerateAnchorIsIgnored(t *testing.T) {
	diff := `diff --git a/brace.go b/brace.go
--- a/brace.go
+++ b/brace.go
@@ -1,0 +1,4 @@
+}
+}
+	value := compute()
+}
`
	idx := parseDiffIndex(diff)

	c := idx.resolve(Concern{File: "brace.go", Line: ptr(4), Side: SideAdditions, Anchor: "}"})
	if c.Line == nil || *c.Line != 4 {
		t.Fatalf("line = %s, want 4 - a non-discriminating anchor must defer to the claimed line", show(c.Line))
	}
}

func TestParseDiffIndexSkipsCombinedHunkHeader(t *testing.T) {
	diff := `diff --cc merged.go
--- a/merged.go
+++ b/merged.go
@@@ -1,2 -1,2 +1,3 @@@
 keep
++conflict
`
	if lines := parseDiffIndex(diff)["merged.go"]; len(lines) != 0 {
		t.Errorf("combined hunk produced lines: %v", render(lines))
	}
}

func TestResolveFallsBackToClaimedLine(t *testing.T) {
	idx := parseDiffIndex(modifiedFile)

	c := idx.resolve(Concern{
		File:   "internal/app/server.go",
		Line:   ptr(12),
		Side:   SideDeletions,
		Anchor: "this text is nowhere in the diff",
	})

	if c.Line == nil || *c.Line != 12 {
		t.Fatalf("line = %s, want 12", show(c.Line))
	}
	if c.Side != SideDeletions {
		t.Errorf("side = %q, want %q", c.Side, SideDeletions)
	}
}

func TestResolveDegradesWhenClaimedLineIsOutsideDiff(t *testing.T) {
	idx := parseDiffIndex(modifiedFile)

	c := idx.resolve(Concern{File: "internal/app/server.go", Line: ptr(400), Side: SideAdditions})
	if c.Line != nil || c.Side != "" {
		t.Errorf("want file-level, got line=%s side=%q", show(c.Line), c.Side)
	}
	if c.File != "internal/app/server.go" {
		t.Errorf("file = %q, want it preserved", c.File)
	}
}

func TestResolveUnknownFileDegrades(t *testing.T) {
	idx := parseDiffIndex(modifiedFile)

	c := idx.resolve(Concern{File: "nowhere/at/all.go", Line: ptr(3), Anchor: "return mux"})
	if c.Line != nil || c.Side != "" {
		t.Errorf("want file-level, got line=%s side=%q", show(c.Line), c.Side)
	}
}

func TestLookupFileMatchesBySuffix(t *testing.T) {
	idx := parseDiffIndex(modifiedFile)

	name, lines, ok := idx.lookupFile("app/server.go")
	if !ok {
		t.Fatalf("suffix lookup failed")
	}
	if name != "internal/app/server.go" {
		t.Errorf("name = %q, want canonical path", name)
	}
	if len(lines) == 0 {
		t.Error("no lines returned")
	}
}

func keys(idx diffIndex) []string {
	out := make([]string, 0, len(idx))
	for k := range idx {
		out = append(out, k)
	}
	return out
}
