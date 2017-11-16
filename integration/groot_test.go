package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
			rootfsURI      = "some-rootfs-uri"
			handle         = "some-handle"
			logLevel       string
			configFilePath string
			env            []string
			tempDir        string
			stdout         *bytes.Buffer
			stderr         *bytes.Buffer
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
			configFilePath = filepath.Join(tempDir, "groot-config.yml")

			logLevel = ""
			env = []string{"TOOT_BASE_DIR=" + tempDir}
			stdout = new(bytes.Buffer)
			stderr = new(bytes.Buffer)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		Describe("success", func() {
			JustBeforeEach(func() {
				configYml := fmt.Sprintf(`log_level: %s`, logLevel)
				Expect(ioutil.WriteFile(configFilePath, []byte(configYml), 0600)).To(Succeed())

				tootArgv := []string{"--config", configFilePath, "create", rootfsURI, handle}
				tootCmd := exec.Command(tootBinPath, tootArgv...)
				tootCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
				tootCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
				tootCmd.Env = append(os.Environ(), env...)
				Expect(tootCmd.Run()).To(Succeed())
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

			It("logs to stderr with an appropriate lager session, defaulting to info level", func() {
				Expect(stderr.String()).To(ContainSubstring("groot.create.bundle-info"))
				Expect(stderr.String()).NotTo(ContainSubstring("bundle-debug"))
			})

			Context("when the log level is specified", func() {
				BeforeEach(func() {
					logLevel = "debug"
				})

				It("logs to stderr with the specified lager level", func() {
					Expect(stderr.String()).To(ContainSubstring("bundle-debug"))
				})
			})
		})

		Describe("failure", func() {
			var (
				writeConfigFile bool
			)

			BeforeEach(func() {
				writeConfigFile = true
			})

			JustBeforeEach(func() {
				if writeConfigFile {
					configYml := fmt.Sprintf(`log_level: %s`, logLevel)
					Expect(ioutil.WriteFile(configFilePath, []byte(configYml), 0600)).To(Succeed())
				}

				tootArgv := []string{"--config", configFilePath, "create", rootfsURI, handle}
				tootCmd := exec.Command(tootBinPath, tootArgv...)
				tootCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
				tootCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
				tootCmd.Env = append(os.Environ(), env...)
				exitErr := tootCmd.Run()
				Expect(exitErr.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
			})

			Context("when driver.Bundle() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "TOOT_BUNDLE_ERROR=true")
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(Equal("bundle-err\n"))
				})
			})

			Context("when no config file is provided", func() {
				BeforeEach(func() {
					configFilePath = ""
					writeConfigFile = false
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("please provide --config"))
				})
			})

			Context("when the config file path is not an existing file", func() {
				BeforeEach(func() {
					writeConfigFile = false
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("no such file or directory"))
				})
			})

			Context("when the config file is invalid yaml", func() {
				BeforeEach(func() {
					writeConfigFile = false
					Expect(ioutil.WriteFile(configFilePath, []byte("%haha"), 0600)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("yaml"))
				})
			})

			Context("when the specified log level is invalid", func() {
				BeforeEach(func() {
					logLevel = "lol"
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("lol"))
				})
			})
		})
	})
})
