#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
failed=0
while IFS= read -r -d '' source; do
  diff=$(gofmt -d "$source")
  if [[ -n "$diff" ]]; then
    printf '%s\n' "$diff"
    failed=1
  fi
done < <(find cmd internal -type f -name '*.go' -print0)
exit "$failed"
