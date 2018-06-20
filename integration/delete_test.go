package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("delete", func() {
	var (
		footCmd        *exec.Cmd
		driverStoreDir string
	)

	BeforeEach(func() {
		driverStoreDir = tempDir("", "groot-integration-tests")
		footCmd = newFootCommand("", driverStoreDir, "delete", "some-handle")
	})

	JustBeforeEach(func() {
		var out []byte
		out, footCmdError = footCmd.Output()
		footCmdOutput = gbytes.BufferWithBytes(out)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
	})

	It("calls driver.Delete() with the expected args", func() {
		Expect(footCmdError).NotTo(HaveOccurred())

		var args foot.DeleteCalls
		unmarshalFile(filepath.Join(driverStoreDir, foot.DeleteArgsFileName), &args)
		Expect(args[0].BundleID).NotTo(BeEmpty())
	})

	Context("when the driver returns an error", func() {
		BeforeEach(func() {
			footCmd.Env = append(os.Environ(), "FOOT_BUNDLE_ERROR=true")
		})

		It("fails", func() {
			expectErrorOutput("delete-err")
		})
	})

	Context("when the incorrect number of args is given", func() {
		BeforeEach(func() {
			footCmd = newFootCommand("", driverStoreDir, "delete")
		})

		It("prints an error", func() {
			Expect(footCmdError).To(HaveOccurred())
			Expect(footCmdOutput).To(gbytes.Say("Incorrect number of args. Expect 1, got 0"))
		})
	})
})
