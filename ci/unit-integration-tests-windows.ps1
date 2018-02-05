$ErrorActionPreference = "Stop";
trap { $host.SetShouldExit(1) }

$env:GOPATH = $PWD
$env:PATH = $env:GOPATH + "/bin;" + $env:PATH

go version

go get github.com/golang/protobuf
Write-Host "Installing Ginkgo"
go get github.com/onsi/ginkgo/ginkgo
if ($LastExitCode -ne 0) {
    throw "Ginkgo installation process returned error code: $LastExitCode"
}

./src/code.cloudfoundry.org/groot/scripts/test.ps1 # -race
Exit $LastExitCode
