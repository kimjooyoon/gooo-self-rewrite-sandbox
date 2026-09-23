package gooo

import (
	"os"
	"strings"
	"testing"
)

func TestLoadPhaseRejectsDuplicateTerminalReasonsAndActivities(t *testing.T) {
	path := t.TempDir() + "/phase.gooo"
	raw := "program p\nnamespace n\nphase phase.v1\nterminal_reason CLOSED \"first\"\nterminal_reason CLOSED \"second\"\nactivity Lower(SourceGraph) -> SemanticIR computes \"lower\"\nactivity Lower(SourceGraph) -> SemanticIR computes \"lower-again\"\n"
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPhase(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate terminal_reason CLOSED") {
		t.Fatalf("LoadPhase duplicate declaration error = %v", err)
	}
}
