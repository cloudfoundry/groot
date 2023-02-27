#!/usr/bin/env bash
set -euo pipefail

go version

go install github.com/onsi/ginkgo/ginkgo@latest

./scripts/test -race
