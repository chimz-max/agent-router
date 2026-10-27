#!/usr/bin/env bash
# Push packaged charts to an OCI registry without overwriting anything already published.
# Usage: hack/nscale/push-charts.sh <oci registry> <chart version> <chart archive>...
#   e.g. hack/nscale/push-charts.sh oci://registry.nscale.com/helm-charts v1.2.0-nscale.1 out/ai-gateway-*-v1.2.0-nscale.1.tgz
#
# Harbor overwrites a version on push. Before pushing anything, this pulls each chart's version:
#   - not found: push it
#   - found with the same contents: skip it, so a re-run finishes a partial publish
#   - found with different contents, or any other registry error: fail, and push nothing
# Helm 3 stamps each archive entry with the packaging time, so two packages of the same commit differ
# byte for byte. The comparison is of the extracted files instead.
# Log in first with `go tool -modfile=tools/go.mod helm registry login`.
set -euo pipefail
registry="${1:?usage: $0 <oci registry> <chart version> <chart archive>...}"
version="${2:?usage: $0 <oci registry> <chart version> <chart archive>...}"
shift 2
(($# > 0)) || { echo "usage: $0 <oci registry> <chart version> <chart archive>..." >&2; exit 1; }
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
helm() { (cd "$root" && go tool -modfile=tools/go.mod helm "$@"); }
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

pushes=()
for archive in "$@"; do
  [[ -f "$archive" ]] || { echo "ERROR: $archive does not exist" >&2; exit 1; }
  archive="$(cd "$(dirname "$archive")" && pwd)/$(basename "$archive")"
  chart="$(basename "$archive" "-$version.tgz")"
  [[ "$chart" != "$(basename "$archive")" ]] || { echo "ERROR: $archive is not a $version archive" >&2; exit 1; }
  mkdir -p "$work/$chart/published" "$work/$chart/local" "$work/$chart/remote"
  if out="$(helm pull "$registry/$chart" --version "$version" --destination "$work/$chart/published" 2>&1)"; then
    tar -xzf "$archive" -C "$work/$chart/local"
    tar -xzf "$work/$chart/published/$chart-$version.tgz" -C "$work/$chart/remote"
    if diff -r "$work/$chart/local" "$work/$chart/remote" >&2; then
      echo "$chart $version is already published with the same contents; skipping"
      continue
    fi
    echo "ERROR: $chart $version is already published with different contents" >&2
    exit 1
  fi
  if ! grep -qiE 'not found|manifest unknown|name unknown' <<<"$out"; then
    echo "ERROR: cannot tell whether $chart $version is published: $out" >&2
    exit 1
  fi
  pushes+=("$archive")
done
for archive in ${pushes[@]+"${pushes[@]}"}; do
  helm push "$archive" "$registry"
done
