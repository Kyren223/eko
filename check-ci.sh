#!/usr/bin/env bash

set -e

trap 'echo "❌ Check failed"; exit 1' ERR

test -z "$(go fmt ./...)"
staticcheck ./...
go test --cover ./...
gosec ./...

echo "✅ All checks passed"
