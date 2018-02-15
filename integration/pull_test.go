package integration_test

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("pull", func() {
	var (
		rootfsURI   string
		footSession *gexec.Session
	)

	BeforeEach(func() {
		tmpDir = tempDir("", "groot-integration-tests")
		configFilePath = filepath.Join(tmpDir, "groot-config.yml")
		rootfsURI = filepath.Join(tmpDir, "rootfs.tar")

		env = []string{}

		writeFile(configFilePath, "")
	})

	JustBeforeEach(func() {
		footSession = runFoot(configFilePath, tmpDir, "pull", rootfsURI)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Describe("Local images", func() {
		BeforeEach(func() {
			writeFile(rootfsURI, "a-rootfs")
		})

		It("does not return an error", func() {
			Expect(footSession).To(gexec.Exit(0))
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

				Expect(runFoot(configFilePath, tmpDir, "pull", rootfsURI)).To(gexec.Exit(0))

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

					Expect(runFoot(configFilePath, tmpDir, "pull", rootfsURI)).To(gexec.Exit(0))

					unmarshalFile(filepath.Join(tmpDir, foot.UnpackArgsFileName), &unpackArgs)
					secondInvocationLayerID := unpackArgs[1].ID

					Expect(secondInvocationLayerID).NotTo(Equal(firstInvocationLayerID))
				})
			})
		})
	})

	Describe("Remote images", func() {
		BeforeEach(func() {
			rootfsURI = "docker:///cfgarden/three-layers"
		})

		It("does not return an error", func() {
			Expect(footSession).To(gexec.Exit(0))
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

				Expect(runFoot(configFilePath, tmpDir, "pull", rootfsURI)).To(gexec.Exit(0))

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

	Describe("failure", func() {
		Describe("Local Images", func() {
			BeforeEach(func() {
				writeFile(rootfsURI, "a-rootfs")
			})

			Context("when driver.Unpack() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "FOOT_UNPACK_ERROR=true")
				})

				It("prints the error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say("unpack-err"))
				})
			})

			Context("when the config file path is not an existing file", func() {
				BeforeEach(func() {
					Expect(os.Remove(configFilePath)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say(notFoundRuntimeError[runtime.GOOS]))
				})
			})

			Context("when the config file is invalid yaml", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "%haha")
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say("yaml"))
				})
			})

			Context("when the specified log level is invalid", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "log_level: lol")
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say("lol"))
				})
			})

			Context("when the rootfs URI is not a file", func() {
				BeforeEach(func() {
					Expect(os.Remove(rootfsURI)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say(notFoundRuntimeError[runtime.GOOS]))
				})
			})
		})

		Describe("Remote Images", func() {
			BeforeEach(func() {
				rootfsURI = "docker:///cfgarden/three-layers"
			})

			Context("when driver.Unpack() returns an error", func() {
				BeforeEach(func() {
					env = append(env, "FOOT_UNPACK_ERROR=true")
				})

				It("prints the error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say("unpack-err"))
				})
			})

			Context("when the config file path is not an existing file", func() {
				BeforeEach(func() {
					Expect(os.Remove(configFilePath)).To(Succeed())
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say(notFoundRuntimeError[runtime.GOOS]))
				})
			})

			Context("when the config file is invalid yaml", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "%haha")
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say("yaml"))
				})
			})

			Context("when the specified log level is invalid", func() {
				BeforeEach(func() {
					writeFile(configFilePath, "log_level: lol")
				})

				It("prints an error", func() {
					Expect(footSession).To(gexec.Exit(1))
					Expect(footSession.Out).To(gbytes.Say("lol"))
				})
			})
		})
	})
})
