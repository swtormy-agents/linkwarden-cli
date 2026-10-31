#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

GO_LICENSES_VERSION="v1.6.0"
SCAN="./cmd/linkwarden"

if ! command -v go-licenses >/dev/null 2>&1; then
  go install "github.com/google/go-licenses@${GO_LICENSES_VERSION}"
fi
export PATH="$(go env GOPATH)/bin:$PATH"

deny='GPL|AGPL|GNU General Public|GNU Affero'
found=0

while IFS=, read -r pkg lic _; do
  pkg="${pkg//\"/}"
  lic="${lic//\"/}"
  if echo "$lic" | grep -qiE "$deny"; then
    echo "disallowed license: $pkg ($lic)" >&2
    found=1
  fi
done < <(go-licenses csv "$SCAN")

if ! go-licenses check --allowed_licenses=MIT,Apache-2.0,BSD-3-Clause,BSD-2-Clause,ISC "$SCAN"; then
  found=1
fi

if [[ "$found" -ne 0 ]]; then
  echo "dependency license check failed (GPL/AGPL not allowed)" >&2
  exit 1
fi

echo "dependency license check passed (no GPL/AGPL)"
