package gooo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypedRewriteParsing(t *testing.T) {
	value, ok := parseRewrite(`replace-operation LowerSource SourceGraph SemanticIR "lower-source:v2"`)
	if !ok || value.Kind != "replace-operation" || value.Name != "LowerSource" || value.InputType != "SourceGraph" || value.OutputType != "SemanticIR" || value.Program != "lower-source:v2" {
		t.Fatalf("unexpected typed rewrite: %#v, %v", value, ok)
	}
}

func TestLoadPhaseRejectsDuplicateTerminalReasons(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phase.gooo")
	data := []byte("program self-rewrite\nnamespace example\nphase baseline\nactivity Compile(SourceGraph) -> SemanticIR computes \"compile\"\nterminal_reason closed \"first\"\nterminal_reason closed \"second\"\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPhase(path); err == nil || !strings.Contains(err.Error(), "duplicate terminal_reason") {
		t.Fatalf("expected duplicate terminal_reason error, got %v", err)
	}
}

func TestSnapshotDirIsStable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "value.txt"), []byte("value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	firstDigest, first, err := SnapshotDir(root)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, second, err := SnapshotDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != secondDigest || len(first) != 1 || len(second) != 1 || first[0] != second[0] {
		t.Fatalf("snapshot changed: %s/%#v vs %s/%#v", firstDigest, first, secondDigest, second)
	}
}
