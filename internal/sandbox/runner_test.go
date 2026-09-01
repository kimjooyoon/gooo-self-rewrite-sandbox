package sandbox

import (
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
