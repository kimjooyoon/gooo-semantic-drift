# Semantic Drift Protocol v1

## Comparison unit

A comparison binds a base and candidate `gooo/semantic-drift/envelope/v1`.
Each envelope carries language and compiler release digests, an ordered syntax
denominator, a schema fingerprint, ordered lowering identities, and ordered
generated-artifact digests. `release_digest` is the SHA-256 digest of the
envelope with its own digest field blank. `observed_digest` must equal it before
the envelope is considered current.

The input also binds a bounded replay corpus. Every expected input must have a
`REPLAYED` base and candidate observation, each bound to the corresponding
envelope digest. Equal value digests are equivalent observations. Different
value digests are known behavior contradictions and therefore `REFUTED`.

## Independent dimensions

Syntax denominator evolution is classified only through explicit records:

- `ADD`: no base identity and one candidate identity;
- `RETIRE`: one base identity and no candidate identity;
- `SPLIT`: one retired identity and two or more added identities.

Schema changes require a migration receipt that binds both schema versions and
digests. Lowering changes require per-source explicit evidence. A changed
lowering identity with `identity_mode="inferred"` is `REFUTED`, not a guessed
mapping. Generated artifact changes require exact old and new digests plus
evidence; an artifact change is not automatically a behavior regression.

Missing migration, stale envelope, unbounded corpus, or incomplete replay is
`UNKNOWN`. A malformed or contradictory binding is `REFUTED`. Final decisions
use `REFUTED > UNKNOWN > CLOSED`, so a contradiction remains decisive even if
another dimension is incomplete.

## Authority and outputs

The fixed denominator is 15 one-to-one activities in
`contracts/semantic-drift-denominator-v1.json`. The source declaration is
compiled into semantic IR and generated Go. The evaluator reads the committed
chain, validates all chain digests, and writes only to the caller-provided
absolute output directory. No repository file is an evaluator output.

Reports contain integer observations and fixed denominators. They do not emit
percentages, scores, inferred improvement, or cache-hit-as-proof claims.
