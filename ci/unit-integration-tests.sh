#!/usr/bin/env bash
set -euo pipefail

export GOPATH=$PWD

go version

./src/code.cloudfoundry.org/groot/scripts/test -race
