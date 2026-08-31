package drift

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConformanceFixtures(t *testing.T) {
	_, file, _, ok := runtimeCaller()
	if !ok { t.Fatal("runtime caller unavailable") }
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	meta, err := LoadMeta(root)
	if err != nil { t.Fatal(err) }
	cases := map[string]string{
		"equivalent-replay.json": StateClosed,
		"explicit-denominator-migration.json": StateClosed,
		"stale-unbounded.json": StateUnknown,
		"inferred-identity-drift.json": StateRefuted,
		"behavior-regression.json": StateRefuted,
	}
	for name, expected := range cases {
		raw, readErr := os.ReadFile(filepath.Join(root, "fixtures/cases", name))
		if readErr != nil { t.Fatal(readErr) }
		result := Evaluate(raw, meta)
		if result.Decision.State != expected { t.Errorf("%s: expected %s, got %s", name, expected, result.Decision.State) }
		for _, unknown := range result.Unknowns { if unknown.Stage == "" || unknown.Step == "" || unknown.Reason == "" || unknown.UnknownClass == "" || unknown.NextOperation == "" || len(unknown.BlockedBy) == 0 { t.Errorf("%s: malformed UNKNOWN record: %+v", name, unknown) } }
	}
}

func TestDecisionPrecedence(t *testing.T) {
	evaluation := newEvaluation(Meta{Contract: Contract{Total: 15}}, "sha256:"+"0000000000000000000000000000000000000000000000000000000000000000")
	unknown(&evaluation, "TEST", Unknown{Stage: "TEST", Step: "WAIT", Reason: "WAITING", UnknownClass: "EVIDENCE_UNAVAILABLE", NextOperation: "SUPPLY_EVIDENCE", BlockedBy: []string{"input"}})
	refute(&evaluation, "TEST", "KNOWN_CONTRADICTION", "known contradiction", []string{"input"})
	finished := finish(evaluation)
	if finished.Decision.State != StateRefuted { t.Fatalf("expected REFUTED precedence, got %s", finished.Decision.State) }
}

// runtimeCaller is kept as a tiny wrapper so tests do not need any generated
// path or working-directory assumptions beyond the repository root.
func runtimeCaller() (pc uintptr, file string, line int, ok bool) {
	return runtime.Caller(0)
}
