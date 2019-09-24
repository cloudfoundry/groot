$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

cd "$psscriptroot\.."
ginkgo -mod vendor -r -keepGoing -failOnPending -randomizeAllSpecs -randomizeSuites # "$@"
