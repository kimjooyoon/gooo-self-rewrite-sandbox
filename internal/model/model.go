package model

const (
	DecisionClosed  = "CLOSED"
	DecisionUnknown = "UNKNOWN"
	DecisionRefuted = "REFUTED"
)

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Refutation struct {
	Stage          string `json:"stage"`
	Step           string `json:"step"`
	Reason         string `json:"reason"`
	Counterexample string `json:"counterexample"`
}

type Terminal struct {
	Schema               string   `json:"schema"`
	Decision             string   `json:"decision"`
	Stage                string   `json:"stage"`
	Step                 string   `json:"step"`
	Reason               string   `json:"reason"`
	UnknownClass         string   `json:"unknown_class"`
	NextOperation        string   `json:"next_operation"`
	BlockedBy            []string `json:"blocked_by"`
	Counterexample       string   `json:"counterexample"`
	CounterexampleDigest string   `json:"counterexample_digest"`
}

type Declaration struct {
	Kind       string   `json:"kind"`
	StableID   string   `json:"stable_id"`
	Name       string   `json:"name"`
	Parameters []string `json:"parameters,omitempty"`
	Result     string   `json:"result,omitempty"`
}

type PhaseActivity struct {
	Name       string `json:"name"`
	InputType  string `json:"input_type"`
	OutputType string `json:"output_type"`
	Program    string `json:"program"`
}

type Phase struct {
	Program          string            `json:"program"`
	Namespace        string            `json:"namespace"`
	ID               string            `json:"id"`
	AllowedASTNodes  []string          `json:"allowed_ast_nodes"`
	Preconditions    []string          `json:"preconditions"`
	Postconditions   []string          `json:"postconditions"`
	Capabilities     []string          `json:"capabilities"`
	ForbiddenEffects []string          `json:"forbidden_effects"`
	Generation       string            `json:"generation_boundary"`
	Acceptance       []string          `json:"acceptance"`
	Precedence       []string          `json:"precedence"`
	TerminalReasons  map[string]string `json:"terminal_reasons"`
	Activities       []PhaseActivity   `json:"activities"`
	SourcePath       string            `json:"source_path"`
	SourceDigest     string            `json:"source_digest"`
}

type TypedRewrite struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	InputType  string `json:"input_type,omitempty"`
	OutputType string `json:"output_type,omitempty"`
	Program    string `json:"program,omitempty"`
	Decision   string `json:"decision,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type Candidate struct {
	Program          string       `json:"program"`
	ID               string       `json:"id"`
	TargetPhase      string       `json:"target_phase"`
	TargetActivity   string       `json:"target_activity"`
	Rewrite          TypedRewrite `json:"rewrite"`
	Preconditions    []string     `json:"preconditions"`
	Postconditions   []string     `json:"postconditions"`
	Capabilities     []string     `json:"capabilities"`
	Effects          []string     `json:"effects"`
	Generation       string       `json:"generation_boundary"`
	Acceptance       []string     `json:"acceptance"`
	RefutationPolicy []string     `json:"refutation_policy"`
	SourcePath       string       `json:"source_path"`
	SourceDigest     string       `json:"source_digest"`
}

type Artifact struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Bytes  int64  `json:"bytes"`
	Kind   string `json:"kind"`
}

type StageResult struct {
	Stage           string       `json:"stage"`
	PhaseDigest     string       `json:"phase_digest"`
	SourceDigest    string       `json:"source_digest"`
	SemanticDigest  string       `json:"semantic_digest"`
	GeneratedDigest string       `json:"generated_digest"`
	TerminalDigest  string       `json:"terminal_digest"`
	Decision        string       `json:"decision"`
	Unknowns        []Unknown    `json:"unknowns"`
	Refutations     []Refutation `json:"refutations"`
	Terminal        Terminal     `json:"terminal"`
	Artifacts       []Artifact   `json:"artifacts"`
	Executed        bool         `json:"executed"`
}

type Pair struct {
	Scenario       string `json:"scenario"`
	SourceDigest   string `json:"source_digest"`
	ContractDigest string `json:"contract_digest"`
	Toolchain      string `json:"toolchain"`
	BeforeDigest   string `json:"before_digest"`
	AfterDigest    string `json:"after_digest"`
}

type Improvement struct {
	Status    string   `json:"status"`
	ExactPair bool     `json:"exact_pair"`
	Before    Pair     `json:"before"`
	After     Pair     `json:"after"`
	Unknown   *Unknown `json:"unknown,omitempty"`
}

type CaseResult struct {
	ID               string       `json:"id"`
	Kind             string       `json:"kind"`
	Expected         string       `json:"expected"`
	Decision         string       `json:"decision"`
	CandidatePath    string       `json:"candidate_path"`
	CandidateDigest  string       `json:"candidate_digest"`
	CandidateApplied bool         `json:"candidate_applied"`
	Baseline         StageResult  `json:"baseline"`
	Candidate        *StageResult `json:"candidate,omitempty"`
	Unknowns         []Unknown    `json:"unknowns"`
	Refutations      []Refutation `json:"refutations"`
	Improvement      Improvement  `json:"improvement"`
	Match            bool         `json:"match"`
}

type CorpusCase struct {
	ID        string `json:"id"`
	Candidate string `json:"candidate"`
	Expected  string `json:"expected"`
	Kind      string `json:"kind"`
}

type Corpus struct {
	Schema        string       `json:"schema"`
	UnknownFields []string     `json:"unknown_fields"`
	Cases         []CorpusCase `json:"cases"`
}

type Report struct {
	Schema                   string       `json:"schema"`
	Decision                 string       `json:"decision"`
	InputRoot                string       `json:"input_root"`
	InputDigest              string       `json:"input_digest"`
	ContractDigest           string       `json:"contract_digest"`
	Toolchain                string       `json:"toolchain"`
	RepositoryWrites         int          `json:"repository_writes"`
	AutomaticCommitPushMerge int          `json:"automatic_commit_push_merge"`
	Cases                    []CaseResult `json:"cases"`
	UnknownFields            []string     `json:"unknown_fields"`
	Precedence               []string     `json:"precedence"`
	GeneratedArtifactsCount  int          `json:"generated_artifacts_count"`
	GeneratedArtifactsBytes  int64        `json:"generated_artifacts_bytes"`
}

func Reduce(unknowns []Unknown, refutations []Refutation) string {
	if len(refutations) > 0 {
		return DecisionRefuted
	}
	if len(unknowns) > 0 {
		return DecisionUnknown
	}
	return DecisionClosed
}
