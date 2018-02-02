package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var footBinPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	binPath, err := gexec.Build("code.cloudfoundry.org/groot/integration/cmd/foot")
	Expect(err).NotTo(HaveOccurred())
	return []byte(binPath)
}, func(footBinPathBytes []byte) {
	footBinPath = string(footBinPathBytes)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}
