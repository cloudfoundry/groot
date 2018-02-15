package integration_test

import (
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("groot", func() {
	Describe("delete", func() {
		BeforeEach(func() {
			tmpDir = tempDir("", "groot-integration-tests")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		It("calls driver.Delete() with the expected args", func() {
			Expect(runFoot("", tmpDir, "delete", "some-handle")).To(gexec.Exit(0))

			var args foot.DeleteCalls
			unmarshalFile(filepath.Join(tmpDir, foot.DeleteArgsFileName), &args)
			Expect(args[0].BundleID).NotTo(BeEmpty())
		})

		Context("when the driver returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_BUNDLE_ERROR=true")
			})

			It("fails", func() {
				session := runFoot("", tmpDir, "delete", "some-handle")
				Expect(session.Out).To(gbytes.Say("delete-err"))
			})
		})
	})
})
