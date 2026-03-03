#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"

mapfile -t go_files < <(git ls-files '*.go')

if ((${#go_files[@]} > 0)); then
  echo "Formatting Go files with gofmt..."
  gofmt -w "${go_files[@]}"
else
  echo "No Go files found to format."
fi

echo "Running Go tests..."
go test ./...

echo "Verification complete."
