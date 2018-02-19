package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("create", func() {
	var (
		rootfsURI string
		imageSize int64
		session   *gexec.Session
		footCmd   *exec.Cmd
	)

	BeforeEach(func() {
		driverStoreDir = tempDir("", "groot-integration-tests")
		configFilePath = filepath.Join(driverStoreDir, "groot-config.yml")
		rootfsURI = filepath.Join(driverStoreDir, "rootfs.tar")

		writeFile(configFilePath, "")

		imageContents := "a-rootfs"
		imageSize = int64(len(imageContents))
		writeFile(rootfsURI, imageContents)

		footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle")
	})

	JustBeforeEach(func() {
		session = gexecStart(footCmd).Wait()
	})

	AfterEach(func() {
		Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
	})

	Describe("Local images", func() {
		It("does not return an error", func() {
			Expect(session).To(gexec.Exit(0))
		})

		It("calls driver.Unpack() with the expected args", func() {
			var args foot.UnpackCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &args)
			Expect(args[0].ID).NotTo(BeEmpty())
			Expect(args[0].ParentIDs).To(BeEmpty())
		})

		It("calls driver.Unpack() with the correct stream", func() {
			var args foot.UnpackCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &args)
			Expect(string(args[0].LayerTarContents)).To(Equal("a-rootfs"))
		})

		Describe("subsequent invocations", func() {
			It("generates the same layer ID", func() {
				var unpackArgs foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
				firstInvocationLayerID := unpackArgs[0].ID

				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle")
				Expect(gexecStart(footCmd).Wait()).To(gexec.Exit(0))

				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
				secondInvocationLayerID := unpackArgs[1].ID

				Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
			})

			Context("when the rootfs file timestamp has changed", func() {
				It("generates a different layer ID", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
					firstInvocationLayerID := unpackArgs[0].ID

					now := time.Now()
					Expect(os.Chtimes(rootfsURI, now.Add(time.Hour), now.Add(time.Hour))).To(Succeed())

					footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle")
					Expect(gexecStart(footCmd).Wait()).To(gexec.Exit(0))

					unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
					secondInvocationLayerID := unpackArgs[1].ID

					Expect(secondInvocationLayerID).NotTo(Equal(firstInvocationLayerID))
				})
			})
		})

		Describe("layer caching", func() {
			It("calls exists", func() {
				var existsArgs foot.ExistsCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.ExistsArgsFileName), &existsArgs)
				Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
			})

			It("calls driver.Unpack() with the layerID", func() {
				var existsArgs foot.ExistsCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.ExistsArgsFileName), &existsArgs)
				Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

				Expect(filepath.Join(driverStoreDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

				var unpackArgs foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
				Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

				lastCall := len(unpackArgs) - 1
				for i := range unpackArgs {
					Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
				}
			})

			Context("when the layer is cached", func() {
				BeforeEach(func() {
					footCmd.Env = append(os.Environ(), "FOOT_LAYER_EXISTS=true")
				})

				It("doesn't call driver.Unpack()", func() {
					Expect(filepath.Join(driverStoreDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
				})
			})
		})

		It("calls driver.Bundle() with expected args", func() {
			var unpackArgs foot.UnpackCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)

			var bundleArgs foot.BundleCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.BundleArgsFileName), &bundleArgs)
			unpackLayerIds := []string{}
			for _, call := range unpackArgs {
				unpackLayerIds = append(unpackLayerIds, call.ID)
			}
			Expect(bundleArgs[0].ID).To(Equal("some-handle"))
			Expect(bundleArgs[0].LayerIDs).To(ConsistOf(unpackLayerIds))
			Expect(bundleArgs[0].DiskLimit).To(Equal(int64(0)))
		})

		It("calls driver.WriteMetadata() with expected args", func() {
			var writeMetadataArgs foot.WriteMetadataCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

			Expect(writeMetadataArgs[0].ID).To(Equal("some-handle"))
			Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
		})

		Context("--disk-limit-size-bytes is given", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "500")
			})

			It("calls driver.Bundle() with expected args", func() {
				var unpackArgs foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)

				var bundleArgs foot.BundleCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.BundleArgsFileName), &bundleArgs)
				unpackLayerIds := []string{}
				for _, call := range unpackArgs {
					unpackLayerIds = append(unpackLayerIds, call.ID)
				}
				Expect(bundleArgs[0].ID).To(Equal("some-handle"))
				Expect(bundleArgs[0].LayerIDs).To(ConsistOf(unpackLayerIds))
				Expect(bundleArgs[0].DiskLimit).To(Equal(500 - imageSize))
			})

			Context("--exclude-image-from-quota is given as well", func() {
				BeforeEach(func() {
					footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "500", "--exclude-image-from-quota")
				})

				It("calls driver.Bundle() with expected args", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)

					var bundleArgs foot.BundleCalls
					unmarshalFile(filepath.Join(driverStoreDir, foot.BundleArgsFileName), &bundleArgs)
					unpackLayerIds := []string{}
					for _, call := range unpackArgs {
						unpackLayerIds = append(unpackLayerIds, call.ID)
					}
					Expect(bundleArgs[0].ID).To(Equal("some-handle"))
					Expect(bundleArgs[0].LayerIDs).To(ConsistOf(unpackLayerIds))
					Expect(bundleArgs[0].DiskLimit).To(Equal(int64(500)))
				})
			})
		})
	})

	Describe("Remote images", func() {
		Context("when the image has multiple layers", func() {
			It("correctly passes parent IDs to each driver.Unpack() call", func() {
				var args foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &args)

				chainIDs := []string{}
				for _, a := range args {
					Expect(a.ParentIDs).To(Equal(chainIDs))
					chainIDs = append(chainIDs, a.ID)
				}
			})
		})
	})

	Describe("failure", func() {
		Context("--disk-limit-size-bytes is negative", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "-500", "--exclude-image-from-quota")
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("invalid disk limit: -500"))
			})
		})

		Context("--disk-limit-size-bytes is less than the image size", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "5")
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("pulling image: layers exceed disk quota 8/5 bytes"))
			})
		})

		Context("--disk-limit-size-bytes is exactly the image size", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "8")
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("disk limit 8 must be larger than image size 8"))
			})
		})

		Context("when driver.Unpack() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_UNPACK_ERROR=true")
			})

			It("prints the error", func() {
				Expect(session.Out).To(gbytes.Say("unpack-err"))
			})
		})

		Context("when the config file path is not an existing file", func() {
			BeforeEach(func() {
				Expect(os.Remove(configFilePath)).To(Succeed())
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say(notFoundRuntimeError[runtime.GOOS]))
			})
		})

		Context("when the config file is invalid yaml", func() {
			BeforeEach(func() {
				writeFile(configFilePath, "%haha")
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("yaml"))
			})
		})

		Context("when the specified log level is invalid", func() {
			BeforeEach(func() {
				writeFile(configFilePath, "log_level: lol")
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say("lol"))
			})
		})

		Context("when driver.Bundle() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_BUNDLE_ERROR=true")
			})

			It("prints the error", func() {
				Expect(session.Out).To(gbytes.Say("bundle-err"))
			})
		})

		Context("when driver.WriteMetadata() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_WRITE_METADATA_ERROR=true")
			})

			It("prints the error", func() {
				Expect(session.Out).To(gbytes.Say("write-metadata-err"))
			})
		})

		Context("when the rootfs URI is not a file", func() {
			BeforeEach(func() {
				Expect(os.Remove(rootfsURI)).To(Succeed())
			})

			It("prints an error", func() {
				Expect(session.Out).To(gbytes.Say(notFoundRuntimeError[runtime.GOOS]))
			})
		})
	})
})
