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
	"code.cloudfoundry.org/lager/v3/lagertest"
	"github.com/containers/image/v5/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const emptyImageConfigBlob = "sha256:cdc9e972ad70c8dea1d185462fbc6120399e0839d02e33738b6c1c71b75b1c35"

var _ = Describe("Layer source: Docker", func() {
	var (
		layerSource source.LayerSource

		logger   *lagertest.TestLogger
		imageURL *url.URL

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

		layerInfos = []imagepuller.LayerInfo{
			{
				BlobID:    "sha256:742f13f887a914778308c47981b4601c01d0d98dad3e507c3c884c8cb78fe812",
				DiffID:    "707ae567c303ed86ab2069eefd9853e657efc2070d62a8ce0b5db014627a8f72",
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Size:      90,
			},
			{
				BlobID:    "sha256:85f1e021a8be19d023c275a080cdc3aaa72f4f405f0083eebe0c44479738ed37",
				DiffID:    "34cb21e628859162deee5826b04a29e7740e9f136dc63c2ec3ef4280e5b1ae83",
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Size:      89,
			},
		}

		fakeRegistry = testhelpers.NewFakeRegistry(urlParse("https://registry-1.docker.io"))

		logger = lagertest.NewTestLogger("test-layer-source")
		imageURL = urlParse("docker:///cfgarden/empty:groot")
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
			Expect(manifest.ConfigInfo().Digest.String()).To(Equal(emptyImageConfigBlob))
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
				imageURL = privateDockerImageURL()
				systemContext.DockerAuthConfig = privateDockerAuthConfig()
			})

			It("fetches the manifest", func() {
				By("not returning an error")
				Expect(manifestErr).NotTo(HaveOccurred())

				By("fetching the manifest")
				Expect(manifest.ConfigInfo().Digest.String()).To(Equal(emptyImageConfigBlob))
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
					systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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
				imageURL = urlParse("docker:cfgarden/empty:groot")
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
				Expect(logger).To(gbytes.Say("requested access to the resource is denied"))
			})
		})

		Context("when registry communication fails temporarily", func() {
			BeforeEach(func() {
				fakeRegistry.Start()
				fakeRegistry.FailNextManifestRequests(2)
				systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
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
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("fails to fetch the manifest", func() {
				Expect(manifestErr).To(HaveOccurred())
			})

			Context("when the private registry is whitelisted", func() {
				BeforeEach(func() {
					systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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
				imageURL = privateDockerImageURL()
				systemContext.DockerAuthConfig = privateDockerAuthConfig()
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
				imageURL = privateDockerImageURL()
			})

			It("does not return an error", func() {
				Expect(configErr).NotTo(HaveOccurred())
			})

			It("fetches the config", func() {
				Expect(config.RootFS.DiffIDs).To(HaveLen(2))
				Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal(layerInfos[0].DiffID))
				Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal(layerInfos[1].DiffID))
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
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
				systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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

				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
				systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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
				imageURL = privateDockerImageURL()
				systemContext.DockerAuthConfig = privateDockerAuthConfig()
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
					systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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
				imageURL = urlParse("docker:cfgarden/empty:groot")
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the blob does not exist", func() {
			BeforeEach(func() {
				layerInfos[0] = imagepuller.LayerInfo{BlobID: "sha256:3a50a9ff45117c33606ba54f4a7f55cebbdd2e96a06ab48e7e981a02ff1fd665"}
			})

			It("returns an error", func() {
				Expect(blobErr).To(MatchError(And(ContainSubstring("blob unknown to registry"), ContainSubstring("3a50a9ff45117c33606ba54f4a7f55cebbdd2e96a06ab48e7e981a02ff1fd665"))))
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
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
				systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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
				systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
				fakeRegistry.FailNextBlobRequests(2)
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
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
			})

			AfterEach(func() {
				fakeRegistry.Stop()
			})

			It("returns an error", func() {
				Expect(blobErr).To(HaveOccurred())
			})

			Context("when the private registry is whitelisted", func() {
				BeforeEach(func() {
					systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
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
				systemContext.DockerInsecureSkipTLSVerify = types.OptionalBoolTrue
				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:groot", fakeRegistry.Addr()))
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
