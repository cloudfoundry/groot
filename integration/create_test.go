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

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("create", func() {
	var (
		rootfsURI         string
		handle            = "some-handle"
		expectedDiskLimit int64
		imageSize         int64
		createArgs        []string
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

		createArgs = []string{}
		expectedDiskLimit = 0
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	runCreateCmd := func() error {
		footArgv := append([]string{"--config", configFilePath, "--driver-store", tempDir, "create"}, createArgs...)
		footArgv = append(footArgv, rootfsURI, handle)
		footCmd := exec.Command(footBinPath, footArgv...)
		footCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
		footCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
		footCmd.Env = append(os.Environ(), env...)
		return footCmd.Run()
	}

	bundleAndWriteMetadataSuccessful := func(handle string) {
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
			Expect(bundleArgs[0].DiskLimit).To(Equal(expectedDiskLimit))
		})

		It("calls driver.WriteMetadata() with expected args", func() {
			var writeMetadataArgs foot.WriteMetadataCalls
			readTestArgsFile(foot.WriteMetadataArgsFileName, &writeMetadataArgs)

			Expect(writeMetadataArgs[0].ID).To(Equal(handle))
			Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
		})
	}

	bundleAndWriteMetadataUnsuccessful := func() {
		Context("when driver.Bundle() returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_BUNDLE_ERROR=true")
			})

			It("prints the error", func() {
				Expect(stdout.String()).To(ContainSubstring("bundle-err\n"))
			})
		})

		Context("when driver.WriteMetadata() returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_WRITE_METADATA_ERROR=true")
			})

			It("prints the error", func() {
				Expect(stdout.String()).To(ContainSubstring("write-metadata-err\n"))
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
				imageContents := "a-rootfs"
				imageSize = int64(len(imageContents))
				writeFile(rootfsURI, imageContents)
				Expect(runCreateCmd()).To(Succeed())
			})

			Context("when command succeeds", func() {
				unpackIsSuccessful(runCreateCmd)
				bundleAndWriteMetadataSuccessful(handle)
			})

			Context("--disk-limit-size-bytes is given", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "500"}
					expectedDiskLimit = 500 - imageSize
				})

				unpackIsSuccessful(runCreateCmd)
				bundleAndWriteMetadataSuccessful(handle)

				Context("--exclude-image-from-quota is given as well", func() {
					BeforeEach(func() {
						createArgs = []string{"--disk-limit-size-bytes", "500", "--exclude-image-from-quota"}
						expectedDiskLimit = 500
					})

					unpackIsSuccessful(runCreateCmd)
					bundleAndWriteMetadataSuccessful(handle)
				})
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
				imageSize = 297
				rootfsURI = "docker:///cfgarden/three-layers"

				Expect(runCreateCmd()).To(Succeed())
			})

			Context("when command succeeds", func() {
				unpackIsSuccessful(runCreateCmd)
				bundleAndWriteMetadataSuccessful(handle)
			})

			Context("--disk-limit-size-bytes is given", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "500"}
					expectedDiskLimit = 500 - imageSize
				})

				unpackIsSuccessful(runCreateCmd)
				bundleAndWriteMetadataSuccessful(handle)

				Context("--exclude-image-from-quota is given as well", func() {
					BeforeEach(func() {
						createArgs = []string{"--disk-limit-size-bytes", "500", "--exclude-image-from-quota"}
						expectedDiskLimit = 500
					})

					unpackIsSuccessful(runCreateCmd)
					bundleAndWriteMetadataSuccessful(handle)
				})
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
			bundleAndWriteMetadataUnsuccessful()

			Context("when the rootfs URI is not a file", func() {
				BeforeEach(func() {
					createRootfsTar = false
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
				})
			})

			Context("--disk-limit-size-bytes is negative", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "-500", "--exclude-image-from-quota"}
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("invalid disk limit: -500"))
				})
			})

			Context("--disk-limit-size-bytes is less than the image size", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "5"}
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("pulling image: layers exceed disk quota 8/5 bytes"))
				})
			})

			Context("--disk-limit-size-bytes is exactly the image size", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "8"}
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("disk limit 8 must be larger than image size 8"))
				})
			})
		})

		Describe("Remote Images", func() {
			BeforeEach(func() {
				rootfsURI = "docker:///cfgarden/three-layers"
			})

			whenUnpackIsUnsuccessful(runCreateCmd)
			bundleAndWriteMetadataUnsuccessful()
		})
	})
})
