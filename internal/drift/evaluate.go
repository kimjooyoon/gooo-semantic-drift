package drift

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-semantic-drift/internal/generated"
)

const (
	inputSchema = "gooo/semantic-drift/input/v1"
	replaySchema = "gooo/semantic-drift/replay-receipt/v1"
	manifestSchema = "gooo/semantic-drift/drift-manifest/v1"
	evidenceSchema = "gooo/semantic-drift/causal-evidence/v1"
	decisionSchema = "gooo/semantic-drift/decision-receipt/v1"
)

func LoadMeta(root string) (Meta, error) {
	paths := map[string]string{
		"source":    filepath.Join(root, "examples/semantic-drift/main.gooo"),
		"ir":        filepath.Join(root, "internal/generated/semantic-ir.json"),
		"generated": filepath.Join(root, "internal/generated/semantic.gooo.go"),
		"evaluator": filepath.Join(root, "internal/drift/evaluate.go"),
		"contract":  filepath.Join(root, "contracts/semantic-drift-denominator-v1.json"),
	}
	read := func(key string) ([]byte, error) { return os.ReadFile(paths[key]) }
	source, err := read("source")
	if err != nil {
		return Meta{}, err
	}
	irRaw, err := read("ir")
	if err != nil {
		return Meta{}, err
	}
	generatedRaw, err := read("generated")
	if err != nil {
		return Meta{}, err
	}
	evaluatorRaw, err := read("evaluator")
	if err != nil {
		return Meta{}, err
	}
	contract, contractRaw, err := LoadContract(paths["contract"])
	if err != nil {
		return Meta{}, err
	}
	var ir SemanticIR
	decoder := json.NewDecoder(bytes.NewReader(irRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ir); err != nil {
		return Meta{}, err
	}
	if ir.Schema != IRScheme || ir.SourcePath == "" || ir.ContractPath == "" || len(ir.Activities) != contract.Total {
		return Meta{}, errors.New("INVALID_SEMANTIC_IR")
	}
	if ir.SourceDigest != DigestBytes(source) || ir.ContractDigest != DigestBytes(contractRaw) {
		return Meta{}, errors.New("SEMANTIC_IR_INPUT_DIGEST_MISMATCH")
	}
	meta := Meta{
		Root: root,
		SourcePath: "examples/semantic-drift/main.gooo", SourceDigest: DigestBytes(source),
		SemanticIRPath: "internal/generated/semantic-ir.json", SemanticIRDigest: DigestBytes(irRaw),
		GeneratedGoPath: "internal/generated/semantic.gooo.go", GeneratedGoDigest: DigestBytes(generatedRaw),
		EvaluatorPath: "internal/drift/evaluate.go", EvaluatorDigest: DigestBytes(evaluatorRaw),
		ContractPath: "contracts/semantic-drift-denominator-v1.json", ContractDigest: DigestBytes(contractRaw),
		Contract: contract,
		AuthorityChain: AuthorityChain{
			Source: ArtifactRef{Path: "examples/semantic-drift/main.gooo", Digest: DigestBytes(source)},
			SemanticIR: ArtifactRef{Path: "internal/generated/semantic-ir.json", Digest: DigestBytes(irRaw)},
			GeneratedGo: ArtifactRef{Path: "internal/generated/semantic.gooo.go", Digest: DigestBytes(generatedRaw)},
			Evaluator: ArtifactRef{Path: "internal/drift/evaluate.go", Digest: DigestBytes(evaluatorRaw)},
		},
	}
	if err := validateGenerated(meta, ir); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

func validateGenerated(meta Meta, ir SemanticIR) error {
	if generated.SourcePath != meta.SourcePath || generated.SourceDigest != meta.SourceDigest || generated.SemanticIRDigest != meta.SemanticIRDigest || generated.ContractPath != meta.ContractPath || generated.ContractDigest != meta.ContractDigest || generated.ActivityCount != meta.Contract.Total || len(generated.Activities) != meta.Contract.Total {
		return errors.New("GENERATED_AUTHORITY_CHAIN_MISMATCH")
	}
	for index, cell := range meta.Contract.Cells {
		activity := generated.Activities[index]
		node := ir.Activities[index]
		if activity.ID != cell.ID || activity.Activity != cell.Activity || activity.Name != cell.Name || activity.Dimension != cell.Dimension || activity.Artifact != cell.GeneratedArtifact || activity.Evaluator != cell.Evaluator || node.ID != activity.ID || node.Activity != activity.Activity || node.Name != activity.Name || node.Dimension != activity.Dimension || node.Artifact != activity.Artifact || node.Evaluator != activity.Evaluator {
			return fmt.Errorf("GENERATED_ACTIVITY_BINDING_MISMATCH_%d", cell.Ordinal)
		}
	}
	return nil
}

func Evaluate(raw []byte, meta Meta) Evaluation {
	evaluation := newEvaluation(meta, DigestBytes(raw))
	var input Comparison
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		evaluation.Decision.CaseID = "MALFORMED_INPUT"
		refute(&evaluation, "INPUT", "MALFORMED_INPUT", "Input is not valid JSON for the protocol.", nil)
		return finish(evaluation)
	}
	evaluation.Manifest.CaseID = input.CaseID
	evaluation.Manifest.OptionalInputs = append([]OptionalRelease(nil), input.OptionalInputs...)
	evaluation.Decision.CaseID = input.CaseID
	evaluation.CausalEvidence.CaseID = input.CaseID
	if input.Schema != inputSchema || input.CaseID == "" {
		refute(&evaluation, "INPUT", "INVALID_INPUT_CONTRACT", "Input schema or case identifier is invalid.", nil)
	}
	if input.Authority.RepositoryWrites != 0 || input.Authority.LocalTestExecutions != 0 || input.Authority.CrossProjectRequiredGates != 0 {
		refute(&evaluation, "AUTHORITY", "AUTHORITY_ESCALATION_REFUTED", "Evaluation authority fields must remain zero.", nil)
	}
	for _, optional := range input.OptionalInputs {
		if optional.Name == "" || optional.Tag == "" || !validDigest(optional.Digest) {
			refute(&evaluation, "AUTHORITY", "INVALID_OPTIONAL_RELEASE_INPUT", "Optional release inputs require an immutable tag and digest.", []string{optional.Name})
		}
	}

	baseValid := inspectEnvelope(&evaluation, "base", input.Base)
	candidateValid := inspectEnvelope(&evaluation, "candidate", input.Candidate)
	evaluation.Manifest.Base = envelopeSummary(input.Base, baseValid)
	evaluation.Manifest.Candidate = envelopeSummary(input.Candidate, candidateValid)
	if baseValid && candidateValid {
		evaluation.Manifest.Denominator = compareDenominator(&evaluation, input)
		evaluation.Manifest.SchemaMigration = compareSchema(&evaluation, input)
		evaluation.Manifest.Lowering = compareLowering(&evaluation, input)
		evaluation.Manifest.GeneratedArtifacts = compareArtifacts(&evaluation, input)
	} else {
		evaluation.Manifest.Denominator.State = StateRefuted
		evaluation.Manifest.Denominator.Dimension = "SYNTAX_DENOMINATOR"
		evaluation.Manifest.SchemaMigration.Dimension = "SCHEMA_MIGRATION"
		evaluation.Manifest.SchemaMigration.State = StateRefuted
		evaluation.Manifest.Lowering.Dimension = "LOWERING_IDENTITY"
		evaluation.Manifest.Lowering.State = StateRefuted
		evaluation.Manifest.GeneratedArtifacts.Dimension = "GENERATED_ARTIFACT"
		evaluation.Manifest.GeneratedArtifacts.State = StateRefuted
	}
	evaluation.Replay = compareReplay(&evaluation, input)
	evaluation.Manifest.ObservableBehavior = DimensionResult{
		Dimension: "OBSERVABLE_BEHAVIOR", State: evaluation.Replay.State,
		Changed: evaluation.Replay.Counts.Contradiction > 0,
		Reason: evaluation.Replay.Reason,
	}
	return finish(evaluation)
}

func newEvaluation(meta Meta, inputDigest string) Evaluation {
	authority := AuthorityReport{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, ReadOnly: true}
	return Evaluation{
		Manifest: DriftManifest{
			Schema: manifestSchema, InputDigest: inputDigest, Precedence: append([]string(nil), Precedence...),
			DenominatorTotal: meta.Contract.Total, Authority: authority, AuthorityChain: meta.AuthorityChain,
			Inventory: Inventory{RootREADMEExcluded: true, Violations: []string{}}, OptionalInputs: []OptionalRelease{},
			Denominator: DenominatorResult{Dimension: "SYNTAX_DENOMINATOR", Events: []DenominatorEvent{}},
			SchemaMigration: SchemaResult{Dimension: "SCHEMA_MIGRATION"},
			Lowering: LoweringResult{Dimension: "LOWERING_IDENTITY", ChangedSourceIDs: []string{}},
			GeneratedArtifacts: ArtifactResult{Dimension: "GENERATED_ARTIFACT", ChangedNames: []string{}},
		},
		CausalEvidence: CausalEvidence{Schema: evidenceSchema, InputDigest: inputDigest, Events: []EvidenceEvent{}},
		Replay: ReplayReceipt{Schema: replaySchema, ExpectedInputIDs: []string{}, Observations: []ReplayObservationResult{}},
		Decision: DecisionReceipt{Schema: decisionSchema, InputDigest: inputDigest, Precedence: append([]string(nil), Precedence...), Unknowns: []Unknown{}, Contradictions: []string{}, DenominatorTotal: meta.Contract.Total},
		Unknowns: []Unknown{}, Contradictions: []string{},
	}
}

func inspectEnvelope(evaluation *Evaluation, label string, binding EnvelopeBinding) bool {
	envelope := binding.Envelope
	if envelope.Schema != EnvelopeSchema || envelope.EnvelopeID == "" {
		refute(evaluation, "IMMUTABLE_ENVELOPE", "INVALID_ENVELOPE_CONTRACT", "Envelope schema or identifier is invalid.", []string{label})
		return false
	}
	if !validDigest(envelope.ReleaseDigest) || envelopeDigest(envelope) != envelope.ReleaseDigest {
		refute(evaluation, "IMMUTABLE_ENVELOPE", "INVALID_ENVELOPE_DIGEST", "Declared release digest does not bind the envelope content.", []string{label})
		return false
	}
	if !validDigest(envelope.LanguageRelease.Digest) || !validDigest(envelope.CompilerRelease.Digest) || !validDigest(envelope.SchemaFingerprint.Digest) {
		refute(evaluation, "IMMUTABLE_ENVELOPE", "INVALID_RELEASE_COMPONENT_DIGEST", "Language, compiler, and schema digests must be valid.", []string{label})
		return false
	}
	if !orderedSyntaxCells(envelope.SyntaxDenominator) || !uniqueLowering(envelope.Lowering) || !uniqueArtifacts(envelope.GeneratedArtifacts) {
		refute(evaluation, "IMMUTABLE_ENVELOPE", "NONDETERMINISTIC_ENVELOPE_ORDER", "Envelope collections must be ordered and unique.", []string{label})
		return false
	}
	for _, cell := range envelope.SyntaxDenominator {
		if cell.Ordinal < 1 || cell.ID == "" || cell.CanonicalName == "" || !validDigest(cell.SemanticsDigest) {
			refute(evaluation, "IMMUTABLE_ENVELOPE", "INVALID_SYNTAX_CELL", "Syntax denominator cells require explicit identity and digest.", []string{label, cell.ID})
			return false
		}
	}
	for _, identity := range envelope.Lowering {
		if identity.SourceID == "" || identity.TargetID == "" || (identity.IdentityMode != "explicit" && identity.IdentityMode != "inferred") || !validDigest(identity.MappingDigest) {
			refute(evaluation, "IMMUTABLE_ENVELOPE", "INVALID_LOWERING_IDENTITY", "Lowering entries require a declared identity mode and digest.", []string{label, identity.SourceID})
			return false
		}
	}
	for _, artifact := range envelope.GeneratedArtifacts {
		if artifact.Name == "" || artifact.Kind == "" || !validDigest(artifact.Digest) {
			refute(evaluation, "IMMUTABLE_ENVELOPE", "INVALID_GENERATED_ARTIFACT", "Generated artifacts require a name, kind, and digest.", []string{label, artifact.Name})
			return false
		}
	}
	if binding.ObservedDigest != envelope.ReleaseDigest {
		unknown(evaluation, "IMMUTABLE_ENVELOPE", Unknown{Stage: "ENVELOPE", Step: "VERIFY_OBSERVED_RELEASE_DIGEST", Reason: "STALE_IMMUTABLE_ENVELOPE", UnknownClass: "STALE_RELEASE", NextOperation: "PIN_OBSERVED_RELEASE_DIGEST", BlockedBy: []string{label + "-observed-digest"}})
	}
	return true
}

func envelopeSummary(binding EnvelopeBinding, valid bool) EnvelopeSummary {
	return EnvelopeSummary{
		EnvelopeID: binding.Envelope.EnvelopeID, LanguageVersion: binding.Envelope.LanguageRelease.Version,
		CompilerVersion: binding.Envelope.CompilerRelease.Version, ReleaseDigest: binding.Envelope.ReleaseDigest,
		ObservedDigest: binding.ObservedDigest, DigestValid: valid,
		ObservationBound: valid && binding.ObservedDigest == binding.Envelope.ReleaseDigest,
	}
}

func compareDenominator(evaluation *Evaluation, input Comparison) DenominatorResult {
	base := input.Base.Envelope.SyntaxDenominator
	candidate := input.Candidate.Envelope.SyntaxDenominator
	result := DenominatorResult{Dimension: "SYNTAX_DENOMINATOR", State: StateClosed, BaseCellCount: len(base), CandidateCellCount: len(candidate), Events: []DenominatorEvent{}}
	baseByID := map[string]SyntaxCell{}
	candidateByID := map[string]SyntaxCell{}
	for _, cell := range base { baseByID[cell.ID] = cell }
	for _, cell := range candidate { candidateByID[cell.ID] = cell }
	added, retired, redefined := []string{}, []string{}, []string{}
	for id, cell := range candidateByID {
		old, ok := baseByID[id]
		if !ok { added = append(added, id) } else if old.CanonicalName != cell.CanonicalName || old.SemanticsDigest != cell.SemanticsDigest { redefined = append(redefined, id) }
	}
	for id := range baseByID { if _, ok := candidateByID[id]; !ok { retired = append(retired, id) } }
	sort.Strings(added); sort.Strings(retired); sort.Strings(redefined)
	result.Added, result.Retired, result.Redefined = len(added), len(retired), len(redefined)
	result.Changed = len(added)+len(retired)+len(redefined) > 0
	if !result.Changed {
		if len(input.DenominatorMigrations) > 0 { refute(evaluation, "SYNTAX_DENOMINATOR", "UNEXPECTED_DENOMINATOR_MIGRATION", "A migration record is present without denominator evolution.", nil); result.State = StateRefuted; result.Reason = "UNEXPECTED_DENOMINATOR_MIGRATION" } else { result.Reason = "NO_SYNTAX_DENOMINATOR_CHANGE" }
		return result
	}
	if len(redefined) > 0 { unknown(evaluation, "SYNTAX_DENOMINATOR", Unknown{Stage: "SYNTAX_DENOMINATOR", Step: "BIND_SYNTAX_REDEFINITION", Reason: "SYNTAX_REDEFINITION_UNBOUND", UnknownClass: "MIGRATION_EVIDENCE_UNAVAILABLE", NextOperation: "PROVIDE_EXPLICIT_REDEFINITION_MIGRATION", BlockedBy: []string{"syntax-redefinition"}}) }
	if len(input.DenominatorMigrations) == 0 { unknown(evaluation, "SYNTAX_DENOMINATOR", Unknown{Stage: "SYNTAX_DENOMINATOR", Step: "REQUIRE_EXPLICIT_MIGRATION", Reason: "DENOMINATOR_MIGRATION_UNBOUND", UnknownClass: "MIGRATION_EVIDENCE_UNAVAILABLE", NextOperation: "PROVIDE_ADD_RETIRE_OR_SPLIT_RECORDS", BlockedBy: []string{"denominator-migration"}}); result.State = StateUnknown; result.Reason = "DENOMINATOR_MIGRATION_UNBOUND"; return result }
	addedSet, retiredSet := make(map[string]bool), make(map[string]bool)
	for _, id := range added { addedSet[id] = true }
	for _, id := range retired { retiredSet[id] = true }
	uncoveredAdded, uncoveredRetired := len(added), len(retired)
	coveredAdded, coveredRetired := map[string]bool{}, map[string]bool{}
	for _, migration := range input.DenominatorMigrations {
		if migration.Reason == "" || !validDigest(migration.EvidenceDigest) { refute(evaluation, "SYNTAX_DENOMINATOR", "INVALID_DENOMINATOR_MIGRATION", "Every migration record needs a reason and immutable evidence digest.", nil); continue }
		if len(migration.FromIDs) == 0 && migration.Kind == "ADD" && len(migration.ToIDs) == 1 && addedSet[migration.ToIDs[0]] {
			if coveredAdded[migration.ToIDs[0]] { refute(evaluation, "SYNTAX_DENOMINATOR", "DUPLICATE_DENOMINATOR_MIGRATION", "A changed syntax identity is covered more than once.", migration.ToIDs) } else { coveredAdded[migration.ToIDs[0]] = true; uncoveredAdded--; result.Events = append(result.Events, DenominatorEvent{Kind: migration.Kind, FromIDs: migration.FromIDs, ToIDs: migration.ToIDs, Reason: migration.Reason, EvidenceDigest: migration.EvidenceDigest}); evidence(evaluation, EvidenceEvent{Dimension: "SYNTAX_DENOMINATOR", Code: "EXPLICIT_" + migration.Kind, State: StateClosed, EvidenceDigests: []string{migration.EvidenceDigest}, BoundInputs: append(append([]string{}, migration.FromIDs...), migration.ToIDs...), Reason: migration.Reason}) }
			continue
		}
		if migration.Kind == "RETIRE" && len(migration.FromIDs) == 1 && len(migration.ToIDs) == 0 && retiredSet[migration.FromIDs[0]] {
			if coveredRetired[migration.FromIDs[0]] { refute(evaluation, "SYNTAX_DENOMINATOR", "DUPLICATE_DENOMINATOR_MIGRATION", "A changed syntax identity is covered more than once.", migration.FromIDs) } else { coveredRetired[migration.FromIDs[0]] = true; uncoveredRetired--; result.Events = append(result.Events, DenominatorEvent{Kind: migration.Kind, FromIDs: migration.FromIDs, ToIDs: migration.ToIDs, Reason: migration.Reason, EvidenceDigest: migration.EvidenceDigest}); evidence(evaluation, EvidenceEvent{Dimension: "SYNTAX_DENOMINATOR", Code: "EXPLICIT_" + migration.Kind, State: StateClosed, EvidenceDigests: []string{migration.EvidenceDigest}, BoundInputs: append(append([]string{}, migration.FromIDs...), migration.ToIDs...), Reason: migration.Reason}) }
			continue
		}
		if migration.Kind == "SPLIT" && len(migration.FromIDs) == 1 && len(migration.ToIDs) >= 2 && retiredSet[migration.FromIDs[0]] {
			validTargets := true
			for _, id := range migration.ToIDs { if !addedSet[id] || coveredAdded[id] { validTargets = false } }
			if !validTargets || coveredRetired[migration.FromIDs[0]] { refute(evaluation, "SYNTAX_DENOMINATOR", "INVALID_DENOMINATOR_SPLIT", "A split must cover one retired cell and each newly added cell exactly once.", append(append([]string{}, migration.FromIDs...), migration.ToIDs...)) } else { coveredRetired[migration.FromIDs[0]] = true; uncoveredRetired--; result.Split++; for _, id := range migration.ToIDs { coveredAdded[id] = true; uncoveredAdded-- }; result.Events = append(result.Events, DenominatorEvent{Kind: migration.Kind, FromIDs: migration.FromIDs, ToIDs: migration.ToIDs, Reason: migration.Reason, EvidenceDigest: migration.EvidenceDigest}); evidence(evaluation, EvidenceEvent{Dimension: "SYNTAX_DENOMINATOR", Code: "EXPLICIT_" + migration.Kind, State: StateClosed, EvidenceDigests: []string{migration.EvidenceDigest}, BoundInputs: append(append([]string{}, migration.FromIDs...), migration.ToIDs...), Reason: migration.Reason}) }
			continue
		}
		refute(evaluation, "SYNTAX_DENOMINATOR", "INVALID_DENOMINATOR_MIGRATION", "Migration kind and identity sets do not match the observed denominator change.", append(append([]string{}, migration.FromIDs...), migration.ToIDs...))
	}
	if uncoveredAdded != 0 || uncoveredRetired != 0 || len(redefined) > 0 { unknown(evaluation, "SYNTAX_DENOMINATOR", Unknown{Stage: "SYNTAX_DENOMINATOR", Step: "CLOSE_EXPLICIT_MIGRATION", Reason: "INCOMPLETE_DENOMINATOR_MIGRATION", UnknownClass: "MIGRATION_EVIDENCE_INCOMPLETE", NextOperation: "COMPLETE_MIGRATION_COVERAGE", BlockedBy: []string{"denominator-migration-coverage"}}) }
	if len(evaluation.Contradictions) > 0 { result.State = StateRefuted; result.Reason = "DENOMINATOR_MIGRATION_CONTRADICTION" } else if len(evaluation.Unknowns) > 0 { result.State = StateUnknown; result.Reason = "EXPLICIT_MIGRATION_REQUIRES_COMPLETE_BINDING" } else { result.Reason = "EXPLICIT_DENOMINATOR_MIGRATION_BOUND" }
	return result
}

func compareSchema(evaluation *Evaluation, input Comparison) SchemaResult {
	base, candidate := input.Base.Envelope.SchemaFingerprint, input.Candidate.Envelope.SchemaFingerprint
	result := SchemaResult{Dimension: "SCHEMA_MIGRATION", State: StateClosed, BaseID: base.ID, CandidateID: candidate.ID, BaseVersion: base.Version, CandidateVersion: candidate.Version, BaseDigest: base.Digest, CandidateDigest: candidate.Digest}
	result.Changed = base.ID != candidate.ID || base.Version != candidate.Version || base.Digest != candidate.Digest
	if !result.Changed {
		if input.SchemaMigration != nil { refute(evaluation, "SCHEMA_MIGRATION", "UNEXPECTED_SCHEMA_MIGRATION", "A schema migration record is present without a schema change.", nil); result.State = StateRefuted; result.Reason = "UNEXPECTED_SCHEMA_MIGRATION" } else { result.Reason = "SCHEMA_FINGERPRINT_UNCHANGED" }
		return result
	}
	if input.SchemaMigration == nil { unknown(evaluation, "SCHEMA_MIGRATION", Unknown{Stage: "SCHEMA_MIGRATION", Step: "REQUIRE_EXPLICIT_MIGRATION", Reason: "SCHEMA_MIGRATION_UNBOUND", UnknownClass: "MIGRATION_EVIDENCE_UNAVAILABLE", NextOperation: "PROVIDE_SCHEMA_MIGRATION_RECEIPT", BlockedBy: []string{"schema-migration"}}); result.State = StateUnknown; result.Reason = "SCHEMA_MIGRATION_UNBOUND"; return result }
	migration := input.SchemaMigration
	if migration.FromVersion != base.Version || migration.ToVersion != candidate.Version || migration.FromDigest != base.Digest || migration.ToDigest != candidate.Digest || migration.Reason == "" || !validDigest(migration.EvidenceDigest) { refute(evaluation, "SCHEMA_MIGRATION", "INVALID_SCHEMA_MIGRATION", "Explicit schema migration does not bind the two schema fingerprints.", nil); result.State = StateRefuted; result.Reason = "INVALID_SCHEMA_MIGRATION"; return result }
	evidence(evaluation, EvidenceEvent{Dimension: "SCHEMA_MIGRATION", Code: "EXPLICIT_SCHEMA_MIGRATION", State: StateClosed, EvidenceDigests: []string{migration.EvidenceDigest}, BoundInputs: []string{base.Digest, candidate.Digest}, Reason: migration.Reason})
	result.Reason = "EXPLICIT_SCHEMA_MIGRATION_BOUND"
	return result
}

func compareLowering(evaluation *Evaluation, input Comparison) LoweringResult {
	result := LoweringResult{Dimension: "LOWERING_IDENTITY", State: StateClosed, ChangedSourceIDs: []string{}}
	base, candidate := loweringMap(input.Base.Envelope.Lowering), loweringMap(input.Candidate.Envelope.Lowering)
	ids := unionKeys(base, candidate)
	for _, id := range ids { if !sameLowering(base[id], candidate[id]) { result.Changed = true; result.ChangedSourceIDs = append(result.ChangedSourceIDs, id); if base[id].IdentityMode == "inferred" || candidate[id].IdentityMode == "inferred" { result.InferredChanges++ } } }
	if !result.Changed {
		if len(input.LoweringEvidence) > 0 { refute(evaluation, "LOWERING_IDENTITY", "UNEXPECTED_LOWERING_EVIDENCE", "Lowering evidence is present without an identity change.", nil); result.State = StateRefuted; result.Reason = "UNEXPECTED_LOWERING_EVIDENCE" } else { result.Reason = "LOWERING_IDENTITIES_UNCHANGED" }
		return result
	}
	if result.InferredChanges > 0 { refute(evaluation, "LOWERING_IDENTITY", "INFERRED_IDENTITY_DRIFT", "A changed lowering identity relies on inference instead of an explicit binding.", result.ChangedSourceIDs); result.State = StateRefuted; result.Reason = "INFERRED_IDENTITY_DRIFT"; return result }
	evidenceByID := map[string]LoweringEvidence{}
	for _, item := range input.LoweringEvidence { if _, exists := evidenceByID[item.SourceID]; exists { refute(evaluation, "LOWERING_IDENTITY", "DUPLICATE_LOWERING_EVIDENCE", "A lowering identity change is covered more than once.", []string{item.SourceID}) }; evidenceByID[item.SourceID] = item }
	for _, id := range result.ChangedSourceIDs {
		item, ok := evidenceByID[id]
		if !ok { unknown(evaluation, "LOWERING_IDENTITY", Unknown{Stage: "LOWERING_IDENTITY", Step: "BIND_EXPLICIT_LOWERING_CHANGE", Reason: "LOWERING_CHANGE_UNBOUND", UnknownClass: "IDENTITY_EVIDENCE_UNAVAILABLE", NextOperation: "PROVIDE_EXPLICIT_LOWERING_MAPPING", BlockedBy: []string{"lowering:" + id}}); continue }
		if item.FromTargetID != base[id].TargetID || item.ToTargetID != candidate[id].TargetID || item.Reason == "" || !validDigest(item.EvidenceDigest) { refute(evaluation, "LOWERING_IDENTITY", "INVALID_LOWERING_EVIDENCE", "Lowering evidence does not bind the observed identity change.", []string{id}); continue }
		result.ExplicitChanges++
		evidence(evaluation, EvidenceEvent{Dimension: "LOWERING_IDENTITY", Code: "EXPLICIT_LOWERING_CHANGE", State: StateClosed, EvidenceDigests: []string{item.EvidenceDigest}, BoundInputs: []string{id}, Reason: item.Reason})
	}
	for id := range evidenceByID { if !contains(result.ChangedSourceIDs, id) { refute(evaluation, "LOWERING_IDENTITY", "UNBOUND_LOWERING_EVIDENCE", "Lowering evidence references an unchanged source identity.", []string{id}) } }
	if len(evaluation.Contradictions) > 0 { result.State = StateRefuted; result.Reason = "LOWERING_IDENTITY_CONTRADICTION" } else if result.ExplicitChanges != len(result.ChangedSourceIDs) { result.State = StateUnknown; result.Reason = "LOWERING_CHANGE_UNBOUND" } else { result.Reason = "EXPLICIT_LOWERING_CHANGES_BOUND" }
	return result
}

func compareArtifacts(evaluation *Evaluation, input Comparison) ArtifactResult {
	result := ArtifactResult{Dimension: "GENERATED_ARTIFACT", State: StateClosed, ChangedNames: []string{}}
	base, candidate := artifactMap(input.Base.Envelope.GeneratedArtifacts), artifactMap(input.Candidate.Envelope.GeneratedArtifacts)
	for _, name := range unionKeys(base, candidate) { if base[name].Digest != candidate[name].Digest || base[name].Kind != candidate[name].Kind { result.Changed = true; result.ChangedNames = append(result.ChangedNames, name) } }
	if !result.Changed {
		if len(input.ArtifactEvidence) > 0 { refute(evaluation, "GENERATED_ARTIFACT", "UNEXPECTED_ARTIFACT_EVIDENCE", "Artifact evidence is present without an artifact change.", nil); result.State = StateRefuted; result.Reason = "UNEXPECTED_ARTIFACT_EVIDENCE" } else { result.Reason = "GENERATED_ARTIFACTS_UNCHANGED" }
		return result
	}
	evidenceByName := map[string]ArtifactEvidence{}
	for _, item := range input.ArtifactEvidence { if _, exists := evidenceByName[item.Name]; exists { refute(evaluation, "GENERATED_ARTIFACT", "DUPLICATE_ARTIFACT_EVIDENCE", "An artifact change is covered more than once.", []string{item.Name}) }; evidenceByName[item.Name] = item }
	sort.Strings(result.ChangedNames)
	for _, name := range result.ChangedNames {
		item, ok := evidenceByName[name]
		if !ok { unknown(evaluation, "GENERATED_ARTIFACT", Unknown{Stage: "GENERATED_ARTIFACT", Step: "BIND_GENERATED_CHANGE", Reason: "GENERATED_ARTIFACT_CHANGE_UNBOUND", UnknownClass: "ARTIFACT_EVIDENCE_UNAVAILABLE", NextOperation: "PROVIDE_GENERATED_ARTIFACT_RECEIPT", BlockedBy: []string{"artifact:" + name}}); continue }
		from, to := "", ""
		if value, exists := base[name]; exists { from = value.Digest }
		if value, exists := candidate[name]; exists { to = value.Digest }
		if item.FromDigest != from || item.ToDigest != to || item.Reason == "" || !validDigest(item.EvidenceDigest) { refute(evaluation, "GENERATED_ARTIFACT", "INVALID_ARTIFACT_EVIDENCE", "Artifact evidence does not bind the observed digest transition.", []string{name}); continue }
		result.ExplicitChanges++
		evidence(evaluation, EvidenceEvent{Dimension: "GENERATED_ARTIFACT", Code: "EXPLICIT_ARTIFACT_CHANGE", State: StateClosed, EvidenceDigests: []string{item.EvidenceDigest}, BoundInputs: []string{name}, Reason: item.Reason})
	}
	for name := range evidenceByName { if !contains(result.ChangedNames, name) { refute(evaluation, "GENERATED_ARTIFACT", "UNBOUND_ARTIFACT_EVIDENCE", "Artifact evidence references an unchanged artifact.", []string{name}) } }
	if len(evaluation.Contradictions) > 0 { result.State = StateRefuted; result.Reason = "GENERATED_ARTIFACT_CONTRADICTION" } else if result.ExplicitChanges != len(result.ChangedNames) { result.State = StateUnknown; result.Reason = "GENERATED_ARTIFACT_CHANGE_UNBOUND" } else { result.Reason = "EXPLICIT_GENERATED_ARTIFACT_CHANGES_BOUND" }
	return result
}

func compareReplay(evaluation *Evaluation, input Comparison) ReplayReceipt {
	replay := input.Replay
	receipt := ReplayReceipt{Schema: replaySchema, CorpusID: replay.CorpusID, Bound: replay.Bound, ExpectedInputIDs: append([]string(nil), replay.ExpectedInputIDs...), Observations: []ReplayObservationResult{}, Counts: ReplayCounts{}}
	if !replay.Bound || replay.CorpusID == "" || len(replay.ExpectedInputIDs) == 0 {
		unknown(evaluation, "OBSERVABLE_BEHAVIOR", Unknown{Stage: "REPLAY", Step: "BIND_REPLAY_CORPUS", Reason: "REPLAY_CORPUS_UNBOUNDED", UnknownClass: "REPLAY_EVIDENCE_UNAVAILABLE", NextOperation: "PROVIDE_BOUNDED_REPLAY_CORPUS", BlockedBy: []string{"replay-corpus"}})
		receipt.State, receipt.Reason = StateUnknown, "REPLAY_CORPUS_UNBOUNDED"
		return receipt
	}
	if !sortedUnique(replay.ExpectedInputIDs) { refute(evaluation, "OBSERVABLE_BEHAVIOR", "NONDETERMINISTIC_REPLAY_INPUT_ORDER", "Expected replay inputs must be sorted and unique.", nil) }
	expected := map[string]bool{}
	for _, id := range replay.ExpectedInputIDs { expected[id] = true }
	seen := map[string]bool{}
	for _, observation := range replay.Observations {
		receipt.Counts.Total++
		result := ReplayObservationResult{InputID: observation.InputID, InputDigest: observation.InputDigest, BaseValueDigest: observation.Base.ValueDigest, CandidateValueDigest: observation.Candidate.ValueDigest}
		if seen[observation.InputID] || !expected[observation.InputID] || observation.ReplayStatus != "REPLAYED" || !validDigest(observation.InputDigest) || !validDigest(observation.Base.ValueDigest) || !validDigest(observation.Candidate.ValueDigest) || observation.Base.EnvelopeDigest != input.Base.Envelope.ReleaseDigest || observation.Candidate.EnvelopeDigest != input.Candidate.Envelope.ReleaseDigest {
			refute(evaluation, "OBSERVABLE_BEHAVIOR", "REPLAY_BINDING_CONTRADICTION", "Replay observation is duplicated, unlisted, malformed, or bound to another envelope.", []string{observation.InputID})
			result.Relation = "REFUTED"
		} else if observation.Base.ValueDigest != observation.Candidate.ValueDigest {
			receipt.Counts.Contradiction++
			result.Relation = "CONTRADICTION"
			refute(evaluation, "OBSERVABLE_BEHAVIOR", "OBSERVABLE_BEHAVIOR_CONTRADICTION", "Replayed candidate behavior differs from the bound base observation.", []string{observation.InputID})
		} else {
			receipt.Counts.Equivalent++
			result.Relation = "EQUIVALENT"
		}
		seen[observation.InputID] = true
		receipt.Observations = append(receipt.Observations, result)
	}
	if len(replay.Observations) != len(replay.ExpectedInputIDs) { unknown(evaluation, "OBSERVABLE_BEHAVIOR", Unknown{Stage: "REPLAY", Step: "COMPLETE_REPLAY_CORPUS", Reason: "REPLAY_CORPUS_INCOMPLETE", UnknownClass: "REPLAY_EVIDENCE_INCOMPLETE", NextOperation: "REPLAY_EVERY_EXPECTED_INPUT", BlockedBy: []string{"replay-input-count"}}) }
	for _, id := range replay.ExpectedInputIDs { if !seen[id] { receipt.Counts.Unknown++; unknown(evaluation, "OBSERVABLE_BEHAVIOR", Unknown{Stage: "REPLAY", Step: "OBSERVE_REPLAYED_INPUT", Reason: "REPLAY_INPUT_UNOBSERVED", UnknownClass: "REPLAY_EVIDENCE_INCOMPLETE", NextOperation: "REPLAY_EXPECTED_INPUT", BlockedBy: []string{"replay:" + id}}) } }
	if len(evaluation.Contradictions) > 0 { receipt.State, receipt.Reason = StateRefuted, "OBSERVABLE_BEHAVIOR_CONTRADICTION" } else if len(evaluation.Unknowns) > 0 { receipt.State, receipt.Reason = StateUnknown, "REPLAY_EVIDENCE_INCOMPLETE" } else { receipt.State, receipt.Reason = StateClosed, "ALL_BOUND_REPLAY_OBSERVATIONS_EQUIVALENT" }
	return receipt
}

func finish(evaluation Evaluation) Evaluation {
	state, reason := StateClosed, "ALL_FIVE_DIMENSIONS_BOUND_AND_REPLAYED"
	if len(evaluation.Contradictions) > 0 { state, reason = StateRefuted, evaluation.Contradictions[0] } else if len(evaluation.Unknowns) > 0 { state, reason = StateUnknown, evaluation.Unknowns[0].Reason }
	evaluation.Decision.State, evaluation.Decision.Reason = state, reason
	evaluation.Decision.Unknowns = append([]Unknown(nil), evaluation.Unknowns...)
	evaluation.Decision.Contradictions = append([]string(nil), evaluation.Contradictions...)
	evaluation.Decision.ExactCounts = evaluation.Replay.Counts
	evaluation.HumanReport = HumanReport(evaluation)
	return evaluation
}

func refute(evaluation *Evaluation, dimension, code, reason string, boundInputs []string) {
	if !contains(evaluation.Contradictions, code) { evaluation.Contradictions = append(evaluation.Contradictions, code) }
	evidence(evaluation, EvidenceEvent{Dimension: dimension, Code: code, State: StateRefuted, BoundInputs: boundInputs, Reason: reason})
}

func unknown(evaluation *Evaluation, dimension string, item Unknown) {
	if item.Stage == "" || item.Step == "" || item.Reason == "" || item.UnknownClass == "" || item.NextOperation == "" || len(item.BlockedBy) == 0 { refute(evaluation, dimension, "MALFORMED_UNKNOWN_RECORD", "Every UNKNOWN record must carry stage, step, reason, unknown_class, next_operation, and blocked_by.", nil); return }
	for _, existing := range evaluation.Unknowns { if existing.Reason == item.Reason && strings.Join(existing.BlockedBy, ",") == strings.Join(item.BlockedBy, ",") { return } }
	evaluation.Unknowns = append(evaluation.Unknowns, item)
	evidence(evaluation, EvidenceEvent{Dimension: dimension, Code: item.Reason, State: StateUnknown, BoundInputs: item.BlockedBy, Reason: item.Reason})
}

func evidence(evaluation *Evaluation, event EvidenceEvent) { evaluation.CausalEvidence.Events = append(evaluation.CausalEvidence.Events, event) }

func loweringMap(values []LoweringIdentity) map[string]LoweringIdentity { result := map[string]LoweringIdentity{}; for _, value := range values { result[value.SourceID] = value }; return result }
func artifactMap(values []GeneratedArtifact) map[string]GeneratedArtifact { result := map[string]GeneratedArtifact{}; for _, value := range values { result[value.Name] = value }; return result }
func sameLowering(base, candidate LoweringIdentity) bool { return base.SourceID == candidate.SourceID && base.TargetID == candidate.TargetID && base.IdentityMode == candidate.IdentityMode && base.MappingDigest == candidate.MappingDigest }
func unionKeys[A any](left, right map[string]A) []string { result := make([]string, 0, len(left)+len(right)); seen := map[string]bool{}; for key := range left { seen[key] = true; result = append(result, key) }; for key := range right { if !seen[key] { result = append(result, key) } }; sort.Strings(result); return result }
func contains(values []string, target string) bool { for _, value := range values { if value == target { return true } }; return false }
func sortedUnique(values []string) bool { copy := append([]string(nil), values...); sort.Strings(copy); if len(copy) != len(values) { return false }; for index := range values { if values[index] != copy[index] || index > 0 && values[index] == values[index-1] { return false } }; return true }
func orderedSyntaxCells(values []SyntaxCell) bool { seen := map[string]bool{}; for index, value := range values { if value.Ordinal != index+1 || seen[value.ID] { return false }; seen[value.ID] = true }; return true }
func uniqueLowering(values []LoweringIdentity) bool { seen := map[string]bool{}; for _, value := range values { if seen[value.SourceID] { return false }; seen[value.SourceID] = true }; return sortedStrings(values, func(value LoweringIdentity) string { return value.SourceID }) }
func uniqueArtifacts(values []GeneratedArtifact) bool { seen := map[string]bool{}; for _, value := range values { if seen[value.Name] { return false }; seen[value.Name] = true }; return sortedStrings(values, func(value GeneratedArtifact) string { return value.Name }) }
func sortedStrings[T any](values []T, key func(T) string) bool { for index := 1; index < len(values); index++ { if key(values[index-1]) >= key(values[index]) { return false } }; return true }
