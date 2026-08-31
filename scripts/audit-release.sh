#!/usr/bin/env bash
set -euo pipefail

tag=$1
expected_commit=$2
asset_name=$3
: "${GH_TOKEN:?GH_TOKEN is required}"

release_json=$(gh api "repos/kimjooyoon/gooo-semantic-drift/releases/tags/$tag")
immutable=$(printf '%s\n' "$release_json" | jq -r '.immutable')
test "$immutable" = true

tag_target=$(git rev-parse "$tag^{commit}")
test "$tag_target" = "$expected_commit"
test "$(git cat-file -t "$tag^{tag}")" = tag

asset=$(printf '%s\n' "$release_json" | jq -c --arg name "$asset_name" '.assets[] | select(.name == $name)')
test -n "$asset"
asset_id=$(printf '%s\n' "$asset" | jq -r '.id')
api_bytes=$(printf '%s\n' "$asset" | jq -r '.size')
api_digest=$(printf '%s\n' "$asset" | jq -r '.digest')

audit_dir=$(mktemp -d "$RUNNER_TEMP/release-audit.XXXXXX")
download="$audit_dir/$asset_name"
trap 'unlink "$download"; rmdir "$audit_dir"' EXIT
curl -fsSL -o "$download" "https://github.com/kimjooyoon/gooo-semantic-drift/releases/download/$tag/$asset_name"
downloaded_bytes=$(wc -c < "$download" | tr -d ' ')
downloaded_digest="sha256:$(shasum -a 256 "$download" | awk '{print $1}')"
test "$downloaded_bytes" = "$api_bytes"
test "$downloaded_digest" = "$api_digest"

jq -n \
  --arg tag "$tag" \
  --arg expected_commit "$expected_commit" \
  --arg tag_target "$tag_target" \
  --arg asset_id "$asset_id" \
  --arg asset_name "$asset_name" \
  --arg api_digest "$api_digest" \
  --arg downloaded_digest "$downloaded_digest" \
  --argjson bytes "$downloaded_bytes" \
  '{schema:"gooo/semantic-drift/release-audit/v1",tag:$tag,expected_commit:$expected_commit,tag_target:$tag_target,annotated_tag:true,immutable:true,asset:{id:$asset_id,name:$asset_name,bytes:$bytes,api_digest:$api_digest,downloaded_digest:$downloaded_digest,bytes_match:true,digest_match:true}}'
