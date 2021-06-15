package layerfetcher_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/groot/imagepuller"

	"code.cloudfoundry.org/groot/fetcher/layerfetcher"
	"code.cloudfoundry.org/groot/fetcher/layerfetcher/layerfetcherfakes"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/containers/image/v5/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	digestpkg "github.com/opencontainers/go-digest"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("LayerFetcher", func() {
	var (
		fakeSource        *layerfetcherfakes.FakeSource
		fetcher           *layerfetcher.LayerFetcher
		logger            *lagertest.TestLogger
		gzipedBlobContent string
	)

	BeforeEach(func() {
		fakeSource = new(layerfetcherfakes.FakeSource)

		gzipBuffer := bytes.NewBuffer([]byte{})
		gzipWriter := gzip.NewWriter(gzipBuffer)
		writeString(gzipWriter, "hello-world")
		Expect(gzipWriter.Close()).To(Succeed())
		gzipedBlobContent = readAll(gzipBuffer)

		fetcher = layerfetcher.NewLayerFetcher(fakeSource)

		var err error
		logger = lagertest.NewTestLogger("test-layer-fetcher")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ImageInfo", func() {
		It("fetches the manifest", func() {
			fakeManifest := new(layerfetcherfakes.FakeManifest)
			fakeManifest.OCIConfigReturns(&specsv1.Image{}, nil)
			fakeSource.ManifestReturns(fakeManifest, nil)

			_, err := fetcher.ImageInfo(logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSource.ManifestCallCount()).To(Equal(1))
		})

		Context("when fetching the manifest fails", func() {
			BeforeEach(func() {
				fakeSource.ManifestReturns(nil, errors.New("fetching the manifest"))
			})

			It("returns an error", func() {
				_, err := fetcher.ImageInfo(logger)
				Expect(err).To(MatchError(ContainSubstring("fetching the manifest")))
			})
		})

		It("returns the correct list of layer digests", func() {
			config := &specsv1.Image{
				RootFS: specsv1.RootFS{
					DiffIDs: []digestpkg.Digest{
						digestpkg.NewDigestFromHex("sha256", "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"),
						digestpkg.NewDigestFromHex("sha256", "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0"),
					},
				},
			}
			fakeManifest := new(layerfetcherfakes.FakeManifest)
			fakeManifest.OCIConfigReturns(config, nil)
			fakeManifest.LayerInfosReturns([]types.BlobInfo{
				types.BlobInfo{
					Digest:      digestpkg.NewDigestFromHex("sha256", "47e3dd80d678c83c50cb133f4cf20e94d088f890679716c8b763418f55827a58"),
					Size:        1024,
					Annotations: map[string]string{"org.cloudfoundry.experimental.image.base-directory": "/home/cool-user"},
				},
				types.BlobInfo{
					Digest: digestpkg.NewDigestFromHex("sha256", "7f2760e7451ce455121932b178501d60e651f000c3ab3bc12ae5d1f57614cc76"),
					Size:   2048,
				},
			})
			fakeSource.ManifestReturns(fakeManifest, nil)

			imageInfo, err := fetcher.ImageInfo(logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(imageInfo.LayerInfos).To(Equal([]imagepuller.LayerInfo{
				imagepuller.LayerInfo{
					BlobID:        "sha256:47e3dd80d678c83c50cb133f4cf20e94d088f890679716c8b763418f55827a58",
					ChainID:       "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5",
					DiffID:        "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5",
					ParentChainID: "",
					Size:          1024,
				},
				imagepuller.LayerInfo{
					BlobID:        "sha256:7f2760e7451ce455121932b178501d60e651f000c3ab3bc12ae5d1f57614cc76",
					ChainID:       "9242945d3c9c7cf5f127f9352fea38b1d3efe62ee76e25f70a3e6db63a14c233",
					DiffID:        "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0",
					ParentChainID: "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5",
					Size:          2048,
				},
			}))
		})

		Context("when retrieving the OCI Config fails", func() {
			BeforeEach(func() {
				fakeManifest := new(layerfetcherfakes.FakeManifest)
				fakeManifest.OCIConfigReturns(&specsv1.Image{}, errors.New("OCI Config retrieval failed"))
				fakeSource.ManifestReturns(fakeManifest, nil)
			})

			It("returns the error", func() {
				_, err := fetcher.ImageInfo(logger)
				Expect(err).To(MatchError(ContainSubstring("OCI Config retrieval failed")))
			})
		})

		It("returns the correct OCI image config", func() {
			timestamp := time.Time{}.In(time.UTC)
			expectedConfig := specsv1.Image{
				Created: &timestamp,
				RootFS: specsv1.RootFS{
					DiffIDs: []digestpkg.Digest{
						digestpkg.NewDigestFromHex("sha256", "afe200c63655576eaa5cabe036a2c09920d6aee67653ae75a9d35e0ec27205a5"),
						digestpkg.NewDigestFromHex("sha256", "d7c6a5f0d9a15779521094fa5eaf026b719984fb4bfe8e0012bd1da1b62615b0"),
					},
				},
			}

			fakeManifest := new(layerfetcherfakes.FakeManifest)
			fakeManifest.OCIConfigReturns(&expectedConfig, nil)
			fakeSource.ManifestReturns(fakeManifest, nil)

			imageInfo, err := fetcher.ImageInfo(logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(imageInfo.Config).To(Equal(expectedConfig))
		})
	})

	Describe("StreamBlob", func() {
		var (
			layerInfo = imagepuller.LayerInfo{
				BlobID: "sha256:layer-digest",
			}
			tmpFile *os.File
		)

		BeforeEach(func() {
			tmpFile = tempFile()
			defer tmpFile.Close()
			writeString(tmpFile, gzipedBlobContent)

			fakeSource.BlobReturns(tmpFile.Name(), 0, nil)
		})

		AfterEach(func() {
			err := os.Remove(tmpFile.Name())
			Expect(tmpFile.Name()).NotTo(BeAnExistingFile(), fmt.Sprintf("%v", err))
		})

		It("uses the source", func() {
			stream, _, err := fetcher.StreamBlob(logger, layerInfo)
			Expect(err).NotTo(HaveOccurred())
			Expect(stream.Close()).To(Succeed())

			Expect(fakeSource.BlobCallCount()).To(Equal(1))
			Expect(layerInfo.BlobID).To(Equal("sha256:layer-digest"))
		})

		It("returns the stream from the source", func() {
			stream, _, err := fetcher.StreamBlob(logger, layerInfo)
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()

			gzipReader, err := gzip.NewReader(stream)
			Expect(err).NotTo(HaveOccurred())
			Expect(readAll(gzipReader)).To(Equal("hello-world"))
		})

		It("returns the size of the stream", func() {
			gzipWriter := gzip.NewWriter(tmpFile)
			defer gzipWriter.Close()

			fakeSource.BlobReturns(tmpFile.Name(), 1024, nil)

			stream, size, err := fetcher.StreamBlob(logger, layerInfo)
			Expect(err).NotTo(HaveOccurred())
			defer stream.Close()
			Expect(size).To(Equal(int64(1024)))
		})

		Context("when the source fails to stream the blob", func() {
			It("returns an error", func() {
				fakeSource.BlobReturns("", 0, errors.New("failed to stream blob"))

				_, _, err := fetcher.StreamBlob(logger, layerInfo)
				Expect(err).To(MatchError(ContainSubstring("failed to stream blob")))
			})
		})
	})

	Describe("Close", func() {
		It("closes the source", func() {
			Expect(fetcher.Close()).To(Succeed())
			Expect(fakeSource.CloseCallCount()).To(Equal(1))
		})
	})
})
