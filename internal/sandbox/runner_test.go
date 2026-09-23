package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kimjooyoon/gooo-self-rewrite-sandbox/internal/model"
)

func TestDecisionPrecedence(t *testing.T) {
	decision := model.Reduce([]model.Unknown{{Reason: "unknown"}}, []model.Refutation{{Reason: "refuted"}})
	if decision != model.DecisionRefuted {
		t.Fatalf("decision precedence = %s", decision)
	}
}

func TestByteReplayComparison(t *testing.T) {
	stage := model.StageResult{Decision: model.DecisionClosed, SemanticDigest: "semantic", GeneratedDigest: "generated", TerminalDigest: "terminal"}
	decision, unknowns, refutations := compareStages("byte-identical-replay", stage, stage)
	if decision != model.DecisionClosed || len(unknowns) != 0 || len(refutations) != 0 {
		t.Fatalf("unexpected replay comparison: %s %#v %#v", decision, unknowns, refutations)
	}
}

func TestEnsureInsideRejectsSymlinkOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside-phase.gooo")
	if err := os.WriteFile(outside, []byte("phase"), 0o600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outside)
	link := filepath.Join(root, "phase.gooo")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := ensureInside(root, link); err == nil {
		t.Fatal("symlink outside immutable input root was accepted")
	}
}
