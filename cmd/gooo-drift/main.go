package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-semantic-drift/internal/drift"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "compile":
		return compile(args[1:], stdout, stderr)
	case "evaluate":
		return evaluate(args[1:], stdout, stderr)
	case "conformance":
		return conformance(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, "gooo-semantic-drift/v0.1.0")
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func compile(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourcePath := flags.String("source", "examples/semantic-drift/main.gooo", "Gooo source path")
	contractPath := flags.String("contract", "contracts/semantic-drift-denominator-v1.json", "fixed denominator path")
	outputIR := flags.String("output-ir", "", "caller-owned absolute semantic IR output")
	outputGo := flags.String("output-go", "", "caller-owned absolute generated Go output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !absolute(*outputIR) || !absolute(*outputGo) {
		fmt.Fprintln(stderr, "compile requires absolute -output-ir and -output-go paths")
		return 2
	}
	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		fmt.Fprintf(stderr, "read source: %v\n", err)
		return 1
	}
	contract, contractRaw, err := drift.LoadContract(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "read contract: %v\n", err)
		return 1
	}
	ir, err := drift.CompileSource(*sourcePath, source, contract, drift.DigestBytes(contractRaw))
	if err != nil {
		fmt.Fprintf(stderr, "compile source: %v\n", err)
		return 1
	}
	irRaw, err := drift.SemanticIRBytes(ir)
	if err != nil {
		fmt.Fprintf(stderr, "encode semantic IR: %v\n", err)
		return 1
	}
	generatedGo, err := drift.GenerateGo(ir, drift.DigestBytes(irRaw))
	if err != nil {
		fmt.Fprintf(stderr, "generate Go: %v\n", err)
		return 1
	}
	if err := writeOutput(*outputIR, irRaw); err != nil {
		fmt.Fprintf(stderr, "write semantic IR: %v\n", err)
		return 1
	}
	if err := writeOutput(*outputGo, generatedGo); err != nil {
		fmt.Fprintf(stderr, "write generated Go: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "compiled source=%s semantic_ir=%s generated_go=%s\n", *sourcePath, *outputIR, *outputGo)
	return 0
}

func evaluate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	inputPath := flags.String("input", "", "comparison input JSON")
	outputDir := flags.String("output-dir", "", "caller-owned absolute output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *inputPath == "" || !absolute(*outputDir) {
		fmt.Fprintln(stderr, "evaluate requires -input and absolute -output-dir")
		return 2
	}
	rootPath, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve root: %v\n", err)
		return 1
	}
	meta, err := drift.LoadMeta(rootPath)
	if err != nil {
		fmt.Fprintf(stderr, "load generated authority chain: %v\n", err)
		return 1
	}
	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	evaluation := drift.Evaluate(raw, meta)
	if err := drift.WriteEvaluation(*outputDir, evaluation); err != nil {
		fmt.Fprintf(stderr, "write evaluation: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s decision=%s reason=%s\n", evaluation.Decision.CaseID, evaluation.Decision.State, evaluation.Decision.Reason)
	return 0
}

func conformance(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("conformance", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	fixtureDir := flags.String("fixtures", "fixtures/cases", "fixture directory")
	outputDir := flags.String("output-dir", "", "caller-owned absolute output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !absolute(*outputDir) {
		fmt.Fprintln(stderr, "conformance requires absolute -output-dir")
		return 2
	}
	rootPath, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintf(stderr, "resolve root: %v\n", err)
		return 1
	}
	meta, err := drift.LoadMeta(rootPath)
	if err != nil {
		fmt.Fprintf(stderr, "load generated authority chain: %v\n", err)
		return 1
	}
	entries, err := os.ReadDir(*fixtureDir)
	if err != nil {
		fmt.Fprintf(stderr, "read fixtures: %v\n", err)
		return 1
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	index := conformanceIndex{Schema: "gooo/semantic-drift/conformance-index/v1", DenominatorTotal: meta.Contract.Total, Cases: []conformanceCase{}, States: map[string]int{StateClosed: 0, StateUnknown: 0, StateRefuted: 0}, PercentageAggregation: false}
	for _, name := range names {
		raw, readErr := os.ReadFile(filepath.Join(*fixtureDir, name))
		if readErr != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", name, readErr)
			return 1
		}
		evaluation := drift.Evaluate(raw, meta)
		caseID := strings.TrimSuffix(name, ".json")
		caseDir := filepath.Join(*outputDir, caseID)
		if err := drift.WriteEvaluation(caseDir, evaluation); err != nil {
			fmt.Fprintf(stderr, "write %s: %v\n", caseID, err)
			return 1
		}
		expected, ok := expectedDecision(caseID)
		if !ok || evaluation.Decision.State != expected {
			fmt.Fprintf(stderr, "%s: expected %s, got %s\n", caseID, expected, evaluation.Decision.State)
			return 1
		}
		index.States[evaluation.Decision.State]++
		index.Cases = append(index.Cases, conformanceCase{CaseID: caseID, Decision: evaluation.Decision.State, Reason: evaluation.Decision.Reason, ReplayCounts: evaluation.Replay.Counts, UnknownCount: len(evaluation.Unknowns), ContradictionCount: len(evaluation.Contradictions)})
		fmt.Fprintf(stdout, "%s decision=%s\n", caseID, evaluation.Decision.State)
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "create output: %v\n", err)
		return 1
	}
	if err := writeJSON(filepath.Join(*outputDir, "conformance-index.json"), index); err != nil {
		fmt.Fprintf(stderr, "write index: %v\n", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(*outputDir, "human-summary.md"), []byte(humanSummary(index)), 0o644); err != nil {
		fmt.Fprintf(stderr, "write summary: %v\n", err)
		return 1
	}
	return 0
}

type conformanceIndex struct {
	Schema                string            `json:"schema"`
	DenominatorTotal      int               `json:"denominator_total"`
	Cases                 []conformanceCase `json:"cases"`
	States                map[string]int    `json:"states"`
	PercentageAggregation bool              `json:"percentage_aggregation"`
}

type conformanceCase struct {
	CaseID             string             `json:"case_id"`
	Decision           string             `json:"decision"`
	Reason             string             `json:"reason"`
	ReplayCounts       drift.ReplayCounts `json:"replay_counts"`
	UnknownCount       int                `json:"unknown_count"`
	ContradictionCount int                `json:"contradiction_count"`
}

func expectedDecision(caseID string) (string, bool) {
	decisions := map[string]string{
		"equivalent-replay":              StateClosed,
		"explicit-denominator-migration": StateClosed,
		"stale-unbounded":                StateUnknown,
		"inferred-identity-drift":        StateRefuted,
		"behavior-regression":            StateRefuted,
	}
	decision, ok := decisions[caseID]
	return decision, ok
}

func humanSummary(index conformanceIndex) string {
	var builder strings.Builder
	builder.WriteString("# Gooo semantic drift conformance\n\n")
	builder.WriteString("| case | decision | replay total | equivalent | contradiction | unknown |\n|---|---|---:|---:|---:|---:|\n")
	for _, item := range index.Cases {
		fmt.Fprintf(&builder, "| %s | %s | %d | %d | %d | %d |\n", item.CaseID, item.Decision, item.ReplayCounts.Total, item.ReplayCounts.Equivalent, item.ReplayCounts.Contradiction, item.ReplayCounts.Unknown)
	}
	fmt.Fprintf(&builder, "\nFixed denominator: %d activities. States: CLOSED=%d, UNKNOWN=%d, REFUTED=%d.\n", index.DenominatorTotal, index.States[StateClosed], index.States[StateUnknown], index.States[StateRefuted])
	builder.WriteString("No percentage aggregation is performed.\n")
	return builder.String()
}

func writeOutput(path string, data []byte) error {
	if !absolute(path) {
		return fmt.Errorf("output path must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func absolute(path string) bool { return path != "" && filepath.IsAbs(path) }
func usage(writer io.Writer) {
	fmt.Fprintln(writer, "usage: gooo-drift <compile|evaluate|conformance|version>")
}

const (
	StateClosed  = drift.StateClosed
	StateUnknown = drift.StateUnknown
	StateRefuted = drift.StateRefuted
)
