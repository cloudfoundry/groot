package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("groot", func() {
	var (
		session        *gexec.Session
		footCmd        *exec.Cmd
		driverStoreDir string
	)

	Describe("delete", func() {
		BeforeEach(func() {
			driverStoreDir = tempDir("", "groot-integration-tests")
			footCmd = newFootCommand("", driverStoreDir, "delete", "some-handle")
		})

		JustBeforeEach(func() {
			session = gexecStart(footCmd).Wait()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
		})

		It("calls driver.Delete() with the expected args", func() {
			Expect(session).To(gexec.Exit(0))

			var args foot.DeleteCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.DeleteArgsFileName), &args)
			Expect(args[0].BundleID).NotTo(BeEmpty())
		})

		Context("when the driver returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_BUNDLE_ERROR=true")
			})

			It("fails", func() {
				Expect(session).NotTo(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("delete-err"))
			})
		})
	})
})
