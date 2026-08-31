# Conformance matrix

The fixture corpus is a protocol boundary, not a quality score. Each row
asserts one exact final state and keeps the five drift dimensions visible in
the generated manifest.

| case | envelope binding | syntax/schema | lowering identity | generated artifacts | replay | final state |
|---|---|---|---|---|---|---|
| equivalent-replay | current | unchanged | explicit and unchanged | unchanged | every bound value equal | `CLOSED` |
| explicit-denominator-migration | current | explicit `SPLIT` and schema migration | explicit old/new bindings | explicit digest transition | every bound value equal | `CLOSED` |
| stale-unbounded | stale observations | unchanged | unchanged | unchanged | corpus not bounded | `UNKNOWN` |
| inferred-identity-drift | current | unchanged | changed identity is inferred | unchanged | values equal | `REFUTED` |
| behavior-regression | current | unchanged | explicit and unchanged | unchanged | one bound value contradicts | `REFUTED` |

The evaluator checks `REFUTED` before `UNKNOWN` before `CLOSED`. The migration
row proves that a legitimate explicit denominator evolution is not called a
regression, while the regression row proves that a behavior contradiction wins
even when the immutable envelope and artifact hashes are valid.
