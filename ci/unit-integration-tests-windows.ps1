$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

$env:GOBIN = $PWD
$env:PATH = $env:GOBIN +";" + $env:PATH

go version

Write-Host "Installing Ginkgo"
go install github.com/onsi/ginkgo/ginkgo@latest
if ($LastExitCode -ne 0) {
    throw "Ginkgo installation process returned error code: $LastExitCode"
}

./scripts/test.ps1 # -race
Exit $LastExitCode
