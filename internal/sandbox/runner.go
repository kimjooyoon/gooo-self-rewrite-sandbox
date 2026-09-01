package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-self-rewrite-sandbox/internal/gooo"
	"github.com/kimjooyoon/gooo-self-rewrite-sandbox/internal/model"
)

type Options struct {
	MetaPath   string
	InputRoot  string
	PhasePath  string
	SourcePath string
	CorpusPath string
	WorkDir    string
	ReportPath string
}

type semanticIR struct {
	Schema       string              `json:"schema"`
	Namespace    string              `json:"namespace"`
	Declarations []model.Declaration `json:"declarations"`
	SourceDigest string              `json:"source_digest"`
	Terminal     model.Terminal      `json:"terminal"`
}

func Run(options Options) (model.Report, error) {
	paths, err := absolutePaths(options)
	if err != nil {
		return model.Report{}, err
	}
	meta, err := gooo.LoadPhase(paths.MetaPath)
	if err != nil {
		return model.Report{}, fmt.Errorf("load meta phase: %w", err)
	}
	baselinePhase, err := gooo.LoadPhase(paths.PhasePath)
	if err != nil {
		return model.Report{}, fmt.Errorf("load baseline phase: %w", err)
	}
	corpusData, err := os.ReadFile(paths.CorpusPath)
	if err != nil {
		return model.Report{}, err
	}
	var corpus model.Corpus
	if err := json.Unmarshal(corpusData, &corpus); err != nil {
		return model.Report{}, fmt.Errorf("decode corpus: %w", err)
	}
	if err := validateCorpus(corpus); err != nil {
		return model.Report{}, err
	}
	if err := ensureInside(paths.InputRoot, paths.PhasePath); err != nil {
		return model.Report{}, err
	}
	if err := ensureInside(paths.InputRoot, paths.SourcePath); err != nil {
		return model.Report{}, err
	}
	if err := ensureOutside(paths.InputRoot, paths.WorkDir); err != nil {
		return model.Report{}, fmt.Errorf("work directory must be outside immutable input: %w", err)
	}
	if err := ensureOutside(paths.InputRoot, paths.ReportPath); err != nil {
		return model.Report{}, fmt.Errorf("report path must be outside immutable input: %w", err)
	}
	if err := os.MkdirAll(paths.WorkDir, 0o755); err != nil {
		return model.Report{}, err
	}
	inputDigest, beforeSnapshot, err := gooo.SnapshotDir(paths.InputRoot)
	if err != nil {
		return model.Report{}, fmt.Errorf("snapshot immutable input: %w", err)
	}
	contractDigest, err := gooo.DigestJSON(meta)
	if err != nil {
		return model.Report{}, err
	}
	corpusDigest := gooo.DigestBytes(corpusData)

	report := model.Report{
		Schema:                   "gooo/self-rewrite-actuation-report/v1",
		Decision:                 model.DecisionClosed,
		InputRoot:                paths.InputRoot,
		InputDigest:              inputDigest,
		ContractDigest:           contractDigest,
		Toolchain:                runtime.Version(),
		RepositoryWrites:         0,
		AutomaticCommitPushMerge: 0,
		Cases:                    []model.CaseResult{},
		UnknownFields:            []string{"stage", "step", "reason", "unknown_class", "next_operation", "blocked_by"},
		Precedence:               []string{model.DecisionRefuted, model.DecisionUnknown, model.DecisionClosed},
	}

	for _, corpusCase := range corpus.Cases {
		caseResult, runErr := runCase(paths, meta, baselinePhase, corpusCase, inputDigest, contractDigest, corpusDigest)
		if runErr != nil {
			return model.Report{}, fmt.Errorf("case %s: %w", corpusCase.ID, runErr)
		}
		report.Cases = append(report.Cases, caseResult)
	}
	afterDigest, afterSnapshot, err := gooo.SnapshotDir(paths.InputRoot)
	if err != nil {
		return model.Report{}, fmt.Errorf("snapshot immutable input after actuation: %w", err)
	}
	if !snapshotsEqual(beforeSnapshot, afterSnapshot) || afterDigest != inputDigest {
		report.RepositoryWrites = 1
	}
	if report.RepositoryWrites != 0 {
		report.Decision = model.DecisionRefuted
	}
	for _, caseResult := range report.Cases {
		if !caseResult.Match {
			report.Decision = model.DecisionRefuted
			break
		}
	}
	report.GeneratedArtifactsCount, report.GeneratedArtifactsBytes = generatedTotals(report.Cases)
	if err := writeReport(paths.ReportPath, report); err != nil {
		return model.Report{}, err
	}
	return report, nil
}

type resolvedOptions struct {
	MetaPath   string
	InputRoot  string
	PhasePath  string
	SourcePath string
	CorpusPath string
	WorkDir    string
	ReportPath string
}

func absolutePaths(options Options) (resolvedOptions, error) {
	values := []string{options.MetaPath, options.InputRoot, options.PhasePath, options.SourcePath, options.CorpusPath, options.WorkDir, options.ReportPath}
	for _, value := range values {
		if value == "" {
			return resolvedOptions{}, errors.New("meta, input-root, phase, source, corpus, work, and report paths are required")
		}
	}
	absolute := func(value string) (string, error) { return filepath.Abs(value) }
	metaPath, err := absolute(options.MetaPath)
	if err != nil {
		return resolvedOptions{}, err
	}
	inputRoot, err := absolute(options.InputRoot)
	if err != nil {
		return resolvedOptions{}, err
	}
	phasePath, err := absolute(options.PhasePath)
	if err != nil {
		return resolvedOptions{}, err
	}
	sourcePath, err := absolute(options.SourcePath)
	if err != nil {
		return resolvedOptions{}, err
	}
	corpusPath, err := absolute(options.CorpusPath)
	if err != nil {
		return resolvedOptions{}, err
	}
	workDir, err := absolute(options.WorkDir)
	if err != nil {
		return resolvedOptions{}, err
	}
	reportPath, err := absolute(options.ReportPath)
	if err != nil {
		return resolvedOptions{}, err
	}
	return resolvedOptions{metaPath, inputRoot, phasePath, sourcePath, corpusPath, workDir, reportPath}, nil
}

func validateCorpus(corpus model.Corpus) error {
	if corpus.Schema != "gooo/self-rewrite-corpus/v1" {
		return fmt.Errorf("unexpected corpus schema %q", corpus.Schema)
	}
	if len(corpus.Cases) != 6 {
		return fmt.Errorf("fixed corpus must contain exactly 6 cases, got %d", len(corpus.Cases))
	}
	seen := map[string]bool{}
	for _, item := range corpus.Cases {
		if item.ID == "" || item.Candidate == "" || item.Expected == "" || item.Kind == "" {
			return errors.New("every fixed case requires id, candidate, expected, and kind")
		}
		if seen[item.ID] {
			return fmt.Errorf("duplicate fixed case %q", item.ID)
		}
		seen[item.ID] = true
		if item.Expected != model.DecisionClosed && item.Expected != model.DecisionUnknown && item.Expected != model.DecisionRefuted {
			return fmt.Errorf("invalid expected decision for %s", item.ID)
		}
	}
	return nil
}

func runCase(paths resolvedOptions, meta model.Phase, baselinePhase model.Phase, corpusCase model.CorpusCase, inputDigest, contractDigest, corpusDigest string) (model.CaseResult, error) {
	caseDir := filepath.Join(paths.WorkDir, "cases", safeName(corpusCase.ID))
	baselineDir := filepath.Join(caseDir, "stage-n")
	if err := os.MkdirAll(baselineDir, 0o755); err != nil {
		return model.CaseResult{}, err
	}
	baseline, err := compileStage(meta, baselinePhase, paths.SourcePath, baselineDir, "N")
	if err != nil {
		return model.CaseResult{}, err
	}
	caseResult := model.CaseResult{
		ID: corpusCase.ID, Kind: corpusCase.Kind, Expected: corpusCase.Expected,
		Decision: baseline.Decision, CandidatePath: corpusCase.Candidate,
		Baseline: baseline, Unknowns: []model.Unknown{}, Refutations: []model.Refutation{},
	}
	candidatePath := corpusCase.Candidate
	if !filepath.IsAbs(candidatePath) {
		repositoryRoot := filepath.Dir(filepath.Dir(filepath.Dir(paths.CorpusPath)))
		candidatePath = filepath.Join(repositoryRoot, candidatePath)
	}
	candidate, err := gooo.LoadCandidate(candidatePath)
	if err != nil {
		return model.CaseResult{}, fmt.Errorf("load candidate %s: %w", candidatePath, err)
	}
	caseResult.CandidatePath = candidatePath
	caseResult.CandidateDigest = candidate.SourceDigest
	unknowns, refutations := validateCandidate(meta, baselinePhase, candidate)
	caseResult.Unknowns = append(caseResult.Unknowns, unknowns...)
	caseResult.Refutations = append(caseResult.Refutations, refutations...)
	if len(unknowns) == 0 && len(refutations) == 0 {
		candidateInput := filepath.Join(caseDir, "candidate-input")
		if err := gooo.CopyTree(paths.InputRoot, candidateInput); err != nil {
			return model.CaseResult{}, fmt.Errorf("copy immutable input to caller-owned temp: %w", err)
		}
		relativePhase, err := filepath.Rel(paths.InputRoot, paths.PhasePath)
		if err != nil {
			return model.CaseResult{}, err
		}
		candidatePhasePath := filepath.Join(candidateInput, relativePhase)
		candidateBytes, err := gooo.ApplyCandidate(candidatePhasePath, candidate)
		if err != nil {
			return model.CaseResult{}, fmt.Errorf("apply typed candidate: %w", err)
		}
		if err := os.WriteFile(candidatePhasePath, candidateBytes, 0o644); err != nil {
			return model.CaseResult{}, fmt.Errorf("write candidate only to temp copy: %w", err)
		}
		candidatePhase, err := gooo.LoadPhase(candidatePhasePath)
		if err != nil {
			return model.CaseResult{}, fmt.Errorf("load applied candidate phase: %w", err)
		}
		candidateDir := filepath.Join(caseDir, "stage-n-plus-1")
		candidateResult, compileErr := compileStage(meta, candidatePhase, filepath.Join(candidateInput, mustRelative(paths.InputRoot, paths.SourcePath)), candidateDir, "N+1")
		if compileErr != nil {
			return model.CaseResult{}, compileErr
		}
		caseResult.Candidate = &candidateResult
		caseResult.CandidateApplied = true
		comparisonDecision, comparisonUnknowns, comparisonRefutations := compareStages(corpusCase.Kind, baseline, candidateResult)
		if comparisonDecision == model.DecisionUnknown && len(comparisonUnknowns) == 0 {
			comparisonUnknowns = append(comparisonUnknowns, model.Unknown{
				Stage: "compare", Step: "compare-candidate", Reason: "COMPARISON_INCOMPLETE",
				UnknownClass: "COMPARISON_INCOMPLETE", NextOperation: "capture-comparison-evidence",
				BlockedBy: []string{"candidate-artifacts"},
			})
		}
		if comparisonDecision == model.DecisionRefuted && len(comparisonRefutations) == 0 {
			comparisonRefutations = append(comparisonRefutations, model.Refutation{
				Stage: "compare", Step: "compare-candidate", Reason: "CANDIDATE_REFUTED", Counterexample: candidate.ID,
			})
		}
		caseResult.Decision = model.Reduce(append(append([]model.Unknown{}, caseResult.Unknowns...), comparisonUnknowns...), append(append([]model.Refutation{}, caseResult.Refutations...), comparisonRefutations...))
		caseResult.Unknowns = append(caseResult.Unknowns, comparisonUnknowns...)
		caseResult.Refutations = append(caseResult.Refutations, comparisonRefutations...)
		caseResult.Match = caseResult.Decision == corpusCase.Expected
		caseResult.Improvement = makeImprovement(corpusCase.ID, inputDigest, contractDigest, runtime.Version(), baseline, candidateResult, caseResult.Decision)
		return caseResult, nil
	}
	caseResult.Decision = model.Reduce(caseResult.Unknowns, caseResult.Refutations)
	caseResult.Match = caseResult.Decision == corpusCase.Expected
	caseResult.Improvement = blockedImprovement(corpusCase.ID, inputDigest, contractDigest, runtime.Version(), caseResult.Unknowns, caseResult.Refutations)
	return caseResult, nil
}

func validateCandidate(meta model.Phase, phase model.Phase, candidate model.Candidate) ([]model.Unknown, []model.Refutation) {
	unknowns := []model.Unknown{}
	refutations := []model.Refutation{}
	if candidate.TargetPhase != phase.ID {
		unknowns = append(unknowns, model.Unknown{
			Stage: "candidate-validation", Step: "bind-target-phase", Reason: "INCOMPLETE_BINDING",
			UnknownClass: "INCOMPLETE_BINDING", NextOperation: "supply-exact-target-binding",
			BlockedBy: []string{"phase.id", candidate.TargetPhase},
		})
		return unknowns, refutations
	}
	activity, ok := activityByName(phase.Activities, candidate.TargetActivity)
	if !ok {
		unknowns = append(unknowns, model.Unknown{
			Stage: "candidate-validation", Step: "bind-target-activity", Reason: "INCOMPLETE_BINDING",
			UnknownClass: "INCOMPLETE_BINDING", NextOperation: "supply-exact-target-binding",
			BlockedBy: []string{"activity.name", candidate.TargetActivity},
		})
		return unknowns, refutations
	}
	if !contains(meta.AllowedASTNodes, "rewrite_candidate") {
		refutations = append(refutations, model.Refutation{Stage: "candidate-validation", Step: "check-allowed-ast", Reason: "FORBIDDEN_AST_NODE", Counterexample: "rewrite_candidate"})
	}
	if candidate.Generation != meta.Generation {
		refutations = append(refutations, model.Refutation{Stage: "candidate-validation", Step: "check-generation-boundary", Reason: "GENERATION_BOUNDARY_MISMATCH", Counterexample: candidate.Generation})
	}
	for _, capability := range candidate.Capabilities {
		if !contains(meta.Capabilities, capability) {
			refutations = append(refutations, model.Refutation{Stage: "candidate-validation", Step: "check-capability", Reason: "UNDECLARED_CAPABILITY", Counterexample: capability})
		}
	}
	for _, effect := range candidate.Effects {
		if effect == "none" {
			continue
		}
		if contains(meta.ForbiddenEffects, effect) {
			refutations = append(refutations, model.Refutation{Stage: "candidate-validation", Step: "check-capability-effect", Reason: "FORBIDDEN_EFFECT", Counterexample: effect})
		}
	}
	if candidate.Rewrite.Kind == "replace-operation" {
		if candidate.Rewrite.Name != candidate.TargetActivity || candidate.Rewrite.InputType != activity.InputType || candidate.Rewrite.OutputType != activity.OutputType {
			refutations = append(refutations, model.Refutation{Stage: "candidate-validation", Step: "check-typed-rewrite", Reason: "TYPED_REWRITE_MISMATCH", Counterexample: candidate.Rewrite.Name})
		}
	}
	if candidate.Rewrite.Kind == "terminal-reason" && !contains(meta.Acceptance, candidate.Rewrite.Decision) {
		refutations = append(refutations, model.Refutation{Stage: "candidate-validation", Step: "check-terminal-policy", Reason: "UNACCEPTED_DECISION", Counterexample: candidate.Rewrite.Decision})
	}
	return unknowns, refutations
}

func compileStage(meta model.Phase, phase model.Phase, sourcePath, outputDir, stage string) (model.StageResult, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return model.StageResult{}, err
	}
	sourceDigest, _, err := gooo.DigestFile(sourcePath)
	if err != nil {
		return model.StageResult{}, err
	}
	namespace, declarations, sourceErr := gooo.ParseSource(sourcePath)
	unknowns := []model.Unknown{}
	refutations := []model.Refutation{}
	if sourceErr != nil {
		unknowns = append(unknowns, model.Unknown{Stage: "source", Step: "parse-gooo-input", Reason: "SOURCE_NOT_PARSEABLE", UnknownClass: sourceUnknownClass(sourceErr), NextOperation: "repair-gooo-input", BlockedBy: []string{"source.bytes"}})
	}
	seen := map[string]bool{}
	if sourceErr == nil {
		for _, declaration := range declarations {
			if !contains(meta.AllowedASTNodes, declaration.Kind) {
				refutations = append(refutations, model.Refutation{Stage: "lower", Step: "check-allowed-ast", Reason: "FORBIDDEN_AST_NODE", Counterexample: declaration.Kind})
			}
			if seen[declaration.StableID] {
				refutations = append(refutations, model.Refutation{Stage: "lower", Step: "check-stable-id-uniqueness", Reason: "DUPLICATE_STABLE_ID", Counterexample: declaration.StableID})
			}
			seen[declaration.StableID] = true
		}
	}
	decision := model.Reduce(unknowns, refutations)
	terminal := terminalFor(phase, decision, unknowns, refutations)
	ir := semanticIR{Schema: gooo.IRSchema, Namespace: namespace, Declarations: declarations, SourceDigest: sourceDigest, Terminal: terminal}
	irBytes, err := gooo.Marshal(ir)
	if err != nil {
		return model.StageResult{}, err
	}
	semanticDigest := gooo.DigestBytes(irBytes)
	generatedBytes, err := generateGo(phase, ir, semanticDigest)
	if err != nil {
		return model.StageResult{}, err
	}
	terminalBytes, err := gooo.Marshal(terminal)
	if err != nil {
		return model.StageResult{}, err
	}
	receipt := map[string]any{
		"schema": "gooo/self-rewrite-stage-receipt/v1", "stage": stage,
		"phase_digest": phase.SourceDigest, "source_digest": sourceDigest,
		"semantic_digest": semanticDigest, "generated_digest": gooo.DigestBytes(generatedBytes),
		"terminal_digest": gooo.DigestBytes(terminalBytes), "decision": decision,
	}
	receiptBytes, err := gooo.Marshal(receipt)
	if err != nil {
		return model.StageResult{}, err
	}
	paths := map[string][]byte{
		"semantic-ir.json": irBytes, "generated.go": generatedBytes,
		"terminal-record.json": terminalBytes, "receipt.json": receiptBytes,
	}
	artifacts := make([]model.Artifact, 0, len(paths))
	for name, data := range paths {
		path := filepath.Join(outputDir, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return model.StageResult{}, err
		}
		kind := strings.TrimSuffix(strings.ToUpper(strings.ReplaceAll(name, ".json", "")), ".GO")
		if name == "generated.go" {
			kind = "GENERATED_GO"
		}
		artifacts = append(artifacts, model.Artifact{Path: path, Digest: gooo.DigestBytes(data), Bytes: int64(len(data)), Kind: kind})
	}
	sort.Slice(artifacts, func(left, right int) bool { return artifacts[left].Path < artifacts[right].Path })
	if err := executeGenerated(filepath.Join(outputDir, "generated.go"), terminal); err != nil {
		return model.StageResult{}, err
	}
	return model.StageResult{Stage: stage, PhaseDigest: phase.SourceDigest, SourceDigest: sourceDigest, SemanticDigest: semanticDigest, GeneratedDigest: gooo.DigestBytes(generatedBytes), TerminalDigest: gooo.DigestBytes(terminalBytes), Decision: decision, Unknowns: unknowns, Refutations: refutations, Terminal: terminal, Artifacts: artifacts, Executed: true}, nil
}

func generateGo(phase model.Phase, ir semanticIR, semanticDigest string) ([]byte, error) {
	terminalJSON, err := json.Marshal(ir.Terminal)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("// Code generated by gooo-self-rewrite-sandbox from .gooo authority; DO NOT EDIT.\n")
	out.WriteString("package main\n\n")
	out.WriteString("import (\n\t\"encoding/json\"\n\t\"fmt\"\n)\n\n")
	fmt.Fprintf(&out, "const phaseDigest = %q\nconst semanticDigest = %q\n", phase.SourceDigest, semanticDigest)
	out.WriteString("var terminal = json.RawMessage(")
	out.WriteString(strconvQuote(string(terminalJSON)))
	out.WriteString(")\n\nfunc main() {\n\tvar value any\n\tif err := json.Unmarshal(terminal, &value); err != nil { panic(err) }\n\tdata, err := json.Marshal(value)\n\tif err != nil { panic(err) }\n\tfmt.Println(string(data))\n}\n")
	return []byte(out.String()), nil
}

func executeGenerated(path string, expected model.Terminal) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", filepath.Base(path))
	command.Dir = filepath.Dir(path)
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("execute generated compiler artifact: %w", err)
	}
	var actual model.Terminal
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &actual); err != nil {
		return fmt.Errorf("decode generated terminal: %w", err)
	}
	if actual.Decision != expected.Decision || actual.Reason != expected.Reason || actual.CounterexampleDigest != expected.CounterexampleDigest {
		return fmt.Errorf("generated artifact execution diverged from compiled terminal")
	}
	return nil
}

func terminalFor(phase model.Phase, decision string, unknowns []model.Unknown, refutations []model.Refutation) model.Terminal {
	terminal := model.Terminal{Schema: "gooo/self-rewrite-terminal/v1", Decision: decision, Stage: "emit", Step: "canonical-semantic-ir", Reason: phase.TerminalReasons[decision]}
	if terminal.Reason == "" {
		switch decision {
		case model.DecisionClosed:
			terminal.Reason = "SEMANTIC_IR_CANONICAL"
		case model.DecisionUnknown:
			terminal.Reason = "EVIDENCE_INCOMPLETE"
		case model.DecisionRefuted:
			terminal.Reason = "COUNTEREXAMPLE_OBSERVED"
		}
	}
	if len(refutations) > 0 {
		item := refutations[0]
		terminal.Stage, terminal.Step, terminal.Reason = item.Stage, item.Step, item.Reason
		terminal.Counterexample = item.Counterexample
		terminal.CounterexampleDigest = gooo.DigestBytes([]byte(item.Counterexample))
	}
	if len(refutations) == 0 && len(unknowns) > 0 {
		item := unknowns[0]
		terminal.Stage, terminal.Step, terminal.Reason = item.Stage, item.Step, item.Reason
		terminal.UnknownClass, terminal.NextOperation = item.UnknownClass, item.NextOperation
		terminal.BlockedBy = append([]string{}, item.BlockedBy...)
	}
	return terminal
}

func compareStages(kind string, baseline, candidate model.StageResult) (string, []model.Unknown, []model.Refutation) {
	if candidate.Decision != baseline.Decision {
		return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-terminal-decision", Reason: "DECISION_CHANGED", Counterexample: baseline.Decision + " -> " + candidate.Decision}}
	}
	switch kind {
	case "semantics-preserving":
		if baseline.SemanticDigest != candidate.SemanticDigest {
			return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-semantic-ir", Reason: "SEMANTIC_MISMATCH", Counterexample: baseline.SemanticDigest + " -> " + candidate.SemanticDigest}}
		}
	case "explanation-preserving":
		if baseline.SemanticDigest != candidate.SemanticDigest {
			return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-semantic-ir", Reason: "SEMANTIC_MISMATCH", Counterexample: baseline.SemanticDigest + " -> " + candidate.SemanticDigest}}
		}
		if baseline.TerminalDigest != candidate.TerminalDigest {
			return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-terminal-record", Reason: "EXPLANATION_MISMATCH", Counterexample: baseline.TerminalDigest + " -> " + candidate.TerminalDigest}}
		}
	case "changed-terminal-reason":
		if baseline.Terminal.Reason == candidate.Terminal.Reason {
			return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-terminal-reason", Reason: "EXPECTED_REASON_CHANGE_NOT_OBSERVED", Counterexample: candidate.Terminal.Reason}}
		}
		return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-terminal-reason", Reason: "TERMINAL_REASON_CHANGED", Counterexample: baseline.Terminal.Reason + " -> " + candidate.Terminal.Reason}}
	case "byte-identical-replay":
		if baseline.SemanticDigest != candidate.SemanticDigest || baseline.GeneratedDigest != candidate.GeneratedDigest || baseline.TerminalDigest != candidate.TerminalDigest {
			return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "compare-byte-replay", Reason: "BYTE_REPLAY_MISMATCH", Counterexample: baseline.GeneratedDigest + " -> " + candidate.GeneratedDigest}}
		}
	default:
		return model.DecisionRefuted, nil, []model.Refutation{{Stage: "compare", Step: "check-fixed-kind", Reason: "UNSUPPORTED_CASE_KIND", Counterexample: kind}}
	}
	return model.DecisionClosed, nil, nil
}

func makeImprovement(scenario, inputDigest, contractDigest, toolchain string, before, after model.StageResult, decision string) model.Improvement {
	beforePair := model.Pair{Scenario: scenario, SourceDigest: inputDigest, ContractDigest: contractDigest, Toolchain: toolchain, BeforeDigest: before.GeneratedDigest}
	afterPair := model.Pair{Scenario: scenario, SourceDigest: inputDigest, ContractDigest: contractDigest, Toolchain: toolchain, AfterDigest: after.GeneratedDigest}
	exact := before.SourceDigest == after.SourceDigest && before.SemanticDigest != "" && after.SemanticDigest != "" && beforePair.Scenario == afterPair.Scenario && beforePair.SourceDigest == afterPair.SourceDigest && beforePair.ContractDigest == afterPair.ContractDigest && beforePair.Toolchain == afterPair.Toolchain
	if exact && decision == model.DecisionClosed {
		return model.Improvement{Status: model.DecisionClosed, ExactPair: true, Before: beforePair, After: afterPair}
	}
	unknown := model.Unknown{Stage: "improvement", Step: "bind-exact-before-after", Reason: "NO_EXACT_BEFORE_AFTER_PAIR", UnknownClass: "NO_EXACT_PAIR", NextOperation: "capture-matched-before-after", BlockedBy: []string{"scenario", "source_digest", "contract_digest", "toolchain"}}
	return model.Improvement{Status: model.DecisionUnknown, ExactPair: exact, Before: beforePair, After: afterPair, Unknown: &unknown}
}

func blockedImprovement(scenario, inputDigest, contractDigest, toolchain string, unknowns []model.Unknown, refutations []model.Refutation) model.Improvement {
	unknown := model.Unknown{Stage: "improvement", Step: "bind-exact-before-after", Reason: "NO_EXACT_BEFORE_AFTER_PAIR", UnknownClass: "NO_EXACT_PAIR", NextOperation: "capture-matched-before-after", BlockedBy: []string{"candidate-artifact"}}
	if len(unknowns) > 0 {
		unknown = unknowns[0]
	}
	if len(refutations) > 0 {
		unknown = model.Unknown{Stage: "improvement", Step: "bind-exact-before-after", Reason: "NO_ACCEPTED_AFTER_ARTIFACT", UnknownClass: "NO_EXACT_PAIR", NextOperation: "capture-matched-before-after", BlockedBy: []string{"candidate-artifact"}}
	}
	return model.Improvement{Status: model.DecisionUnknown, ExactPair: false, Before: model.Pair{Scenario: scenario, SourceDigest: inputDigest, ContractDigest: contractDigest, Toolchain: toolchain}, After: model.Pair{Scenario: scenario, SourceDigest: inputDigest, ContractDigest: contractDigest, Toolchain: toolchain}, Unknown: &unknown}
}

func sourceUnknownClass(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "namespace is required"):
		return "SYNTAX_ERROR"
	case strings.Contains(message, "invalid namespace"):
		return "DUPLICATE_NAMESPACE"
	case strings.Contains(message, "unsupported declaration"):
		return "UNSUPPORTED_DECLARATION"
	case strings.Contains(message, "stable id is invalid"):
		return "INVALID_DECLARATION"
	default:
		return "SOURCE_PARSE_ERROR"
	}
}

func activityByName(activities []model.PhaseActivity, name string) (model.PhaseActivity, bool) {
	for _, activity := range activities {
		if activity.Name == name {
			return activity, true
		}
	}
	return model.PhaseActivity{}, false
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func ensureInside(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s is outside %s", path, root)
	}
	return nil
}

func ensureOutside(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return errors.New("path is inside immutable input root")
	}
	return nil
}

func snapshotsEqual(left, right []gooo.FileSnapshot) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func generatedTotals(cases []model.CaseResult) (int, int64) {
	count := 0
	var bytes int64
	for _, item := range cases {
		for _, artifact := range item.Baseline.Artifacts {
			count++
			bytes += artifact.Bytes
		}
		if item.Candidate != nil {
			for _, artifact := range item.Candidate.Artifacts {
				count++
				bytes += artifact.Bytes
			}
		}
	}
	return count, bytes
}

func writeReport(path string, report model.Report) error {
	data, err := gooo.Marshal(report)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func safeName(value string) string {
	value = strings.ToLower(value)
	var out strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out.WriteRune(r)
		} else {
			out.WriteByte('-')
		}
	}
	return strings.Trim(out.String(), "-")
}

func mustRelative(root, path string) string {
	value, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return value
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
