package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("groot", func() {
	Describe("delete", func() {
		var (
			handle = "some-handle"
			env    []string
			stdout *bytes.Buffer
		)

		argFilePath := func(filename string) string {
			return filepath.Join(tmpDir, filename)
		}

		readTestArgsFile := func(filename string, ptr interface{}) {
			content := readFile(argFilePath(filename))
			Expect(json.Unmarshal(content, ptr)).To(Succeed())
		}

		BeforeEach(func() {
			tmpDir = tempDir("", "groot-integration-tests")

			env = []string{}
			stdout = new(bytes.Buffer)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		runFootCmd := func() error {
			footArgv := []string{"--driver-store", tmpDir, "delete", handle}
			footCmd := exec.Command(footBinPath, footArgv...)
			footCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
			footCmd.Env = append(os.Environ(), env...)
			return footCmd.Run()
		}

		It("calls driver.Delete() with the expected args", func() {
			Expect(runFootCmd()).To(Succeed())
			var args foot.DeleteCalls
			readTestArgsFile(foot.DeleteArgsFileName, &args)
			Expect(args[0].BundleID).NotTo(BeEmpty())
		})

		Context("when the driver returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_BUNDLE_ERROR=true")
			})

			It("fails", func() {
				_ = runFootCmd()
				Expect(stdout.String()).To(ContainSubstring("delete-err\n"))
			})
		})
	})
})
