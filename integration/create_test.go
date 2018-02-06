package integration_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("create", func() {
	var (
		rootfsURI string
		handle    = "some-handle"
	)

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "groot-integration-tests")
		Expect(err).NotTo(HaveOccurred())
		configFilePath = filepath.Join(tempDir, "groot-config.yml")
		rootfsURI = filepath.Join(tempDir, "rootfs.tar")

		logLevel = ""
		env = []string{}
		stdout = new(bytes.Buffer)
		stderr = new(bytes.Buffer)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	runCreateCmd := func() error {
		footArgv := []string{"--config", configFilePath, "--driver-store", tempDir, "create", rootfsURI, handle}
		footCmd := exec.Command(footBinPath, footArgv...)
		footCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
		footCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
		footCmd.Env = append(os.Environ(), env...)
		return footCmd.Run()
	}

	bundleIsSuccessful := func(handle string) {
		It("calls driver.Bundle() with expected args", func() {
			var unpackArgs foot.UnpackCalls
			readTestArgsFile(foot.UnpackArgsFileName, &unpackArgs)

			var bundleArgs foot.BundleCalls
			readTestArgsFile(foot.BundleArgsFileName, &bundleArgs)
			unpackLayerIds := []string{}
			for _, call := range unpackArgs {
				unpackLayerIds = append(unpackLayerIds, call.ID)
			}
			Expect(bundleArgs[0].ID).To(Equal(handle))
			Expect(bundleArgs[0].LayerIDs).To(ConsistOf(unpackLayerIds))
		})
	}

	whenBundleIsUnsuccessful := func() {
		Context("when driver.Bundle() returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_BUNDLE_ERROR=true")
			})

			It("prints the error", func() {
				Expect(stdout.String()).To(ContainSubstring("bundle-err\n"))
			})
		})
	}

	Describe("success", func() {
		JustBeforeEach(func() {
			if configFilePath != "" {
				writeFile(configFilePath, "log_level: "+logLevel)
			}
		})

		Describe("Local images", func() {
			JustBeforeEach(func() {
				writeFile(rootfsURI, "a-rootfs")
				Expect(runCreateCmd()).To(Succeed())
			})

			Context("when command succeeds", func() {
				unpackIsSuccessful(runCreateCmd)
				bundleIsSuccessful(handle)
			})

			It("calls driver.Unpack() with the correct stream", func() {
				var args foot.UnpackCalls
				readTestArgsFile(foot.UnpackArgsFileName, &args)
				Expect(string(args[0].LayerTarContents)).To(Equal("a-rootfs"))
			})

			Describe("subsequent invocations", func() {
				Context("when the rootfs file timestamp has changed", func() {
					It("generates a different layer ID", func() {
						var unpackArgs foot.UnpackCalls
						readTestArgsFile(foot.UnpackArgsFileName, &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						now := time.Now()
						Expect(os.Chtimes(rootfsURI, now.Add(time.Hour), now.Add(time.Hour))).To(Succeed())

						Expect(runCreateCmd()).To(Succeed())

						readTestArgsFile(foot.UnpackArgsFileName, &unpackArgs)
						secondInvocationLayerID := unpackArgs[1].ID

						Expect(secondInvocationLayerID).NotTo(Equal(firstInvocationLayerID))
					})
				})
			})
		})

		Describe("Remote images", func() {
			JustBeforeEach(func() {
				rootfsURI = "docker:///cfgarden/three-layers"

				Expect(runCreateCmd()).To(Succeed())
			})

			Context("when command succeeds", func() {
				unpackIsSuccessful(runCreateCmd)
				bundleIsSuccessful(handle)
			})

			Context("when the image has multiple layers", func() {
				It("correctly passes parent IDs to each driver.Unpack() call", func() {
					var args foot.UnpackCalls
					readTestArgsFile(foot.UnpackArgsFileName, &args)

					chainIDs := []string{}
					for _, a := range args {
						Expect(a.ParentIDs).To(Equal(chainIDs))
						chainIDs = append(chainIDs, a.ID)
					}
				})
			})
		})
	})

	Describe("failure", func() {
		Describe("Local Images", func() {
			var createRootfsTar bool

			BeforeEach(func() {
				createRootfsTar = true
			})

			JustBeforeEach(func() {
				if createRootfsTar {
					writeFile(rootfsURI, "a-rootfs")
				}
			})

			whenUnpackIsUnsuccessful(runCreateCmd)
			whenBundleIsUnsuccessful()

			Context("when the rootfs URI is not a file", func() {
				BeforeEach(func() {
					createRootfsTar = false
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
				})
			})
		})

		Describe("Remote Images", func() {
			BeforeEach(func() {
				rootfsURI = "docker:///cfgarden/three-layers"
			})

			whenUnpackIsUnsuccessful(runCreateCmd)
			whenBundleIsUnsuccessful()
		})
	})
})
