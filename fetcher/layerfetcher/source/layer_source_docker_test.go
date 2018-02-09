package source_test

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"code.cloudfoundry.org/groot/fetcher/layerfetcher"
	"code.cloudfoundry.org/groot/fetcher/layerfetcher/source"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/groot/testhelpers"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/containers/image/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Layer source: Docker", func() {
	var (
		layerSource source.LayerSource

		logger   *lagertest.TestLogger
		imageURL *url.URL

		configBlob    string
		layerInfos    []imagepuller.LayerInfo
		systemContext types.SystemContext

		skipOCILayerValidation bool
	)

	BeforeEach(func() {
		systemContext = types.SystemContext{
			DockerAuthConfig: &types.DockerAuthConfig{
				Username: RegistryUsername,
				Password: RegistryPassword,
			},
		}

		skipOCILayerValidation = false

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

		logger = lagertest.NewTestLogger("test-layer-source")
		imageURL = urlParse("docker:///cfgarden/empty:v0.1.1")
	})

	JustBeforeEach(func() {
		layerSource = source.NewLayerSource(systemContext, skipOCILayerValidation)
	})

	Describe("Manifest", func() {
		It("fetches the manifest", func() {
			manifest, err := layerSource.Manifest(logger, imageURL)
			Expect(err).NotTo(HaveOccurred())

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
				manifest, err := layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())

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
				imageURL = urlParse("docker:///cfgarden/private")

				configBlob = "sha256:c2bf00eb303023869c676f91af930a12925c24d677999917e8d52c73fa10b73a"
				layerInfos[0].BlobID = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
				layerInfos[0].DiffID = "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"
				layerInfos[1].BlobID = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"
			})

			Context("when the correct credentials are provided", func() {
				It("fetches the manifest", func() {
					manifest, err := layerSource.Manifest(logger, imageURL)
					Expect(err).NotTo(HaveOccurred())

					Expect(manifest.ConfigInfo().Digest.String()).To(Equal(configBlob))

					Expect(manifest.LayerInfos()).To(HaveLen(2))
					Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
					Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
					Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
					Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
				})
			})

			Context("when the registry returns a 401 when trying to get the auth token", func() {
				// We need a fake registry here because Dockerhub was rate limiting on multiple bad credential auth attempts
				var fakeRegistry *testhelpers.FakeRegistry

				BeforeEach(func() {
					dockerHubUrl := urlParse("https://registry-1.docker.io")
					fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
					fakeRegistry.Start()
					fakeRegistry.ForceTokenAuthError()
					imageURL = urlParse(fmt.Sprintf("docker://%s/doesnt-matter-because-fake-registry", fakeRegistry.Addr()))

					systemContext.DockerInsecureSkipTLSVerify = true
				})

				AfterEach(func() {
					fakeRegistry.Stop()
				})

				It("returns an informative error", func() {
					_, err := layerSource.Manifest(logger, imageURL)
					Expect(err).To(MatchError(ContainSubstring("unable to retrieve auth token")))
				})
			})
		})

		Context("when the image url is invalid", func() {
			It("returns an error", func() {
				url := urlParse("docker:cfgarden/empty:v0.1.0")

				_, err := layerSource.Manifest(logger, url)
				Expect(err).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the image does not exist", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker:///cfgarden/non-existing-image")

				systemContext.DockerAuthConfig.Username = ""
				systemContext.DockerAuthConfig.Password = ""
			})

			It("wraps the containers/image with a useful error", func() {
				_, err := layerSource.Manifest(logger, imageURL)
				Expect(err.Error()).To(MatchRegexp("^fetching image reference"))
			})

			It("logs the original error message", func() {
				_, err := layerSource.Manifest(logger, imageURL)
				Expect(err).To(HaveOccurred())

				Expect(logger).To(gbytes.Say("fetching-image-reference-failed"))
				Expect(logger).To(gbytes.Say("unauthorized: authentication required"))
			})
		})
	})

	Describe("Config", func() {
		It("fetches the config", func() {
			manifest, err := layerSource.Manifest(logger, imageURL)
			Expect(err).NotTo(HaveOccurred())
			config, err := manifest.OCIConfig()
			Expect(err).NotTo(HaveOccurred())

			Expect(config.RootFS.DiffIDs).To(HaveLen(2))
			Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal(layerInfos[0].DiffID))
			Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal(layerInfos[1].DiffID))
		})

		Context("when the image is private", func() {
			var manifest layerfetcher.Manifest

			BeforeEach(func() {
				imageURL = urlParse("docker:///cfgarden/private")
			})

			JustBeforeEach(func() {
				layerSource = source.NewLayerSource(systemContext, skipOCILayerValidation)
				var err error
				manifest, err = layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the correct credentials are provided", func() {
				It("fetches the config", func() {
					config, err := manifest.OCIConfig()
					Expect(err).NotTo(HaveOccurred())

					Expect(config.RootFS.DiffIDs).To(HaveLen(2))
					Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal("780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296"))
					Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal("56702ece901015f4f42dc82d1386c5ffc13625c008890d52548ff30dd142838b"))
				})
			})
		})

		Context("when the image url is invalid", func() {
			It("returns an error", func() {
				url := urlParse("docker:cfgarden/empty:v0.1.0")

				_, err := layerSource.Manifest(logger, url)
				Expect(err).To(MatchError(ContainSubstring("parsing url failed")))
			})
		})

		Context("when the image schema version is 1", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker://cfgarden/empty:schemaV1")
			})

			It("fetches the config", func() {
				manifest, err := layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())
				config, err := manifest.OCIConfig()
				Expect(err).NotTo(HaveOccurred())

				Expect(config.RootFS.DiffIDs).To(HaveLen(3))
				Expect(config.RootFS.DiffIDs[0].String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[0].DiffID))
				Expect(config.RootFS.DiffIDs[1].String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[1].DiffID))
				Expect(config.RootFS.DiffIDs[2].String()).To(Equal(testhelpers.SchemaV1EmptyImage.Layers[2].DiffID))
			})
		})
	})

	Context("when registry communication fails temporarily", func() {
		var fakeRegistry *testhelpers.FakeRegistry

		BeforeEach(func() {
			dockerHubUrl := urlParse("https://registry-1.docker.io")
			fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
			fakeRegistry.Start()

			systemContext.DockerInsecureSkipTLSVerify = true
			imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))
		})

		AfterEach(func() {
			fakeRegistry.Stop()
		})

		It("retries fetching the manifest twice", func() {
			fakeRegistry.FailNextRequests(2)

			_, err := layerSource.Manifest(logger, imageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-1"))
			Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-2"))
			Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-3"))
			Expect(logger.TestSink.LogMessages()).To(ContainElement("test-layer-source.fetching-image-manifest.attempt-get-image-success"))
		})

		It("retries fetching a blob twice", func() {
			fakeRegistry.FailNextRequests(2)

			blobPath, _, err := layerSource.Blob(logger, imageURL, layerInfos[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Remove(blobPath)).To(Succeed())

			expectedMessage := "test-layer-source.streaming-blob.attempt-get-blob-failed"
			Expect(logger.TestSink.LogMessages()).To(ContainElement(expectedMessage))
		})

		It("retries fetching the config blob twice", func() {
			fakeRegistry.WhenGettingBlob(configBlob, 1, func(resp http.ResponseWriter, req *http.Request) {
				resp.WriteHeader(http.StatusTeapot)
				_, _ = resp.Write([]byte("null"))
				return
			})

			_, err := layerSource.Manifest(logger, imageURL)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeRegistry.RequestedBlobs()).To(Equal([]string{configBlob}), "config blob was not prefetched within the retry")

			Expect(logger.TestSink.LogMessages()).To(
				ContainElement("test-layer-source.fetching-image-manifest.fetching-image-config-failed"))
		})
	})

	Context("when a private registry is used", func() {
		var fakeRegistry *testhelpers.FakeRegistry

		BeforeEach(func() {
			dockerHubUrl := urlParse("https://registry-1.docker.io")
			fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
			fakeRegistry.Start()

			imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))

		})

		AfterEach(func() {
			fakeRegistry.Stop()
		})

		It("fails to fetch the manifest", func() {
			_, err := layerSource.Manifest(logger, imageURL)
			Expect(err).To(HaveOccurred())
		})

		Context("when the private registry is whitelisted", func() {
			BeforeEach(func() {
				systemContext.DockerInsecureSkipTLSVerify = true
			})

			It("fetches the manifest", func() {
				manifest, err := layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(manifest.LayerInfos()).To(HaveLen(2))
				Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
				Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
				Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
				Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
			})

			It("fetches the config", func() {
				manifest, err := layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())

				config, err := manifest.OCIConfig()
				Expect(err).NotTo(HaveOccurred())

				Expect(config.RootFS.DiffIDs).To(HaveLen(2))
				Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal(layerInfos[0].DiffID))
				Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal(layerInfos[1].DiffID))
			})

			It("downloads and uncompresses the blob", func() {
				blobPath, size, err := layerSource.Blob(logger, imageURL, layerInfos[0])
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(blobPath)

				blobReader, err := os.Open(blobPath)
				Expect(err).NotTo(HaveOccurred())
				defer blobReader.Close()

				Expect(size).To(Equal(int64(90)))
				entries := tarEntries(blobReader)
				Expect(entries).To(ContainElement("hello"))
			})
		})

		Context("when using private images", func() {
			BeforeEach(func() {
				imageURL = urlParse("docker:///cfgarden/private")

				layerInfos[0].BlobID = "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4"
				layerInfos[0].DiffID = "780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296"
				layerInfos[1].BlobID = "sha256:48ce60c2de08a424e10810c41ec2f00916cfd0f12333e96eb4363eb63723be87"
				layerInfos[1].DiffID = "56702ece901015f4f42dc82d1386c5ffc13625c008890d52548ff30dd142838b"
			})

			JustBeforeEach(func() {
				layerSource = source.NewLayerSource(systemContext, skipOCILayerValidation)
			})

			It("fetches the manifest", func() {
				manifest, err := layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())

				Expect(manifest.LayerInfos()).To(HaveLen(2))
				Expect(manifest.LayerInfos()[0].Digest.String()).To(Equal(layerInfos[0].BlobID))
				Expect(manifest.LayerInfos()[0].Size).To(Equal(layerInfos[0].Size))
				Expect(manifest.LayerInfos()[1].Digest.String()).To(Equal(layerInfos[1].BlobID))
				Expect(manifest.LayerInfos()[1].Size).To(Equal(layerInfos[1].Size))
			})

			It("fetches the config", func() {
				manifest, err := layerSource.Manifest(logger, imageURL)
				Expect(err).NotTo(HaveOccurred())

				config, err := manifest.OCIConfig()
				Expect(err).NotTo(HaveOccurred())

				Expect(config.RootFS.DiffIDs).To(HaveLen(2))
				Expect(config.RootFS.DiffIDs[0].Hex()).To(Equal(layerInfos[0].DiffID))
				Expect(config.RootFS.DiffIDs[1].Hex()).To(Equal(layerInfos[1].DiffID))
			})

			It("downloads and uncompresses the blob", func() {
				blobPath, size, err := layerSource.Blob(logger, imageURL, layerInfos[0])
				Expect(err).NotTo(HaveOccurred())
				defer os.Remove(blobPath)

				blobReader, err := os.Open(blobPath)
				Expect(err).NotTo(HaveOccurred())
				defer blobReader.Close()

				Expect(size).To(Equal(int64(90)))
				entries := tarEntries(blobReader)
				Expect(entries).To(ContainElement("hello"))
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
			blobPath, blobSize, blobErr = layerSource.Blob(logger, imageURL, layerInfos[0])
		})

		AfterEach(func() {
			if _, err := os.Stat(blobPath); err == nil {
				Expect(os.Remove(blobPath)).To(Succeed())
			}
		})

		It("downloads and uncompresses the blob", func() {
			Expect(blobErr).NotTo(HaveOccurred())

			blobReader, err := os.Open(blobPath)
			Expect(err).NotTo(HaveOccurred())
			defer blobReader.Close()

			Expect(blobSize).To(Equal(int64(90)))
			entries := tarEntries(blobReader)
			Expect(entries).To(ContainElement("hello"))
		})

		Context("when the media type doesn't match the blob", func() {
			var fakeRegistry *testhelpers.FakeRegistry

			BeforeEach(func() {
				dockerHubUrl := urlParse("https://registry-1.docker.io")
				fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)

				fakeRegistry.WhenGettingBlob(layerInfos[0].BlobID, 1, func(rw http.ResponseWriter, req *http.Request) {
					_, _ = rw.Write([]byte("bad-blob"))
				})

				fakeRegistry.Start()

				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))

				systemContext.DockerInsecureSkipTLSVerify = true
				layerInfos[0].MediaType = "gzip"
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
				imageURL = urlParse("docker:///cfgarden/private")

				layerInfos = []imagepuller.LayerInfo{
					{
						BlobID:    "sha256:dabca1fccc91489bf9914945b95582f16d6090f423174641710083d6651db4a4",
						DiffID:    "780016ca8250bcbed0cbcf7b023c75550583de26629e135a1e31c0bf91fba296",
						MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
						Size:      90,
					},
				}
			})

			Context("when the correct credentials are provided", func() {
				It("fetches the config", func() {
					Expect(blobErr).NotTo(HaveOccurred())

					blobReader, err := os.Open(blobPath)
					Expect(err).NotTo(HaveOccurred())
					defer blobReader.Close()

					Expect(blobSize).To(Equal(int64(90)))
					entries := tarEntries(blobReader)
					Expect(entries).To(ContainElement("hello"))
				})
			})

			Context("when invalid credentials are provided", func() {
				// We need a fake registry here because Dockerhub was rate limiting on multiple bad credential auth attempts
				var fakeRegistry *testhelpers.FakeRegistry

				BeforeEach(func() {
					dockerHubUrl := urlParse("https://registry-1.docker.io")
					fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
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
			It("returns an error", func() {
				_, _, err := layerSource.Blob(logger, imageURL, imagepuller.LayerInfo{BlobID: "sha256:steamed-blob"})
				Expect(err).To(MatchError(ContainSubstring("fetching blob 404")))
			})
		})

		Context("when the blob is corrupted", func() {
			var fakeRegistry *testhelpers.FakeRegistry

			BeforeEach(func() {
				dockerHubUrl := urlParse("https://registry-1.docker.io")
				fakeRegistry = testhelpers.NewFakeRegistry(dockerHubUrl)
				fakeRegistry.WhenGettingBlob(layerInfos[0].BlobID, 1, func(rw http.ResponseWriter, req *http.Request) {
					gzipWriter := gzip.NewWriter(rw)
					_, _ = gzipWriter.Write([]byte("bad-blob"))
					gzipWriter.Close()
				})
				fakeRegistry.Start()

				imageURL = urlParse(fmt.Sprintf("docker://%s/cfgarden/empty:v0.1.1", fakeRegistry.Addr()))

				systemContext.DockerInsecureSkipTLSVerify = true
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
	})
})
