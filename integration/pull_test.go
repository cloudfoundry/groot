package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("pull", func() {
	var (
		rootfsURI      string
		footCmd        *exec.Cmd
		driverStoreDir string
		configFilePath string
	)

	BeforeEach(func() {
		driverStoreDir = tempDir("", "groot-integration-tests")
		configFilePath = filepath.Join(driverStoreDir, "groot-config.yml")
		rootfsURI = filepath.Join(driverStoreDir, "rootfs.tar")

		writeFile(configFilePath, "")
		writeFile(rootfsURI, "a-rootfs")
		footCmd = newFootCommand(configFilePath, driverStoreDir, "pull", rootfsURI)
	})

	JustBeforeEach(func() {
		var out []byte
		out, footCmdError = footCmd.Output()
		footCmdOutput = gbytes.BufferWithBytes(out)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
	})

	Describe("Local images", func() {
		It("does not return an error", func() {
			Expect(footCmdError).NotTo(HaveOccurred())
		})

		It("calls driver.Unpack() with the expected args", func() {
			var args foot.UnpackCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &args)
			Expect(args[0].ID).NotTo(BeEmpty())
			Expect(args[0].ParentIDs).To(BeEmpty())
		})

		Describe("subsequent invocations", func() {
			It("generates the same layer ID", func() {
				var unpackArgs foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
				firstInvocationLayerID := unpackArgs[0].ID

				footCmd = newFootCommand(configFilePath, driverStoreDir, "pull", rootfsURI)
				Expect(footCmd.Run()).To(Succeed())

				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
				secondInvocationLayerID := unpackArgs[0].ID

				Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
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

		It("calls driver.Unpack() with the correct stream", func() {
			var args foot.UnpackCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &args)
			Expect(string(args[0].LayerTarContents)).To(Equal("a-rootfs"))
		})

		Describe("subsequent invocations", func() {
			Context("when the rootfs file timestamp has changed", func() {
				It("generates a different layer ID", func() {
					var unpackArgs foot.UnpackCalls
					unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
					firstInvocationLayerID := unpackArgs[0].ID

					now := time.Now()
					Expect(os.Chtimes(rootfsURI, now.Add(time.Hour), now.Add(time.Hour))).To(Succeed())

					footCmd = newFootCommand(configFilePath, driverStoreDir, "pull", rootfsURI)
					Expect(footCmd.Run()).To(Succeed())

					unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
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
		Context("when driver.Unpack() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_UNPACK_ERROR=true")
			})

			It("prints the error", func() {
				expectErrorOutput("unpack-err")
			})
		})

		Context("when the config file path is not an existing file", func() {
			BeforeEach(func() {
				Expect(os.Remove(configFilePath)).To(Succeed())
			})

			It("prints an error", func() {
				expectErrorOutput(notFoundRuntimeError[runtime.GOOS])
			})
		})

		Context("when the config file is invalid yaml", func() {
			BeforeEach(func() {
				writeFile(configFilePath, "%haha")
			})

			It("prints an error", func() {
				expectErrorOutput("yaml")
			})
		})

		Context("when the specified log level is invalid", func() {
			BeforeEach(func() {
				writeFile(configFilePath, "log_level: lol")
			})

			It("prints an error", func() {
				expectErrorOutput("lol")
			})
		})

		Context("when the rootfs URI is not a file", func() {
			BeforeEach(func() {
				Expect(os.Remove(rootfsURI)).To(Succeed())
			})

			It("prints an error", func() {
				expectErrorOutput(notFoundRuntimeError[runtime.GOOS])
			})
		})
	})
})
