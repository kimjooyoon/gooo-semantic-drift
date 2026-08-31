package drift

import "encoding/json"

const (
	ProtocolSchema = "gooo/semantic-drift/protocol/v1"
	EnvelopeSchema = "gooo/semantic-drift/envelope/v1"
	IRSchema       = "gooo/semantic-drift/ir/v1"
	ContractSchema = "gooo/semantic-drift/denominator/v1"
	StateClosed    = "CLOSED"
	StateUnknown   = "UNKNOWN"
	StateRefuted   = "REFUTED"
)

var Precedence = []string{StateRefuted, StateUnknown, StateClosed}

type Comparison struct {
	Schema                string                   `json:"schema"`
	CaseID                string                   `json:"case_id"`
	Base                  EnvelopeBinding         `json:"base"`
	Candidate             EnvelopeBinding         `json:"candidate"`
	DenominatorMigrations []DenominatorMigration `json:"denominator_migrations"`
	SchemaMigration       *SchemaMigration       `json:"schema_migration"`
	LoweringEvidence      []LoweringEvidence      `json:"lowering_evidence"`
	ArtifactEvidence      []ArtifactEvidence      `json:"artifact_evidence"`
	Replay                ReplayInput             `json:"replay"`
	OptionalInputs        []OptionalRelease       `json:"optional_inputs"`
	Authority             AuthorityInput          `json:"authority"`
}

type EnvelopeBinding struct {
	Envelope       Envelope `json:"envelope"`
	ObservedDigest string   `json:"observed_digest"`
}

type Envelope struct {
	Schema             string              `json:"schema"`
	EnvelopeID         string              `json:"envelope_id"`
	LanguageRelease    LanguageRelease     `json:"language_release"`
	CompilerRelease    CompilerRelease     `json:"compiler_release"`
	SyntaxDenominator  []SyntaxCell        `json:"syntax_denominator"`
	SchemaFingerprint  SchemaFingerprint   `json:"schema_fingerprint"`
	Lowering           []LoweringIdentity  `json:"lowering"`
	GeneratedArtifacts []GeneratedArtifact `json:"generated_artifacts"`
	ReleaseDigest      string              `json:"release_digest"`
}

type LanguageRelease struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type CompilerRelease struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type SyntaxCell struct {
	Ordinal       int    `json:"ordinal"`
	ID            string `json:"id"`
	CanonicalName string `json:"canonical_name"`
	SemanticsDigest string `json:"semantics_digest"`
}

type SchemaFingerprint struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type LoweringIdentity struct {
	SourceID     string `json:"source_id"`
	TargetID     string `json:"target_id"`
	IdentityMode string `json:"identity_mode"`
	MappingDigest string `json:"mapping_digest"`
}

type GeneratedArtifact struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Digest string `json:"digest"`
}

type DenominatorMigration struct {
	Kind          string   `json:"kind"`
	FromIDs       []string `json:"from_ids"`
	ToIDs         []string `json:"to_ids"`
	Reason        string   `json:"reason"`
	EvidenceDigest string  `json:"evidence_digest"`
}

type SchemaMigration struct {
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	FromDigest  string `json:"from_digest"`
	ToDigest    string `json:"to_digest"`
	Reason      string `json:"reason"`
	EvidenceDigest string `json:"evidence_digest"`
}

type LoweringEvidence struct {
	SourceID       string `json:"source_id"`
	FromTargetID   string `json:"from_target_id"`
	ToTargetID     string `json:"to_target_id"`
	EvidenceDigest string `json:"evidence_digest"`
	Reason         string `json:"reason"`
}

type ArtifactEvidence struct {
	Name           string `json:"name"`
	FromDigest     string `json:"from_digest"`
	ToDigest       string `json:"to_digest"`
	EvidenceDigest string `json:"evidence_digest"`
	Reason         string `json:"reason"`
}

type ReplayInput struct {
	CorpusID         string             `json:"corpus_id"`
	Bound            bool               `json:"bound"`
	ExpectedInputIDs []string           `json:"expected_input_ids"`
	Observations     []ReplayObservation `json:"observations"`
}

type ReplayObservation struct {
	InputID       string              `json:"input_id"`
	InputDigest   string              `json:"input_digest"`
	ReplayStatus  string              `json:"replay_status"`
	Base          ValueObservation    `json:"base"`
	Candidate     ValueObservation    `json:"candidate"`
}

type ValueObservation struct {
	EnvelopeDigest string `json:"envelope_digest"`
	ValueDigest    string `json:"value_digest"`
}

type OptionalRelease struct {
	Name  string `json:"name"`
	Tag   string `json:"tag"`
	Digest string `json:"digest"`
}

type AuthorityInput struct {
	RepositoryWrites          int `json:"repository_writes"`
	LocalTestExecutions       int `json:"local_test_executions"`
	CrossProjectRequiredGates int `json:"cross_project_required_gates"`
}

type Contract struct {
	Schema        string         `json:"schema"`
	DenominatorID string         `json:"denominator_id"`
	Total         int            `json:"total"`
	Cells         []ContractCell `json:"cells"`
}

type ContractCell struct {
	Ordinal           int    `json:"ordinal"`
	ID                string `json:"id"`
	Activity          string `json:"activity"`
	Name              string `json:"name"`
	Dimension         string `json:"dimension"`
	MetricDenominator int    `json:"metric_denominator"`
	Source            string `json:"source"`
	IR                string `json:"ir"`
	GeneratedArtifact string `json:"generated_artifact"`
	Evaluator         string `json:"evaluator"`
}

type ActivityIR struct {
	ID         string `json:"id"`
	Activity   string `json:"activity"`
	Name       string `json:"name"`
	Dimension  string `json:"dimension"`
	Step       string `json:"step"`
	Artifact   string `json:"artifact"`
	Evaluator  string `json:"evaluator"`
	SourceLine int    `json:"source_line"`
}

type SemanticIR struct {
	Schema       string       `json:"schema"`
	SourcePath   string       `json:"source_path"`
	SourceDigest string       `json:"source_digest"`
	ContractPath string       `json:"contract_path"`
	ContractDigest string     `json:"contract_digest"`
	Activities   []ActivityIR `json:"activities"`
}

type Meta struct {
	Root             string
	SourcePath       string
	SourceDigest     string
	SemanticIRPath   string
	SemanticIRDigest string
	GeneratedGoPath  string
	GeneratedGoDigest string
	EvaluatorPath    string
	EvaluatorDigest  string
	ContractPath     string
	ContractDigest   string
	Contract         Contract
	AuthorityChain   AuthorityChain
}

type ArtifactRef struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type AuthorityChain struct {
	Source      ArtifactRef `json:"source"`
	SemanticIR  ArtifactRef `json:"semantic_ir"`
	GeneratedGo ArtifactRef `json:"generated_go"`
	Evaluator   ArtifactRef `json:"evaluator"`
}

type EnvelopeSummary struct {
	EnvelopeID       string `json:"envelope_id"`
	LanguageVersion  string `json:"language_version"`
	CompilerVersion  string `json:"compiler_version"`
	ReleaseDigest    string `json:"release_digest"`
	ObservedDigest   string `json:"observed_digest"`
	DigestValid      bool   `json:"digest_valid"`
	ObservationBound bool   `json:"observation_bound"`
}

type DenominatorEvent struct {
	Kind          string   `json:"kind"`
	FromIDs       []string `json:"from_ids"`
	ToIDs         []string `json:"to_ids"`
	Reason        string   `json:"reason"`
	EvidenceDigest string  `json:"evidence_digest"`
}

type DimensionResult struct {
	Dimension string   `json:"dimension"`
	State     string   `json:"state"`
	Changed   bool     `json:"changed"`
	Reason    string   `json:"reason"`
	Events    []string `json:"events"`
}

type DenominatorResult struct {
	Dimension       string             `json:"dimension"`
	State           string             `json:"state"`
	Changed         bool               `json:"changed"`
	BaseCellCount   int                `json:"base_cell_count"`
	CandidateCellCount int              `json:"candidate_cell_count"`
	Added           int                `json:"added"`
	Retired         int                `json:"retired"`
	Split           int                `json:"split"`
	Redefined       int                `json:"redefined"`
	Events          []DenominatorEvent `json:"events"`
	Reason          string             `json:"reason"`
}

type SchemaResult struct {
	Dimension       string `json:"dimension"`
	State           string `json:"state"`
	Changed         bool   `json:"changed"`
	BaseID          string `json:"base_id"`
	CandidateID     string `json:"candidate_id"`
	BaseVersion     string `json:"base_version"`
	CandidateVersion string `json:"candidate_version"`
	BaseDigest      string `json:"base_digest"`
	CandidateDigest string `json:"candidate_digest"`
	Reason          string `json:"reason"`
}

type LoweringResult struct {
	Dimension       string   `json:"dimension"`
	State           string   `json:"state"`
	Changed         bool     `json:"changed"`
	ChangedSourceIDs []string `json:"changed_source_ids"`
	ExplicitChanges int      `json:"explicit_changes"`
	InferredChanges int      `json:"inferred_changes"`
	Reason          string   `json:"reason"`
}

type ArtifactResult struct {
	Dimension       string   `json:"dimension"`
	State           string   `json:"state"`
	Changed         bool     `json:"changed"`
	ChangedNames    []string `json:"changed_names"`
	ExplicitChanges int      `json:"explicit_changes"`
	Reason          string   `json:"reason"`
}

type ReplayCounts struct {
	Total         int `json:"total"`
	Equivalent    int `json:"equivalent"`
	Contradiction int `json:"contradiction"`
	Unknown       int `json:"unknown"`
}

type ReplayObservationResult struct {
	InputID       string `json:"input_id"`
	InputDigest   string `json:"input_digest"`
	BaseValueDigest string `json:"base_value_digest"`
	CandidateValueDigest string `json:"candidate_value_digest"`
	Relation      string `json:"relation"`
}

type ReplayReceipt struct {
	Schema           string                   `json:"schema"`
	CorpusID         string                   `json:"corpus_id"`
	Bound            bool                     `json:"bound"`
	ExpectedInputIDs []string                 `json:"expected_input_ids"`
	Observations     []ReplayObservationResult `json:"observations"`
	Counts           ReplayCounts             `json:"counts"`
	State            string                   `json:"state"`
	Reason           string                   `json:"reason"`
}

type EvidenceEvent struct {
	Dimension       string   `json:"dimension"`
	Code            string   `json:"code"`
	State           string   `json:"state"`
	EvidenceDigests []string `json:"evidence_digests"`
	BoundInputs     []string `json:"bound_inputs"`
	Reason          string   `json:"reason"`
}

type CausalEvidence struct {
	Schema       string          `json:"schema"`
	CaseID       string          `json:"case_id"`
	InputDigest  string          `json:"input_digest"`
	Events       []EvidenceEvent `json:"events"`
}

type AuthorityReport struct {
	RepositoryWrites          int  `json:"repository_writes"`
	LocalTestExecutions       int  `json:"local_test_executions"`
	CrossProjectRequiredGates int  `json:"cross_project_required_gates"`
	ReadOnly                  bool `json:"read_only"`
}

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Inventory struct {
	RootREADMEExcluded bool     `json:"root_readme_excluded"`
	Violations         []string `json:"violations"`
}

type DriftManifest struct {
	Schema              string             `json:"schema"`
	CaseID              string             `json:"case_id"`
	InputDigest         string             `json:"input_digest"`
	Precedence          []string           `json:"precedence"`
	DenominatorTotal    int                `json:"denominator_total"`
	Base                EnvelopeSummary    `json:"base"`
	Candidate           EnvelopeSummary    `json:"candidate"`
	Denominator         DenominatorResult  `json:"denominator"`
	SchemaMigration     SchemaResult       `json:"schema_migration"`
	Lowering            LoweringResult     `json:"lowering"`
	GeneratedArtifacts  ArtifactResult     `json:"generated_artifacts"`
	ObservableBehavior  DimensionResult    `json:"observable_behavior"`
	Authority           AuthorityReport    `json:"authority"`
	OptionalInputs      []OptionalRelease   `json:"optional_inputs"`
	AuthorityChain      AuthorityChain     `json:"authority_chain"`
	Inventory           Inventory          `json:"inventory"`
}

type DecisionReceipt struct {
	Schema         string     `json:"schema"`
	CaseID         string     `json:"case_id"`
	InputDigest    string     `json:"input_digest"`
	State          string     `json:"state"`
	Reason         string     `json:"reason"`
	Precedence     []string   `json:"precedence"`
	Unknowns       []Unknown  `json:"unknowns"`
	Contradictions []string   `json:"contradictions"`
	ExactCounts    ReplayCounts `json:"exact_replay_counts"`
	DenominatorTotal int       `json:"denominator_total"`
}

type Evaluation struct {
	Manifest        DriftManifest
	CausalEvidence  CausalEvidence
	Replay          ReplayReceipt
	Decision        DecisionReceipt
	Unknowns        []Unknown
	Contradictions  []string
	HumanReport     string
}

func (e Evaluation) JSON(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
