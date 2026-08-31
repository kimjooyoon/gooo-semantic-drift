# Gooo Semantic Drift

Gooo Semantic Drift is a deterministic protocol for comparing two immutable
language/compiler envelopes across releases. It keeps five questions separate:

1. syntax denominator evolution (`ADD`, `RETIRE`, `SPLIT`);
2. schema migration;
3. lowering and source-identity changes;
4. generated artifact changes; and
5. observable behavior under replay.

Every result is exactly `CLOSED`, `UNKNOWN`, or `REFUTED`, with precedence
`REFUTED > UNKNOWN > CLOSED`. `UNKNOWN` records stage, step, reason,
`unknown_class`, `next_operation`, and `blocked_by`. A known replay
contradiction is `REFUTED`, even if every envelope, schema, and artifact hash
is valid. Inferred identity cannot close a lowering change.

The implementation follows the metaprogramming chain:

```text
main.gooo → semantic-ir.json → semantic.gooo.go → machine receipts → human report
```

`gooo-drift conformance` emits five files for every fixture:
`drift-manifest.json`, `causal-evidence.json`, `replay-receipt.json`,
`decision-receipt.json`, and `human-report.md`. It also emits an exact-count
`conformance-index.json` and `human-summary.md` in the caller-owned output
directory.

The repository is CI-first. GitHub Actions is the authority for formatting,
build, vetting, tests, race tests, generation, and conformance. The evaluator
records `repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`; other projects may appear only as optional
immutable release-plus-digest inputs.

Development provenance is explicit: the substantive implementation was
initially pushed to `main` in commit
`e83e42611eeed30100018a98c1f1835e1f17b821` before the single implementation
PR. This known PR-first deviation is preserved in
`development-provenance.json` and keeps the process state `REFUTED`; successful
CI does not erase it.

## Fixtures

The five fixtures are intentionally small and explicit:

| fixture | expected decision | purpose |
|---|---|---|
| `equivalent-replay` | `CLOSED` | bound envelopes and equivalent replay |
| `explicit-denominator-migration` | `CLOSED` | legitimate schema, split, lowering, and artifact migration |
| `stale-unbounded` | `UNKNOWN` | stale envelope observations and no bounded corpus |
| `inferred-identity-drift` | `REFUTED` | changed identity is inferred |
| `behavior-regression` | `REFUTED` | replayed candidate behavior contradicts the base |
