package drift

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func HumanReport(evaluation Evaluation) string {
	var builder strings.Builder
	builder.WriteString("# Gooo semantic drift report\n\n")
	fmt.Fprintf(&builder, "- decision: `%s`\n", evaluation.Decision.State)
	fmt.Fprintf(&builder, "- case: `%s`\n", evaluation.Decision.CaseID)
	fmt.Fprintf(&builder, "- reason: `%s`\n", evaluation.Decision.Reason)
	fmt.Fprintf(&builder, "- denominator_total: `%d`\n", evaluation.Manifest.DenominatorTotal)
	fmt.Fprintf(&builder, "- repository_writes: `%d`\n", evaluation.Manifest.Authority.RepositoryWrites)
	fmt.Fprintf(&builder, "- local_test_executions: `%d`\n", evaluation.Manifest.Authority.LocalTestExecutions)
	fmt.Fprintf(&builder, "- cross_project_required_gates: `%d`\n", evaluation.Manifest.Authority.CrossProjectRequiredGates)
	builder.WriteString("\n## Dimensions\n\n")
	builder.WriteString("| dimension | state | changed | exact observation | reason |\n|---|---|---:|---:|---|\n")
	fmt.Fprintf(&builder, "| syntax denominator | %s | %t | %d added, %d retired, %d split, %d redefined | %s |\n", evaluation.Manifest.Denominator.State, evaluation.Manifest.Denominator.Changed, evaluation.Manifest.Denominator.Added, evaluation.Manifest.Denominator.Retired, evaluation.Manifest.Denominator.Split, evaluation.Manifest.Denominator.Redefined, evaluation.Manifest.Denominator.Reason)
	fmt.Fprintf(&builder, "| schema migration | %s | %t | base `%s` to candidate `%s` | %s |\n", evaluation.Manifest.SchemaMigration.State, evaluation.Manifest.SchemaMigration.Changed, evaluation.Manifest.SchemaMigration.BaseVersion, evaluation.Manifest.SchemaMigration.CandidateVersion, evaluation.Manifest.SchemaMigration.Reason)
	fmt.Fprintf(&builder, "| lowering identity | %s | %t | %d explicit changes, %d inferred changes | %s |\n", evaluation.Manifest.Lowering.State, evaluation.Manifest.Lowering.Changed, evaluation.Manifest.Lowering.ExplicitChanges, evaluation.Manifest.Lowering.InferredChanges, evaluation.Manifest.Lowering.Reason)
	fmt.Fprintf(&builder, "| generated artifact | %s | %t | %d changed artifacts | %s |\n", evaluation.Manifest.GeneratedArtifacts.State, evaluation.Manifest.GeneratedArtifacts.Changed, len(evaluation.Manifest.GeneratedArtifacts.ChangedNames), evaluation.Manifest.GeneratedArtifacts.Reason)
	fmt.Fprintf(&builder, "| observable behavior | %s | %t | %d total, %d equivalent, %d contradiction, %d unknown | %s |\n", evaluation.Manifest.ObservableBehavior.State, evaluation.Manifest.ObservableBehavior.Changed, evaluation.Replay.Counts.Total, evaluation.Replay.Counts.Equivalent, evaluation.Replay.Counts.Contradiction, evaluation.Replay.Counts.Unknown, evaluation.Manifest.ObservableBehavior.Reason)
	builder.WriteString("\nNo percentage, score, inferred improvement, or cache-hit claim is emitted.\n")
	builder.WriteString("\n## Replay receipt\n\n")
	fmt.Fprintf(&builder, "- corpus: `%s`\n- bound: `%t`\n- exact counts: total=%d, equivalent=%d, contradiction=%d, unknown=%d\n", evaluation.Replay.CorpusID, evaluation.Replay.Bound, evaluation.Replay.Counts.Total, evaluation.Replay.Counts.Equivalent, evaluation.Replay.Counts.Contradiction, evaluation.Replay.Counts.Unknown)
	builder.WriteString("\n## Authority chain\n\n")
	fmt.Fprintf(&builder, "`%s` (%s) -> `%s` (%s) -> `%s` (%s) -> `%s` (%s)\n", evaluation.Manifest.AuthorityChain.Source.Path, evaluation.Manifest.AuthorityChain.Source.Digest, evaluation.Manifest.AuthorityChain.SemanticIR.Path, evaluation.Manifest.AuthorityChain.SemanticIR.Digest, evaluation.Manifest.AuthorityChain.GeneratedGo.Path, evaluation.Manifest.AuthorityChain.GeneratedGo.Digest, evaluation.Manifest.AuthorityChain.Evaluator.Path, evaluation.Manifest.AuthorityChain.Evaluator.Digest)
	if len(evaluation.Unknowns) > 0 {
		builder.WriteString("\n## UNKNOWN records\n\n")
		for _, item := range evaluation.Unknowns {
			fmt.Fprintf(&builder, "- stage=`%s`, step=`%s`, reason=`%s`, unknown_class=`%s`, next_operation=`%s`, blocked_by=`%s`\n", item.Stage, item.Step, item.Reason, item.UnknownClass, item.NextOperation, strings.Join(item.BlockedBy, ","))
		}
	}
	if len(evaluation.Contradictions) > 0 {
		fmt.Fprintf(&builder, "\n## REFUTED contradictions\n\n- %s\n", strings.Join(evaluation.Contradictions, "\n- "))
	}
	return builder.String()
}

func WriteEvaluation(outputDir string, evaluation Evaluation) error {
	if outputDir == "" || !filepath.IsAbs(outputDir) {
		return fmt.Errorf("output directory must be an absolute caller-owned path")
	}
	files := map[string]any{
		"drift-manifest.json":   evaluation.Manifest,
		"causal-evidence.json":  evaluation.CausalEvidence,
		"replay-receipt.json":   evaluation.Replay,
		"decision-receipt.json": evaluation.Decision,
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	for name, value := range files {
		raw, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outputDir, name), append(raw, '\n'), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(outputDir, "human-report.md"), []byte(evaluation.HumanReport), 0o644)
}

func WriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
