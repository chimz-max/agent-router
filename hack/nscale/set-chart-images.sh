#!/usr/bin/env bash
# Point the ai-gateway-helm chart's default images at the Nscale fork's registry before packaging.
# Usage: hack/nscale/set-chart-images.sh <image repository prefix>
#   e.g. hack/nscale/set-chart-images.sh ghcr.io/nscaledev/ai-gateway
#
# Upstream's values.yaml defaults the controller and extproc images to docker.io/envoyproxy/ai-gateway-*,
# the unpatched upstream builds. release.yaml runs this on its checkout so the published chart pulls
# <prefix>-controller and <prefix>-extproc by default. The committed values keep the upstream
# defaults, which the e2e suite and `make helm-test` depend on. The run fails unless both defaults are
# rewritten and the rendered chart references no upstream image.
set -euo pipefail
prefix="${1:?usage: $0 <image repository prefix>}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
chart="$root/manifests/charts/ai-gateway-helm"
upstream="docker.io/envoyproxy/ai-gateway"
for c in controller extproc; do
  line="    repository: ${upstream}-${c}"
  n="$(grep -cxF "$line" "$chart/values.yaml" || true)"
  if [[ "$n" != 1 ]]; then
    echo "ERROR: expected one ${upstream}-${c} repository default in $chart/values.yaml, found $n" >&2
    exit 1
  fi
  sed -i.bak "s|^${line}\$|    repository: ${prefix}-${c}|" "$chart/values.yaml"
  rm "$chart/values.yaml.bak"
done
rendered="$(cd "$root" && go tool -modfile=tools/go.mod helm template "$chart")"
for c in controller extproc; do
  if ! grep -qF "${prefix}-${c}:" <<<"$rendered"; then
    echo "ERROR: rendered chart does not use ${prefix}-${c}" >&2
    exit 1
  fi
done
if grep -qF "${upstream}-" <<<"$rendered"; then
  echo "ERROR: rendered chart still references ${upstream}-*:" >&2
  grep -nF "${upstream}-" <<<"$rendered" >&2
  exit 1
fi
echo "ai-gateway-helm images now default to ${prefix}-controller and ${prefix}-extproc"
