#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
go build -o linkwarden ./cmd/linkwarden
echo "built ./linkwarden"
