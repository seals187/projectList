#!/bin/sh
# Runs Go checks in Docker (no Go toolchain needed on the host).
# usage: scripts/test-go.sh ["<command>"]   default: gofmt + go vet + go test
# e.g.   scripts/test-go.sh "go test ./internal/service/ -run Lifecycle"
set -e
cd "$(dirname "$0")/.."
# web/build must exist for the go:embed in package web
mkdir -p web/build
[ -n "$(ls -A web/build 2>/dev/null)" ] || echo '<!doctype html>' > web/build/index.html
DEFAULT='test -z "$(gofmt -l .)" || { echo "not gofmt-formatted:"; gofmt -l .; exit 1; }; go vet ./... && go test ./...'
MSYS_NO_PATHCONV=1 exec docker run --rm -v "$(pwd -W 2>/dev/null || pwd):/src" -w /src \
  -v projectlist-gomod:/go/pkg/mod -v projectlist-gocache:/root/.cache/go-build -e CGO_ENABLED=0 \
  golang:1.27-alpine sh -c "${1:-$DEFAULT}"
