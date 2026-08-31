#!/usr/bin/env bash
set -euo pipefail

binary=$1
root=$2
output=$3

"$binary" conformance \
  -root "$root" \
  -fixtures "$root/fixtures/cases" \
  -output-dir "$output"

jq -e '
  .schema == "gooo/semantic-drift/conformance-index/v1" and
  .denominator_total == 15 and
  .percentage_aggregation == false and
  (.cases | length) == 5 and
  ([.cases[] | select(.decision == "CLOSED")] | length) == 2 and
  ([.cases[] | select(.decision == "UNKNOWN")] | length) == 1 and
  ([.cases[] | select(.decision == "REFUTED")] | length) == 2
' "$output/conformance-index.json" >/dev/null

for case_dir in "$output"/*/; do
  test -f "$case_dir/drift-manifest.json"
  test -f "$case_dir/causal-evidence.json"
  test -f "$case_dir/replay-receipt.json"
  test -f "$case_dir/decision-receipt.json"
  test -f "$case_dir/human-report.md"
done
