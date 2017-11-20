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
	"runtime"
	"syscall"
	"time"

	"code.cloudfoundry.org/groot/integration/cmd/toot/toot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("groot", func() {
	Describe("create", func() {
		var (
			rootfsURI            string
			handle               = "some-handle"
			logLevel             string
			configFilePath       string
			env                  []string
			tempDir              string
			stdout               *bytes.Buffer
			stderr               *bytes.Buffer
			notFoundRuntimeError = map[string]string{
				"linux":   "no such file or directory",
				"windows": "The system cannot find the file specified.",
			}
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
			rootfsURI = filepath.Join(tempDir, "rootfs.tar")

			logLevel = ""
			env = []string{"TOOT_BASE_DIR=" + tempDir}
			stdout = new(bytes.Buffer)
			stderr = new(bytes.Buffer)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		runTootCmd := func() error {
			tootArgv := []string{"--config", configFilePath, "create", rootfsURI, handle}
			tootCmd := exec.Command(tootBinPath, tootArgv...)
			tootCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
			tootCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
			tootCmd.Env = append(os.Environ(), env...)
			return tootCmd.Run()
		}

		Describe("success", func() {
			JustBeforeEach(func() {
				configYml := fmt.Sprintf(`log_level: %s`, logLevel)
				Expect(ioutil.WriteFile(configFilePath, []byte(configYml), 0600)).To(Succeed())

				Expect(ioutil.WriteFile(rootfsURI, []byte("a-rootfs"), 0600)).To(Succeed())

				Expect(runTootCmd()).To(Succeed())
			})

			It("prints a runtime spec to stdout", func() {
				var runtimeSpec specs.Spec
				Expect(json.Unmarshal(stdout.Bytes(), &runtimeSpec)).To(Succeed())
				Expect(runtimeSpec).To(Equal(toot.BundleRuntimeSpec))
			})

			It("calls driver.Unpack() with the expected args", func() {
				var args toot.UnpackArgs
				readTestArgsFile(toot.UnpackArgsFileName, &args)
				Expect(args.ID).NotTo(BeEmpty())
				Expect(args.ParentID).To(BeEmpty())
				Expect(string(args.LayerTarContents)).To(Equal("a-rootfs"))
			})

			It("calls driver.Bundle() with expected args", func() {
				var unpackArgs toot.UnpackArgs
				readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)

				var bundleArgs toot.BundleArgs
				readTestArgsFile(toot.BundleArgsFileName, &bundleArgs)
				Expect(bundleArgs.ID).To(Equal(handle))
				Expect(bundleArgs.LayerIDs).To(ConsistOf(unpackArgs.ID))
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

			Describe("subsequent invocations", func() {
				Context("when the rootfs file has not changed", func() {
					It("generates the same layer ID", func() {
						var unpackArgs toot.UnpackArgs
						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						firstInvocationLayerID := unpackArgs.ID

						Expect(runTootCmd()).To(Succeed())

						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						secondInvocationLayerID := unpackArgs.ID

						Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
					})
				})

				Context("when the rootfs file timestamp has changed", func() {
					It("generates a different layer ID", func() {
						var unpackArgs toot.UnpackArgs
						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						firstInvocationLayerID := unpackArgs.ID

						rootfsFileModTime := func() int64 {
							rootfsFileInfo, err := os.Stat(rootfsURI)
							Expect(err).NotTo(HaveOccurred())
							return rootfsFileInfo.ModTime().UnixNano()
						}
						initialRootfsFileMtime := rootfsFileModTime()

						Eventually(func() int64 {
							now := time.Now()
							Expect(os.Chtimes(rootfsURI, now, now)).To(Succeed())
							return rootfsFileModTime()
						}, time.Second*20, time.Millisecond*50).ShouldNot(Equal(initialRootfsFileMtime))

						Expect(runTootCmd()).To(Succeed())

						readTestArgsFile(toot.UnpackArgsFileName, &unpackArgs)
						secondInvocationLayerID := unpackArgs.ID

						Expect(secondInvocationLayerID).NotTo(Equal(firstInvocationLayerID))
					})
				})
			})
		})

		Describe("failure", func() {
			var (
				writeConfigFile bool
				createRootfsTar bool
			)

			BeforeEach(func() {
				writeConfigFile = true
				createRootfsTar = true
			})

			JustBeforeEach(func() {
				if writeConfigFile {
					configYml := fmt.Sprintf(`log_level: %s`, logLevel)
					Expect(ioutil.WriteFile(configFilePath, []byte(configYml), 0600)).To(Succeed())
				}

				if createRootfsTar {
					Expect(ioutil.WriteFile(rootfsURI, []byte("a-rootfs"), 0600)).To(Succeed())
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

			Context("when driver.Unpack() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "TOOT_UNPACK_ERROR=true")
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(Equal("unpack-err\n"))
				})
			})

			Context("when the rootfs URI is not a file", func() {
				BeforeEach(func() {
					createRootfsTar = false
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
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
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
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
