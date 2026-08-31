# Immutable release audit

The release audit is a separate, manual GitHub Actions workflow. It requires a
tag, expected commit, and asset filename, then checks the GitHub Release API
for `immutable=true`, verifies the tag is annotated and dereferences to the
expected commit, and downloads the asset to compare exact byte size and digest.

The first release audit is preserved as a `REFUTED` attempt in
`development-provenance.json` because `v0.1.0` was published before repository
immutable releases were enabled. That release is unchanged. Future releases
must pass the audit before being reported as immutable.
