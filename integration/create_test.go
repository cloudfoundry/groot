package integration_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
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
		tmpDir = tempDir("", "groot-integration-tests")
		configFilePath = filepath.Join(tmpDir, "groot-config.yml")
		rootfsURI = filepath.Join(tmpDir, "rootfs.tar")

		env = []string{}
		stdout = new(bytes.Buffer)
		stderr = new(bytes.Buffer)

		createArgs = []string{}
		expectedDiskLimit = 0
		writeFile(configFilePath, "")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	runCreateCmd := func() error {
		footArgv := append([]string{"--config", configFilePath, "--driver-store", tmpDir, "create"}, createArgs...)
		footArgv = append(footArgv, rootfsURI, handle)
		footCmd := exec.Command(footBinPath, footArgv...)
		footCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
		footCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
		footCmd.Env = append(os.Environ(), env...)
		return footCmd.Run()
	}

	Describe("success", func() {
		Describe("Local images", func() {
			var excludeImageFromQuota bool

			BeforeEach(func() {
				excludeImageFromQuota = false
				expectedDiskLimit = 500
			})

			JustBeforeEach(func() {
				imageContents := "a-rootfs"
				imageSize = int64(len(imageContents))

				if !excludeImageFromQuota {
					expectedDiskLimit -= imageSize
				}

				writeFile(rootfsURI, imageContents)
				Expect(runCreateCmd()).To(Succeed())
			})

			Context("when command succeeds", func() {
				JustBeforeEach(func() {
					expectedDiskLimit = 0
				})

				It("calls driver.Unpack() with the expected args", func() {
					var args foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
					Expect(args[0].ID).NotTo(BeEmpty())
					Expect(args[0].ParentIDs).To(BeEmpty())
				})

				Describe("subsequent invocations", func() {
					It("generates the same layer ID", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						Expect(runCreateCmd()).To(Succeed())

						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						secondInvocationLayerID := unpackArgs[0].ID

						Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
					})
				})

				Describe("layer caching", func() {
					It("calls exists", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
					})

					It("calls driver.Unpack() with the layerID", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

						Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

						lastCall := len(unpackArgs) - 1
						for i := range unpackArgs {
							Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
						}
					})

					Context("when the layer is cached", func() {
						BeforeEach(func() {
							env = append(env, "FOOT_LAYER_EXISTS=true")
						})

						It("doesn't call driver.Unpack()", func() {
							Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
						})
					})
				})

				It("calls driver.Bundle() with expected args", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)

					var bundleArgs foot.BundleCalls
					unmarshalFile(filepath.Join(tmpDir, foot.BundleArgsFileName), &bundleArgs)
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
					unmarshalFile(filepath.Join(tmpDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

					Expect(writeMetadataArgs[0].ID).To(Equal(handle))
					Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
				})

			})

			Context("--disk-limit-size-bytes is given", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "500"}
				})

				It("calls driver.Unpack() with the expected args", func() {
					var args foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
					Expect(args[0].ID).NotTo(BeEmpty())
					Expect(args[0].ParentIDs).To(BeEmpty())
				})

				Describe("subsequent invocations", func() {
					It("generates the same layer ID", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						Expect(runCreateCmd()).To(Succeed())

						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						secondInvocationLayerID := unpackArgs[0].ID

						Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
					})
				})

				Describe("layer caching", func() {
					It("calls exists", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
					})

					It("calls driver.Unpack() with the layerID", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

						Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

						lastCall := len(unpackArgs) - 1
						for i := range unpackArgs {
							Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
						}
					})

					Context("when the layer is cached", func() {
						BeforeEach(func() {
							env = append(env, "FOOT_LAYER_EXISTS=true")
						})

						It("doesn't call driver.Unpack()", func() {
							Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
						})
					})
				})

				It("calls driver.Bundle() with expected args", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)

					var bundleArgs foot.BundleCalls
					unmarshalFile(filepath.Join(tmpDir, foot.BundleArgsFileName), &bundleArgs)
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
					unmarshalFile(filepath.Join(tmpDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

					Expect(writeMetadataArgs[0].ID).To(Equal(handle))
					Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
				})

				Context("--exclude-image-from-quota is given as well", func() {
					BeforeEach(func() {
						excludeImageFromQuota = true
						createArgs = []string{"--disk-limit-size-bytes", "500", "--exclude-image-from-quota"}
					})

					It("calls driver.Unpack() with the expected args", func() {
						var args foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
						Expect(args[0].ID).NotTo(BeEmpty())
						Expect(args[0].ParentIDs).To(BeEmpty())
					})

					Describe("subsequent invocations", func() {
						It("generates the same layer ID", func() {
							var unpackArgs foot.UnpackCalls
							unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
							firstInvocationLayerID := unpackArgs[0].ID

							Expect(runCreateCmd()).To(Succeed())

							unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
							secondInvocationLayerID := unpackArgs[0].ID

							Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
						})
					})

					Describe("layer caching", func() {
						It("calls exists", func() {
							var existsArgs foot.ExistsCalls
							unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
							Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
						})

						It("calls driver.Unpack() with the layerID", func() {
							var existsArgs foot.ExistsCalls
							unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
							Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

							Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

							var unpackArgs foot.UnpackCalls
							unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
							Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

							lastCall := len(unpackArgs) - 1
							for i := range unpackArgs {
								Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
							}
						})

						Context("when the layer is cached", func() {
							BeforeEach(func() {
								env = append(env, "FOOT_LAYER_EXISTS=true")
							})

							It("doesn't call driver.Unpack()", func() {
								Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
							})
						})
					})

					It("calls driver.Bundle() with expected args", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)

						var bundleArgs foot.BundleCalls
						unmarshalFile(filepath.Join(tmpDir, foot.BundleArgsFileName), &bundleArgs)
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
						unmarshalFile(filepath.Join(tmpDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

						Expect(writeMetadataArgs[0].ID).To(Equal(handle))
						Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
					})

				})
			})

			It("calls driver.Unpack() with the correct stream", func() {
				var args foot.UnpackCalls
				unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
				Expect(string(args[0].LayerTarContents)).To(Equal("a-rootfs"))
			})

			Describe("subsequent invocations", func() {
				Context("when the rootfs file timestamp has changed", func() {
					It("generates a different layer ID", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						now := time.Now()
						Expect(os.Chtimes(rootfsURI, now.Add(time.Hour), now.Add(time.Hour))).To(Succeed())

						Expect(runCreateCmd()).To(Succeed())

						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
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
				It("calls driver.Unpack() with the expected args", func() {
					var args foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
					Expect(args[0].ID).NotTo(BeEmpty())
					Expect(args[0].ParentIDs).To(BeEmpty())
				})

				Describe("subsequent invocations", func() {
					It("generates the same layer ID", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						Expect(runCreateCmd()).To(Succeed())

						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						secondInvocationLayerID := unpackArgs[0].ID

						Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
					})
				})

				Describe("layer caching", func() {
					It("calls exists", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
					})

					It("calls driver.Unpack() with the layerID", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

						Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

						lastCall := len(unpackArgs) - 1
						for i := range unpackArgs {
							Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
						}
					})

					Context("when the layer is cached", func() {
						BeforeEach(func() {
							env = append(env, "FOOT_LAYER_EXISTS=true")
						})

						It("doesn't call driver.Unpack()", func() {
							Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
						})
					})
				})

				It("calls driver.Bundle() with expected args", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)

					var bundleArgs foot.BundleCalls
					unmarshalFile(filepath.Join(tmpDir, foot.BundleArgsFileName), &bundleArgs)
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
					unmarshalFile(filepath.Join(tmpDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

					Expect(writeMetadataArgs[0].ID).To(Equal(handle))
					Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
				})

			})

			Context("--disk-limit-size-bytes is given", func() {
				BeforeEach(func() {
					createArgs = []string{"--disk-limit-size-bytes", "500"}
				})
				JustBeforeEach(func() {
					expectedDiskLimit = 500 - imageSize
				})

				It("calls driver.Unpack() with the expected args", func() {
					var args foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
					Expect(args[0].ID).NotTo(BeEmpty())
					Expect(args[0].ParentIDs).To(BeEmpty())
				})

				Describe("subsequent invocations", func() {
					It("generates the same layer ID", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						firstInvocationLayerID := unpackArgs[0].ID

						Expect(runCreateCmd()).To(Succeed())

						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						secondInvocationLayerID := unpackArgs[0].ID

						Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
					})
				})

				Describe("layer caching", func() {
					It("calls exists", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
					})

					It("calls driver.Unpack() with the layerID", func() {
						var existsArgs foot.ExistsCalls
						unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
						Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

						Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
						Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

						lastCall := len(unpackArgs) - 1
						for i := range unpackArgs {
							Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
						}
					})

					Context("when the layer is cached", func() {
						BeforeEach(func() {
							env = append(env, "FOOT_LAYER_EXISTS=true")
						})

						It("doesn't call driver.Unpack()", func() {
							Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
						})
					})
				})

				It("calls driver.Bundle() with expected args", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)

					var bundleArgs foot.BundleCalls
					unmarshalFile(filepath.Join(tmpDir, foot.BundleArgsFileName), &bundleArgs)
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
					unmarshalFile(filepath.Join(tmpDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

					Expect(writeMetadataArgs[0].ID).To(Equal(handle))
					Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
				})

				Context("--exclude-image-from-quota is given as well", func() {
					BeforeEach(func() {
						createArgs = []string{"--disk-limit-size-bytes", "500", "--exclude-image-from-quota"}
					})
					JustBeforeEach(func() {
						expectedDiskLimit = 500
					})

					It("calls driver.Unpack() with the expected args", func() {
						var args foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)
						Expect(args[0].ID).NotTo(BeEmpty())
						Expect(args[0].ParentIDs).To(BeEmpty())
					})

					Describe("subsequent invocations", func() {
						It("generates the same layer ID", func() {
							var unpackArgs foot.UnpackCalls
							unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
							firstInvocationLayerID := unpackArgs[0].ID

							Expect(runCreateCmd()).To(Succeed())

							unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
							secondInvocationLayerID := unpackArgs[0].ID

							Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
						})
					})

					Describe("layer caching", func() {
						It("calls exists", func() {
							var existsArgs foot.ExistsCalls
							unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
							Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
						})

						It("calls driver.Unpack() with the layerID", func() {
							var existsArgs foot.ExistsCalls
							unmarshalFile(filepath.Join(tmpDir, foot.ExistsArgsFileName), &existsArgs)
							Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

							Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).To(BeAnExistingFile())

							var unpackArgs foot.UnpackCalls
							unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
							Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

							lastCall := len(unpackArgs) - 1
							for i := range unpackArgs {
								Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
							}
						})

						Context("when the layer is cached", func() {
							BeforeEach(func() {
								env = append(env, "FOOT_LAYER_EXISTS=true")
							})

							It("doesn't call driver.Unpack()", func() {
								Expect(filepath.Join(tmpDir, foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
							})
						})
					})

					It("calls driver.Bundle() with expected args", func() {
						var unpackArgs foot.UnpackCalls
						unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)

						var bundleArgs foot.BundleCalls
						unmarshalFile(filepath.Join(tmpDir, foot.BundleArgsFileName), &bundleArgs)
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
						unmarshalFile(filepath.Join(tmpDir, foot.WriteMetadataArgsFileName), &writeMetadataArgs)

						Expect(writeMetadataArgs[0].ID).To(Equal(handle))
						Expect(writeMetadataArgs[0].VolumeData).To(Equal(groot.VolumeMetadata{BaseImageSize: imageSize}))
					})

				})
			})

			Context("when the image has multiple layers", func() {
				It("correctly passes parent IDs to each driver.Unpack() call", func() {
					var args foot.UnpackCalls
					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &args)

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

				exitErr := runCreateCmd()
				Expect(exitErr).To(HaveOccurred())
				Expect(exitErr.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
			})

			Context("when driver.Unpack() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "FOOT_UNPACK_ERROR=true")
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(ContainSubstring("unpack-err\n"))
				})
			})

			Context("when the config file path is not an existing file", func() {
				BeforeEach(func() {
					Expect(os.Remove(configFilePath)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
				})
			})

			Context("when the config file is invalid yaml", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "%haha")
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("yaml"))
				})
			})

			Context("when the specified log level is invalid", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "log_level: lol")
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("lol"))
				})
			})

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

			JustBeforeEach(func() {
				exitErr := runCreateCmd()
				Expect(exitErr).To(HaveOccurred())
				Expect(exitErr.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
			})

			Context("when driver.Unpack() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "FOOT_UNPACK_ERROR=true")
				})

				It("prints the error", func() {
					Expect(stdout.String()).To(ContainSubstring("unpack-err\n"))
				})
			})

			Context("when the config file path is not an existing file", func() {
				BeforeEach(func() {
					Expect(os.Remove(configFilePath)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
				})
			})

			Context("when the config file is invalid yaml", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "%haha")
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("yaml"))
				})
			})

			Context("when the specified log level is invalid", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "log_level: lol")
				})

				It("prints an error", func() {
					Expect(stdout.String()).To(ContainSubstring("lol"))
				})
			})

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
		})
	})
})
