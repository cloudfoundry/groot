package source_test

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"code.cloudfoundry.org/groot/fetcher/layerfetcher/source"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/groot/testhelpers"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/containers/image/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Layer source: Docker", func() {
	var (
		layerSource source.LayerSource

		logger   *lagertest.TestLogger
		imageURL *url.URL

		configBlob    string
		layerInfos    []imagepuller.LayerInfo
		systemContext types.SystemContext

		fakeRegistry *testhelpers.FakeRegistry

		skipOCILayerValidation   bool
		skipImageQuotaValidation bool
		imageQuota               int64
	)

	BeforeEach(func() {
		systemContext = types.SystemContext{DockerAuthConfig: new(types.DockerAuthConfig)}

		skipOCILayerValidation = false
		skipImageQuotaValidation = true
		imageQuota = 0

		configBlob = "sha256:217f3b4afdf698d639f854d9c6d640903a011413bc7e7bffeabe63c7ca7e4a7d"
		layerInfos = []imagepuller.LayerInfo{
			{
				BlobID:    "sha256:47e3dd80d678c83c50cb133f4cf20e94d088f890679716c8b763418f55827a58",
				DiffID:    "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5",
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Size:      90,
			},
			{
				BlobID:    "sha256:7f2760e7451ce455121932b178501d60e651f000c3ab3bc12ae5d1f57614cc76",
				DiffID:    "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0",
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Size:      88,
			},
		}

		fakeRegistry = testhelpers.NewFakeRegistry(urlParse("https://registry-1.docker.io"))

		logger = lagertest.NewTestLogger("test-layer-source")
		imageURL = urlParse("docker:///cfgarden/empty:v0.1.1")
	})

	JustBeforeEach(func() {
		layerSource = source.NewLayerSource(systemContext, skipOCILayerValidation, skipImageQuotaValidation, imageQuota, imageURL)
	})

	Describe("Manifest", func() {
		var (
			manifest    types.Image
			manifestErr error
		)

		JustBeforeEach(func() {
			manifest, manifestErr = layerSource.Manifest(logger)
		})

		It("fetches the manifest", func() {
			By("not returning an error")
			Expect(manifestErr).NotTo(HaveOccurred())

			By("fetching the manifest")
			Expect(manifest.ConfigInfo().Digest.String()).To(Equal(configBlob))
			Expect(manifest.LayerInfos()).To(HaveLen(2))
			Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
			Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
			Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
			Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
		})

		Context("when the image schema version is 1", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker://cfgarden/empty:schemaV1")
			})

			It("fetches the manifest", func() {
				By("not returning an error")
				Expect(manifestErr).NotTo(HaveOccurred())

				By("fetching the manifest")
				Expect(manifest.ConfigInfo().Digest.String()).To(Equal(testhelpers.SchemaV1EmptyImage.ConfigBlobID))
				Expect(manifest.LayerInfos()).To(HaveLen(3))
				Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[0].BlobID))
				Expect(manifest.LayerInfos()[0].Size).To(Equal(int64(-1)))
				Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[1].BlobID))
				Expect(manifest.LayerInfos()[1].Size).To(Equal(int64(-1)))
				Expect(manifest.LayerInfos()[2].Digest.String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[2].BlobID))
				Expect(manifest.LayerInfos()[2].Size).To(Equal(int64(-1)))
			})
		})

		Context("when the image is private", func() {
			BeforeEach(func() {
				maybeSkipPrivateDockerRegistryTest()
				imageURL = urlParse("docker:///cfgarden/private")
				systemContext.DockerAuthConfig = privateDockerAuthConfig()

				configBlob = "sha256:c2bf00eb303023869c676f91af930a12925c24d677999917e8d52c73fa10b73a"
				layerInfos[0].BlobID = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
				layerInfos[0].DiffID = "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"
				layerInfos[1].BlobID = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"
			})

			It("fetches the manifest", func() {
				By("not returning an error")
				Expect(manifestErr).NotTo(HaveOccurred())

				By("fetching the manifest")
				Expect(manifest.ConfigInfo().Digest.String()).To(Equal(configBlob))
				Expect(manifest.LayerInfos()).To(HaveLen(2))
				Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
				Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
				Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
				Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
			})

			Context("when the registry returns a 401 when trying to get the auth token", func() {
				BeforeEach(func() {
					fakeRegistry.Start()
					fakeRegistry.ForceTokenAuthError()
					imageURL = urlParse(fmt.Sprintf("docker://%s/doesnt-matter-because-fake-registry", fakeRegistry.Addr()))
					systemContext.DockerInsecureSkipTLSVerify = true
				})

				AfterEach(func() {
					fakeRegistry.Stop()
				})

				It("returns an informative error", func() {
					Expect(manifestErr).To(MatchError(ContainSubstring("unable to retrieve auth token")))
				})
			})
		})

		Context("when the image url is invalid", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker:cfgarden/empty:v0.1.0")
			})

			It("returns an error", func() {
				Expect(manifestErr).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the image does not exist", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker:///cfgarden/non-existing-image")

				systemContext.DockerAuthConfig.Username = ""
				systemContext.DockerAuthConfig.Password = ""
			})

			It("wraps the containers/image with a useful error", func() {
				Expect(manifestErr.Error()).To(MatchRegexp("^fetching image reference"))
			})

			It("logs the original error message", func() {
				Expect(logger).To(gbytes.Say("fetching-image-reference-failed"))
				Expect(logger).To(gbytes.Say("unauthorized: authentication required"))
			})
		})

		Context("when registry communication fails temporarily", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				fakeRegistry.FailNextRequests(2)
				systemContext.DockerInsecureSkipTLSVerify = true
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("does not return an error", func() {
				Expect(manifestErr).NotTo(HaveOccurred())
			})

			It("retries fetching the manifest twice", func() {
				Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-1"))
				Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-2"))
				Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-3"))
				Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-success"))
			})
		})

		Context("when a private registry is used", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("fails to fetch the manifest", func() {
				Expect(manifestErr).To(HaveOccurred())
			})

			Context("when the private registry is whitelisted", func() {
				BeforeEach(func() {
					systemContext.DockerInsecureSkipTLSVerify = true
				})

				It("fetches the manifest", func() {
					By("not returning an error")
					Expect(manifestErr).NotTo(HaveOccurred())

					By("fetching the manifest")
					Expect(manifest.LayerInfos()).To(HaveLen(2))
					Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
					Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
					Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
					Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
				})
			})
		})

		Context("when using private images", func() {
			BeforeEach(func() {
				maybeSkipPrivateDockerRegistryTest()
				imageURL = urlParse("docker:///cfgarden/private")
				systemContext.DockerAuthConfig = privateDockerAuthConfig()

				layerInfos[0].BlobID = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
				layerInfos[0].DiffID = "780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296"
				layerInfos[1].BlobID = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"
				layerInfos[1].DiffID = "56702ece901015f4f42dc82d1386c5ffc13625c008890d52548ff30dd142838b"
			})

			It("fetches the manifest", func() {
				By("not returning an error")
				Expect(manifestErr).NotTo(HaveOccurred())

				By("fetching the manifest")
				Expect(manifest.LayerInfos()).To(HaveLen(2))
				Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
				Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
				Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
				Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
			})
		})
	})

	Describe("Config", func() {
		var (
			config    *v1.Image
			configErr error
		)

		JustBeforeEach(func() {
			manifest, err := layerSource.Manifest(logger)
			Expect(err).NotTo(HaveOccurred())
			config, configErr = manifest.OCIConfig(context.TODO())
		})

		It("does not return an error", func() {
			Expect(configErr).NotTo(HaveOccurred())
		})

		It("fetches the config", func() {
			Expect(config.RootFS.DiffIDs).To(HaveLen(2))
			Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal(layerInfos[0].DiffID))
			Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal(layerInfos[1].DiffID))
		})

		Context("when the image is private", func() {
			BeforeEach(func() {
				maybeSkipPrivateDockerRegistryTest()
				systemContext.DockerAuthConfig = privateDockerAuthConfig()
				imageURL = urlParse("docker:///cfgarden/private")
			})

			It("does not return an error", func() {
				Expect(configErr).NotTo(HaveOccurred())
			})

			It("fetches the config", func() {
				Expect(config.RootFS.DiffIDs).To(HaveLen(2))
				Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal("780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296"))
				Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal("56702ece901015f4f42dc82d1386c5ffc13625c008890d52548ff30dd142838b"))
			})
		})

		Context("when the image schema version is 1", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker://cfgarden/empty:schemaV1")
			})

			It("does not return an error", func() {
				Expect(configErr).NotTo(HaveOccurred())
			})

			It("fetches the config", func() {
				Expect(config.RootFS.DiffIDs).To(HaveLen(3))
				Expect(config.RootFS.DiffIDs[0].String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[0].DiffID))
				Expect(config.RootFS.DiffIDs[1].String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[1].DiffID))
				Expect(config.RootFS.DiffIDs[2].String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[2].DiffID))
			})
		})

		Context("when a private registry is used and it is whitelisted", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
				systemContext.DockerInsecureSkipTLSVerify = true
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("fetches the config", func() {
				By("not returning an error")
				Expect(configErr).NotTo(HaveOccurred())

				By("fetching the config")
				Expect(config.RootFS.DiffIDs).To(HaveLen(2))
				Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal(layerInfos[0].DiffID))
				Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal(layerInfos[1].DiffID))
			})
		})
	})

	Describe("Blob", func() {
		var (
			blobPath string
			blobSize int64
			blobErr  error
		)

		JustBeforeEach(func() {
			blobPath, blobSize, blobErr = layerSource.Blob(logger, layerInfos[0])
		})

		AfterEach(func() {
			if _, err := os.Stat(blobPath); err == nil {
				Expect(os.Remove(blobPath)).To(Succeed())
			}
		})

		It("does not return an error", func() {
			Expect(blobErr).NotTo(HaveOccurred())
		})

		It("downloads and uncompresses the blob", func() {
			blobReader := open(blobPath)
			defer blobReader.Close()

			Expect(blobSize).To(Equal(int64(90)))
			expectTarArchiveToContainHello(blobReader)
		})

		Context("when the media type doesn't match the blob", func() {
			BeforeEach(func() {
				fakeRegistry.WhenGettingBlob(layerInfos[0].BlobID, 1, func(rw http.ResponseWriter, req *http.Request) {
					_, _ = io.WriteString(rw, "bad-blob")
				})

				fakeRegistry.Start()

				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
				systemContext.DockerInsecureSkipTLSVerify = true
				layerInfos[0].MediaType = "gzip"
				layerInfos[0].Size = 8
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(ContainSubstring("expected blob to be of type")))
			})
		})

		Context("when the image is private", func() {
			BeforeEach(func() {
				maybeSkipPrivateDockerRegistryTest()
				imageURL = urlParse("docker:///cfgarden/private")
				systemContext.DockerAuthConfig = privateDockerAuthConfig()

				layerInfos = []imagepuller.LayerInfo{
					{
						BlobID:    "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4",
						DiffID:    "780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296",
						MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
						Size:      90,
					},
				}
			})

			It("does not return an error", func() {
				Expect(blobErr).NotTo(HaveOccurred())
			})

			It("downloads and uncompresses the blob", func() {
				blobReader := open(blobPath)
				defer blobReader.Close()

				Expect(blobSize).To(Equal(int64(90)))
				expectTarArchiveToContainHello(blobReader)
			})

			Context("when invalid credentials are provided", func() {
				BeforeEach(func() {
					fakeRegistry.Start()
					fakeRegistry.ForceTokenAuthError()
					imageURL = urlParse(fmt.Sprintf("docker://%s/doesnt-matter-because-fake-registry", fakeRegistry.Addr()))
					systemContext.DockerInsecureSkipTLSVerify = true
				})

				AfterEach(func() {
					fakeRegistry.Stop()
				})

				It("retuns an error", func() {
					Expect(blobErr).To(MatchError(ContainSubstring("unable to retrieve auth token")))
				})
			})
		})

		Context("when the image url is invalid", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker:cfgarden/empty:v0.1.0")
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the blob does not exist", func() {
			BeforeEach(func() {
				layerInfos[0] = imagepuller.LayerInfo{BlobID: "sha256:steamed-blob"}
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(ContainSubstring("fetching blob 400")))
			})
		})

		Context("when the blob is corrupted", func() {
			BeforeEach(func() {
				fakeRegistry.WhenGettingBlob(layerInfos[0].BlobID, 1, func(rw http.ResponseWriter, req *http.Request) {
					gzipWriter := gzip.NewWriter(rw)
					_, _ = io.WriteString(gzipWriter, "bad-blob")
					gzipWriter.Close()
				})
				fakeRegistry.Start()
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
				systemContext.DockerInsecureSkipTLSVerify = true
				layerInfos[0].Size = 32
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(ContainSubstring("layerID digest mismatch")))
			})

			Context("when a devious hacker tries to set skipOCILayerValidation to true", func() {
				BeforeEach(func() {
					skipOCILayerValidation = true
				})

				It("returns an error", func() {
					Expect(blobErr).To(MatchError(ContainSubstring("layerID digest mismatch")))
				})
			})
		})

		Context("when the blob doesn't match the diffID", func() {
			BeforeEach(func() {
				layerInfos[0].DiffID = "0000000000000000000000000000000000000000000000000000000000000000"
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(ContainSubstring("diffID digest mismatch")))
			})
		})

		Context("when registry communication fails temporarily", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				systemContext.DockerInsecureSkipTLSVerify = true
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
				fakeRegistry.FailNextRequests(2)
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("retries fetching a blob twice", func() {
				Expect(blobErr).NotTo(HaveOccurred())
				expectedMessage := "test-layer-source.streaming-blob.attempt-get-blob-failed"
				Expect(logger.TestSink.LogMessages()).To(ContainElement(expectedMessage))
			})
		})

		Context("when a private registry is used", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("returns an error", func() {
				Expect(blobErr).To(HaveOccurred())
			})

			Context("when the private registry is whitelisted", func() {
				BeforeEach(func() {
					systemContext.DockerInsecureSkipTLSVerify = true
				})

				It("does not return an error", func() {
					Expect(blobErr).NotTo(HaveOccurred())
				})

				It("downloads and uncompresses the blob", func() {
					blobReader := open(blobPath)
					defer blobReader.Close()

					Expect(blobSize).To(Equal(int64(90)))
					expectTarArchiveToContainHello(blobReader)
				})
			})
		})

		Context("when registry communication fails temporarily", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				systemContext.DockerInsecureSkipTLSVerify = true
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
				fakeRegistry.WhenGettingBlob(layerInfos[0].BlobID, 1, func(resp http.ResponseWriter, req *http.Request) {
					resp.WriteHeader(http.StatusTeapot)
					_, _ = io.WriteString(resp, "null")
					return
				})
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("retries fetching the blob twice", func() {
				By("not returning an error")
				Expect(blobErr).NotTo(HaveOccurred())

				By("retrying to get the blob")
				Expect(fakeRegistry.RequestedBlobs()).To(Equal([]string{layerInfos[0].BlobID}))
				Expect(logger).To(gbytes.Say("test-layer-source.streaming-blob.attempt-get-blob-2"))
				Expect(logger).To(gbytes.Say("test-layer-source.streaming-blob.attempt-get-blob-success"))
			})
		})

		Context("when image quota validation is not skipped", func() {
			BeforeEach(func() {
				skipImageQuotaValidation = false
			})

			Context("when the uncompressed layer size is bigger that the quota", func() {
				BeforeEach(func() {
					imageQuota = 1
				})

				It("returns quota exceeded error", func() {
					_, _, err := layerSource.Blob(logger, layerInfos[0])
					Expect(err).To(MatchError(ContainSubstring("uncompressed layer size exceeds quota")))
				})
			})
		})
	})

	Describe("Close", func() {
		It("can close prior any interactions", func() {
			Expect(layerSource.Close()).To(Succeed())
		})
	})
})

func expectTarArchiveToContainHello(tar io.Reader) {
	entries := tarEntries(tar)
	Expect(entries).To(ContainElement("hello"))
}
