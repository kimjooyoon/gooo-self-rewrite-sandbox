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

func TestLoadPhaseRejectsDuplicateTerminalReason(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "phase.gooo")
	source := strings.Join([]string{
		"program test",
		"namespace test",
		"phase plan",
		"activity A(Input) -> Output computes \"noop\"",
		"terminal_reason CLOSED \"first\"",
		"terminal_reason CLOSED \"second\"",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
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
