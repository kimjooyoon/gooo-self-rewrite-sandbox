package gooo

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-self-rewrite-sandbox/internal/model"
)

const (
	PhaseSchema     = "gooo/self-rewrite-phase/v1"
	CandidateSchema = "gooo/self-rewrite-candidate/v1"
	IRSchema        = "gooo/self-rewrite-semantic-ir/v1"
)

func DigestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestFile(path string) (string, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return DigestBytes(data), data, nil
}

func Marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func DigestJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return DigestBytes(data), nil
}

func LoadPhase(path string) (model.Phase, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return model.Phase{}, err
	}
	phase := model.Phase{
		Program:         "",
		AllowedASTNodes: []string{}, Preconditions: []string{}, Postconditions: []string{},
		Capabilities: []string{}, ForbiddenEffects: []string{}, Acceptance: []string{},
		Precedence: []string{}, TerminalReasons: map[string]string{}, Activities: []model.PhaseActivity{},
		SourcePath: path, SourceDigest: digest,
	}
	seenProgram, seenNamespace, seenPhase, seenBoundary := false, false, false, false
	if err := scanLines(data, func(line string, lineNumber int) error {
		switch {
		case strings.HasPrefix(line, "program "):
			if seenProgram {
				return fmt.Errorf("line %d: duplicate program declaration", lineNumber)
			}
			seenProgram = true
			phase.Program = strings.TrimSpace(strings.TrimPrefix(line, "program "))
		case strings.HasPrefix(line, "namespace "):
			if seenNamespace {
				return fmt.Errorf("line %d: duplicate namespace declaration", lineNumber)
			}
			seenNamespace = true
			phase.Namespace = strings.TrimSpace(strings.TrimPrefix(line, "namespace "))
		case strings.HasPrefix(line, "phase "):
			if seenPhase {
				return fmt.Errorf("line %d: duplicate phase declaration", lineNumber)
			}
			seenPhase = true
			phase.ID = strings.TrimSpace(strings.TrimPrefix(line, "phase "))
		case strings.HasPrefix(line, "allowed_ast "):
			phase.AllowedASTNodes = append(phase.AllowedASTNodes, strings.TrimSpace(strings.TrimPrefix(line, "allowed_ast ")))
		case strings.HasPrefix(line, "precondition "):
			phase.Preconditions = append(phase.Preconditions, strings.TrimSpace(strings.TrimPrefix(line, "precondition ")))
		case strings.HasPrefix(line, "postcondition "):
			phase.Postconditions = append(phase.Postconditions, strings.TrimSpace(strings.TrimPrefix(line, "postcondition ")))
		case strings.HasPrefix(line, "capability "):
			phase.Capabilities = append(phase.Capabilities, strings.TrimSpace(strings.TrimPrefix(line, "capability ")))
		case strings.HasPrefix(line, "forbidden_effect "):
			phase.ForbiddenEffects = append(phase.ForbiddenEffects, strings.TrimSpace(strings.TrimPrefix(line, "forbidden_effect ")))
		case strings.HasPrefix(line, "boundary "):
			if seenBoundary {
				return fmt.Errorf("line %d: duplicate boundary declaration", lineNumber)
			}
			seenBoundary = true
			phase.Generation = strings.TrimSpace(strings.TrimPrefix(line, "boundary "))
		case strings.HasPrefix(line, "accept "):
			phase.Acceptance = append(phase.Acceptance, strings.TrimSpace(strings.TrimPrefix(line, "accept ")))
		case strings.HasPrefix(line, "precedence "):
			phase.Precedence = strings.Fields(strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(line, "precedence ")), ">", " "))
		case strings.HasPrefix(line, "terminal_reason "):
			decision, reason, ok := parseDecisionReason(strings.TrimSpace(strings.TrimPrefix(line, "terminal_reason ")))
			if !ok {
				return fmt.Errorf("line %d: invalid terminal_reason", lineNumber)
			}
			if _, exists := phase.TerminalReasons[decision]; exists {
				return fmt.Errorf("line %d: duplicate terminal_reason for %s", lineNumber, decision)
			}
			phase.TerminalReasons[decision] = reason
		case strings.HasPrefix(line, "activity "):
			activity, ok := parsePhaseActivity(line)
			if !ok {
				return fmt.Errorf("line %d: invalid activity", lineNumber)
			}
			phase.Activities = append(phase.Activities, activity)
		default:
			return fmt.Errorf("line %d: unsupported phase declaration", lineNumber)
		}
		return nil
	}); err != nil {
		return model.Phase{}, err
	}
	if phase.Program == "" || phase.Namespace == "" || phase.ID == "" {
		return model.Phase{}, fmt.Errorf("phase requires program, namespace, and phase")
	}
	if len(phase.Precedence) == 0 {
		phase.Precedence = []string{model.DecisionRefuted, model.DecisionUnknown, model.DecisionClosed}
	}
	if len(phase.Acceptance) == 0 {
		phase.Acceptance = []string{model.DecisionClosed, model.DecisionUnknown, model.DecisionRefuted}
	}
	if len(phase.Activities) == 0 {
		return model.Phase{}, fmt.Errorf("phase must declare an activity")
	}
	return phase, nil
}

func LoadCandidate(path string) (model.Candidate, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return model.Candidate{}, err
	}
	candidate := model.Candidate{
		Preconditions: []string{}, Postconditions: []string{}, Capabilities: []string{},
		Effects: []string{}, Acceptance: []string{}, RefutationPolicy: []string{},
		SourcePath: path, SourceDigest: digest,
	}
	seenProgram, seenCandidate, seenTargetPhase, seenTargetActivity, seenBoundary, seenRewrite := false, false, false, false, false, false
	if err := scanLines(data, func(line string, lineNumber int) error {
		switch {
		case strings.HasPrefix(line, "program "):
			if seenProgram {
				return fmt.Errorf("line %d: duplicate program declaration", lineNumber)
			}
			seenProgram = true
			candidate.Program = strings.TrimSpace(strings.TrimPrefix(line, "program "))
		case strings.HasPrefix(line, "candidate "):
			if seenCandidate {
				return fmt.Errorf("line %d: duplicate candidate declaration", lineNumber)
			}
			seenCandidate = true
			candidate.ID = strings.TrimSpace(strings.TrimPrefix(line, "candidate "))
		case strings.HasPrefix(line, "target_phase "):
			if seenTargetPhase {
				return fmt.Errorf("line %d: duplicate target_phase declaration", lineNumber)
			}
			seenTargetPhase = true
			candidate.TargetPhase = strings.TrimSpace(strings.TrimPrefix(line, "target_phase "))
		case strings.HasPrefix(line, "target_activity "):
			if seenTargetActivity {
				return fmt.Errorf("line %d: duplicate target_activity declaration", lineNumber)
			}
			seenTargetActivity = true
			candidate.TargetActivity = strings.TrimSpace(strings.TrimPrefix(line, "target_activity "))
		case strings.HasPrefix(line, "rewrite "):
			if seenRewrite {
				return fmt.Errorf("line %d: duplicate rewrite declaration", lineNumber)
			}
			seenRewrite = true
			rewrite, ok := parseRewrite(strings.TrimSpace(strings.TrimPrefix(line, "rewrite ")))
			if !ok {
				return fmt.Errorf("line %d: invalid rewrite", lineNumber)
			}
			candidate.Rewrite = rewrite
		case strings.HasPrefix(line, "precondition "):
			candidate.Preconditions = append(candidate.Preconditions, strings.TrimSpace(strings.TrimPrefix(line, "precondition ")))
		case strings.HasPrefix(line, "postcondition "):
			candidate.Postconditions = append(candidate.Postconditions, strings.TrimSpace(strings.TrimPrefix(line, "postcondition ")))
		case strings.HasPrefix(line, "capability "):
			candidate.Capabilities = append(candidate.Capabilities, strings.TrimSpace(strings.TrimPrefix(line, "capability ")))
		case strings.HasPrefix(line, "effect "):
			candidate.Effects = append(candidate.Effects, strings.TrimSpace(strings.TrimPrefix(line, "effect ")))
		case strings.HasPrefix(line, "boundary "):
			if seenBoundary {
				return fmt.Errorf("line %d: duplicate boundary declaration", lineNumber)
			}
			seenBoundary = true
			candidate.Generation = strings.TrimSpace(strings.TrimPrefix(line, "boundary "))
		case strings.HasPrefix(line, "accept "):
			candidate.Acceptance = append(candidate.Acceptance, strings.TrimSpace(strings.TrimPrefix(line, "accept ")))
		case strings.HasPrefix(line, "refute "):
			candidate.RefutationPolicy = append(candidate.RefutationPolicy, strings.TrimSpace(strings.TrimPrefix(line, "refute ")))
		default:
			return fmt.Errorf("line %d: unsupported candidate declaration", lineNumber)
		}
		return nil
	}); err != nil {
		return model.Candidate{}, err
	}
	if candidate.Program == "" || candidate.ID == "" || candidate.TargetPhase == "" || candidate.TargetActivity == "" {
		return model.Candidate{}, fmt.Errorf("candidate requires program, candidate, target_phase, and target_activity")
	}
	if candidate.Generation == "" {
		return model.Candidate{}, fmt.Errorf("candidate requires a generation boundary")
	}
	if candidate.Rewrite.Kind == "" {
		return model.Candidate{}, fmt.Errorf("candidate requires a typed rewrite")
	}
	return candidate, nil
}

func parseRewrite(value string) (model.TypedRewrite, bool) {
	if value == "noop" {
		return model.TypedRewrite{Kind: "noop"}, true
	}
	if strings.HasPrefix(value, "replace-operation ") {
		rest := strings.TrimSpace(strings.TrimPrefix(value, "replace-operation "))
		quote := strings.IndexByte(rest, '"')
		if quote < 0 {
			return model.TypedRewrite{}, false
		}
		fields := strings.Fields(strings.TrimSpace(rest[:quote]))
		if len(fields) != 3 {
			return model.TypedRewrite{}, false
		}
		program, err := strconv.Unquote(strings.TrimSpace(rest[quote:]))
		if err != nil || program == "" {
			return model.TypedRewrite{}, false
		}
		return model.TypedRewrite{Kind: "replace-operation", Name: fields[0], InputType: fields[1], OutputType: fields[2], Program: program}, true
	}
	if strings.HasPrefix(value, "terminal-reason ") {
		rest := strings.TrimSpace(strings.TrimPrefix(value, "terminal-reason "))
		quote := strings.IndexByte(rest, '"')
		if quote < 0 {
			return model.TypedRewrite{}, false
		}
		decision := strings.TrimSpace(rest[:quote])
		reason, err := strconv.Unquote(strings.TrimSpace(rest[quote:]))
		if err != nil || decision == "" || reason == "" {
			return model.TypedRewrite{}, false
		}
		return model.TypedRewrite{Kind: "terminal-reason", Decision: decision, Reason: reason}, true
	}
	return model.TypedRewrite{}, false
}

func parseDecisionReason(value string) (string, string, bool) {
	quote := strings.IndexByte(value, '"')
	if quote < 0 {
		return "", "", false
	}
	decision := strings.TrimSpace(value[:quote])
	reason, err := strconv.Unquote(strings.TrimSpace(value[quote:]))
	if err != nil || decision == "" || reason == "" {
		return "", "", false
	}
	return decision, reason, true
}

func parsePhaseActivity(line string) (model.PhaseActivity, bool) {
	body := strings.TrimSpace(strings.TrimPrefix(line, "activity "))
	leftParen := strings.IndexByte(body, '(')
	rightParen := strings.IndexByte(body, ')')
	if leftParen < 1 || rightParen <= leftParen {
		return model.PhaseActivity{}, false
	}
	arrow := strings.TrimSpace(body[rightParen+1:])
	if !strings.HasPrefix(arrow, "->") {
		return model.PhaseActivity{}, false
	}
	arrow = strings.TrimSpace(strings.TrimPrefix(arrow, "->"))
	space := strings.IndexByte(arrow, ' ')
	if space <= 0 {
		return model.PhaseActivity{}, false
	}
	output := arrow[:space]
	computed := strings.TrimSpace(arrow[space:])
	if !strings.HasPrefix(computed, "computes ") {
		return model.PhaseActivity{}, false
	}
	program, err := strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(computed, "computes ")))
	if err != nil || program == "" {
		return model.PhaseActivity{}, false
	}
	return model.PhaseActivity{
		Name: body[:leftParen], InputType: strings.TrimSpace(body[leftParen+1 : rightParen]),
		OutputType: output, Program: program,
	}, true
}

func ParseSource(path string) (string, []model.Declaration, error) {
	_, data, err := DigestFile(path)
	if err != nil {
		return "", nil, err
	}
	var namespace string
	declarations := []model.Declaration{}
	if err := scanLines(data, func(line string, lineNumber int) error {
		switch {
		case line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "package "):
			return nil
		case strings.HasPrefix(line, "namespace "):
			fields := strings.Fields(line)
			if len(fields) != 2 || namespace != "" {
				return fmt.Errorf("line %d: invalid namespace", lineNumber)
			}
			namespace = fields[1]
		case strings.HasPrefix(line, "entity "):
			declaration, parseErr := parseEntity(line)
			if parseErr != nil {
				return fmt.Errorf("line %d: %w", lineNumber, parseErr)
			}
			declarations = append(declarations, declaration)
		case strings.HasPrefix(line, "activity "):
			declaration, parseErr := parseSourceActivity(line)
			if parseErr != nil {
				return fmt.Errorf("line %d: %w", lineNumber, parseErr)
			}
			declarations = append(declarations, declaration)
		default:
			return fmt.Errorf("line %d: unsupported declaration", lineNumber)
		}
		return nil
	}); err != nil {
		return "", nil, err
	}
	if namespace == "" {
		return "", nil, fmt.Errorf("namespace is required")
	}
	sort.SliceStable(declarations, func(left, right int) bool {
		if declarations[left].StableID != declarations[right].StableID {
			return declarations[left].StableID < declarations[right].StableID
		}
		if declarations[left].Kind != declarations[right].Kind {
			return declarations[left].Kind < declarations[right].Kind
		}
		return declarations[left].Name < declarations[right].Name
	})
	return namespace, declarations, nil
}

func parseEntity(line string) (model.Declaration, error) {
	fields := strings.Fields(line)
	if len(fields) != 4 || fields[0] != "entity" || fields[2] != "id" {
		return model.Declaration{}, fmt.Errorf("entity must be 'entity Name id \"stable-id\"'")
	}
	id, err := strconv.Unquote(fields[3])
	if err != nil || id == "" {
		return model.Declaration{}, fmt.Errorf("entity stable id is invalid")
	}
	return model.Declaration{Kind: "entity", StableID: id, Name: fields[1]}, nil
}

func parseSourceActivity(line string) (model.Declaration, error) {
	body := strings.TrimSpace(strings.TrimPrefix(line, "activity "))
	left := strings.IndexByte(body, '(')
	right := strings.IndexByte(body, ')')
	if left < 1 || right <= left || !strings.HasPrefix(strings.TrimSpace(body[right+1:]), "->") {
		return model.Declaration{}, fmt.Errorf("activity signature is invalid")
	}
	name := strings.TrimSpace(body[:left])
	result := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(body[right+1:]), "->"))
	if name == "" || result == "" || strings.Contains(result, " ") {
		return model.Declaration{}, fmt.Errorf("activity signature is invalid")
	}
	params := []string{}
	parameterText := strings.TrimSpace(body[left+1 : right])
	if parameterText != "" {
		for _, value := range strings.Split(parameterText, ",") {
			value = strings.TrimSpace(value)
			if value == "" {
				return model.Declaration{}, fmt.Errorf("activity signature is invalid")
			}
			params = append(params, value)
		}
	}
	return model.Declaration{Kind: "activity", StableID: "gooo://activity/" + name, Name: name, Parameters: params, Result: result}, nil
}

func scanLines(data []byte, callback func(string, int) error) error {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		if err := callback(line, lineNumber); err != nil {
			return err
		}
	}
	return scanner.Err()
}

type FileSnapshot struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Bytes  int64  `json:"bytes"`
}

func SnapshotDir(root string) (string, []FileSnapshot, error) {
	files := []FileSnapshot{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not an immutable input: %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, FileSnapshot{Path: filepath.ToSlash(relative), Digest: DigestBytes(data), Bytes: info.Size()})
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	sort.Slice(files, func(left, right int) bool { return files[left].Path < files[right].Path })
	digest, err := DigestJSON(files)
	return digest, files, err
}

func CopyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not supported in immutable input: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func ApplyCandidate(phasePath string, candidate model.Candidate) ([]byte, error) {
	data, err := os.ReadFile(phasePath)
	if err != nil {
		return nil, err
	}
	if candidate.Rewrite.Kind == "noop" {
		return data, nil
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch candidate.Rewrite.Kind {
		case "replace-operation":
			prefix := "activity " + candidate.Rewrite.Name + "("
			if strings.HasPrefix(trimmed, prefix) {
				lines[index] = "activity " + candidate.Rewrite.Name + "(" + candidate.Rewrite.InputType + ") -> " + candidate.Rewrite.OutputType + " computes " + strconv.Quote(candidate.Rewrite.Program)
				changed = true
			}
		case "terminal-reason":
			if strings.HasPrefix(trimmed, "terminal_reason "+candidate.Rewrite.Decision+" ") {
				lines[index] = "terminal_reason " + candidate.Rewrite.Decision + " " + strconv.Quote(candidate.Rewrite.Reason)
				changed = true
			}
		}
	}
	if candidate.Rewrite.Kind == "terminal-reason" && !changed {
		lines = append(lines, "terminal_reason "+candidate.Rewrite.Decision+" "+strconv.Quote(candidate.Rewrite.Reason))
		changed = true
	}
	if !changed {
		return nil, fmt.Errorf("typed rewrite did not bind to phase: %s", candidate.Rewrite.Kind)
	}
	return []byte(strings.Join(lines, "\n")), nil
}
