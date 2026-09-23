package gooo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPhaseRejectsUnknownDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phase.gooo")
	data := []byte("program p\nnamespace n\nphase s\nactivity read(int) -> int computes \"x\"\nunknown_typo value\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPhase(path); err == nil {
		t.Fatal("phase loader accepted an unknown declaration")
	}
}

func TestLoadCandidateRejectsUnknownDeclaration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "candidate.gooo")
	data := []byte("program p\ncandidate c\ntarget_phase s\ntarget_activity read\nrewrite noop\nboundary caller-owned\nunknown_typo value\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCandidate(path); err == nil {
		t.Fatal("candidate loader accepted an unknown declaration")
	}
}
