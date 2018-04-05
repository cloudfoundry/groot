package integration_test

import (
	"compress/gzip"
	"fmt"
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
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("create", func() {
	var (
		rootfsURI      string
		imageSize      int64
		footCmd        *exec.Cmd
		driverStoreDir string
		configFilePath string
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
		var out []byte
		out, footCmdError = footCmd.CombinedOutput()
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
				Expect(footCmd.Run()).To(Succeed())

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
					Expect(footCmd.Run()).To(Succeed())

					unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
					secondInvocationLayerID := unpackArgs[1].ID

					Expect(secondInvocationLayerID).NotTo(Equal(firstInvocationLayerID))
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
		var workDir string

		BeforeEach(func() {
			var err error
			workDir, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			rootfsURI = fmt.Sprintf("oci:///%s/oci-test-images/opq-whiteouts-busybox:latest", workDir)
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

		Context("when --disk-limit-size-bytes is less than compressed image size and exclude-image-from-quota is set", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "1", "--exclude-image-from-quota")
			})

			It("succeeds", func() {
				var args foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &args)
				Expect(len(args)).NotTo(Equal(0))
			})
		})

		Context("when --disk-limit-size-bytes is sufficiently large", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "99999999")
			})

			It("succeeds", func() {
				var unpackArgs foot.UnpackCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.UnpackArgsFileName), &unpackArgs)
				Expect(len(unpackArgs)).NotTo(Equal(0))
			})

			It("calculates the disk limit based on uncompressed layer sizes", func() {
				var bundleArgs foot.BundleCalls
				unmarshalFile(filepath.Join(driverStoreDir, foot.BundleArgsFileName), &bundleArgs)

				blobsPath := fmt.Sprintf("%s/oci-test-images/opq-whiteouts-busybox/blobs/sha256", workDir)

				// yuck, this is white-box because we know this is how foot calculates size
				firstBlobSize := getUncompressedBlobSize(filepath.Join(blobsPath, "56bec22e355981d8ba0878c6c2f23b21f422f30ab0aba188b54f1ffeff59c190"))
				secondBlobSize := getUncompressedBlobSize(filepath.Join(blobsPath, "ed2d7b0f6d7786230b71fd60de08a553680a9a96ab216183bcc49c71f06033ab"))

				Expect(bundleArgs[0].DiskLimit).To(BeEquivalentTo(99999999 - (firstBlobSize + secondBlobSize)))
			})
		})
	})

	Describe("Local images failure", func() {
		Context("--disk-limit-size-bytes is negative", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "-500", "--exclude-image-from-quota")
			})

			It("prints an error", func() {
				expectErrorOutput("invalid disk limit: -500")
			})
		})

		Context("--disk-limit-size-bytes is less than the image size", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "5")
			})

			It("prints an error", func() {
				expectErrorOutput("pulling image: layers exceed disk quota 8/5 bytes")
			})
		})

		Context("--disk-limit-size-bytes is exactly the image size", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "8")
			})

			It("prints an error", func() {
				expectErrorOutput("disk limit 8 must be larger than image size 8")
			})
		})

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

		Context("when driver.Bundle() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_BUNDLE_ERROR=true")
			})

			It("prints the error", func() {
				expectErrorOutput("bundle-err")
			})
		})

		Context("when driver.WriteMetadata() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_WRITE_METADATA_ERROR=true")
			})

			It("prints the error", func() {
				expectErrorOutput("write-metadata-err")
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

	Describe("Remote images failure", func() {
		var workDir string

		BeforeEach(func() {
			var err error
			workDir, err = os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			rootfsURI = fmt.Sprintf("oci:///%s/oci-test-images/opq-whiteouts-busybox:latest", workDir)
		})

		Context("when --disk-limit-size-bytes is more than compressed and less than uncompressed image size", func() {
			BeforeEach(func() {
				footCmd = newFootCommand(configFilePath, driverStoreDir, "create", rootfsURI, "some-handle", "--disk-limit-size-bytes", "668276")
			})

			It("prints an error", func() {
				expectErrorOutput("uncompressed layer size exceeds quota")
			})
		})
	})
})

func getUncompressedBlobSize(path string) int64 {
	f, err := os.Open(path)
	Expect(err).NotTo(HaveOccurred())
	defer f.Close()

	gzipReader, err := gzip.NewReader(f)
	Expect(err).NotTo(HaveOccurred())
	defer gzipReader.Close()

	bytes, err := ioutil.ReadAll(gzipReader)
	Expect(err).NotTo(HaveOccurred())

	return int64(len(bytes))
}
