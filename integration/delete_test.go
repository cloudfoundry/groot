package integration_test

import (
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			_, err := runFoot("", tmpDir, "delete", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			var args foot.DeleteCalls
			unmarshalFile(filepath.Join(tmpDir, foot.DeleteArgsFileName), &args)
			Expect(args[0].BundleID).NotTo(BeEmpty())
		})

		Context("when the driver returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_BUNDLE_ERROR=true")
			})

			It("fails", func() {
				stdout, err := runFoot("", tmpDir, "delete", "some-handle")
				Expect(err).To(HaveOccurred())
				Expect(stdout).To(ContainSubstring("delete-err"))
			})
		})
	})
})
