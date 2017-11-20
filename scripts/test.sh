#!/bin/bash
set -euo pipefail

cd "$(dirname $0)/.."
ginkgo -r -keepGoing -failOnPending -randomizeAllSpecs -randomizeSuites -p "$@"
