#!/bin/sh
# Build command for Cloudflare Workers Builds; also fine to run locally.
set -eu

# History pages need full git history, and CI checkouts are often shallow.
if [ "$(git rev-parse --is-shallow-repository)" = "true" ]; then
  git fetch --unshallow --quiet
fi

go test ./...
go run ./cmd/ryu build
