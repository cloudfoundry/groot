package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"code.cloudfoundry.org/groot/integration/cmd/toot/toot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("groot", func() {
	Describe("create", func() {
		var (
			rootfsURI = "some-rootfs-uri"
			handle    = "some-handle"
			tempDir   string
			stdout    *bytes.Buffer

			tootCmd *exec.Cmd
			exitErr error
		)

		readTestArgsFile := func(filename string, ptr interface{}) {
			content, err := ioutil.ReadFile(filepath.Join(tempDir, filename))
			Expect(err).NotTo(HaveOccurred())
			Expect(json.Unmarshal(content, ptr)).To(Succeed())
		}

		BeforeEach(func() {
			var err error
			tempDir, err = ioutil.TempDir("", "groot-integration-tests")
			Expect(err).NotTo(HaveOccurred())

			stdout = new(bytes.Buffer)
			tootCmd = exec.Command(tootBinPath, "create", rootfsURI, handle)
			tootCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
			tootCmd.Stderr = GinkgoWriter
			tootCmd.Env = append(os.Environ(), "TOOT_BASE_DIR="+tempDir)
		})

		JustBeforeEach(func() {
			exitErr = tootCmd.Run()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		Describe("success", func() {
			JustBeforeEach(func() {
				Expect(exitErr).NotTo(HaveOccurred())
			})

			It("prints a runtime spec to stdout", func() {
				var runtimeSpec specs.Spec
				Expect(json.Unmarshal(stdout.Bytes(), &runtimeSpec)).To(Succeed())
				Expect(runtimeSpec).To(Equal(toot.BundleRuntimeSpec))
			})

			It("calls driver.Bundle() with expected args", func() {
				var bundleArgs toot.BundleArgs
				readTestArgsFile(toot.BundleArgsFileName, &bundleArgs)
				Expect(bundleArgs).To(Equal(toot.BundleArgs{ID: handle, LayerIDs: []string{}}))
			})
		})

		Describe("failure", func() {
			Context("when driver.Bundle() returns an error", func() {
				BeforeEach(func() {
					tootCmd.Env = append(tootCmd.Env, "TOOT_BUNDLE_ERROR=true")
				})

				It("exits with 1", func() {
					Expect(exitErr.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(Equal("error: driver.Bundle: bundle-err\n"))
				})
			})
		})
	})
})
