#!/bin/bash
set -euo pipefail

cd "$(dirname $0)/.."
ginkgo -r -failOnPending -randomizeAllSpecs -randomizeSuites "$@"
