package gooo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTypedRewriteParsing(t *testing.T) {
	value, ok := parseRewrite(`replace-operation LowerSource SourceGraph SemanticIR "lower-source:v2"`)
	if !ok || value.Kind != "replace-operation" || value.Name != "LowerSource" || value.InputType != "SourceGraph" || value.OutputType != "SemanticIR" || value.Program != "lower-source:v2" {
		t.Fatalf("unexpected typed rewrite: %#v, %v", value, ok)
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

func TestLoadPhaseRejectsUnknownDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phase.gooo")
	data := `program gooo-self-rewrite-sandbox v1
namespace self_rewrite
phase reflexive.self-rewrite.v1
unknown_phase_field value
activity LowerSource(SourceGraph) -> SemanticIR computes "lower-source:v1"
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPhase(path); err == nil {
		t.Fatal("unknown phase declaration was accepted")
	}
}

func TestLoadCandidateRejectsUnknownDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidate.gooo")
	data := `program gooo-self-rewrite-sandbox v1
candidate candidate-one
target_phase reflexive.self-rewrite.v1
target_activity LowerSource
boundary N+1
rewrite noop
unknown_candidate_field value
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCandidate(path); err == nil {
		t.Fatal("unknown candidate declaration was accepted")
	}
}
