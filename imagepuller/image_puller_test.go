package imagepuller_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/groot/imagepuller/imagepullerfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("Image Puller", func() {
	var (
		logger           lager.Logger
		fakeFetcher      *imagepullerfakes.FakeFetcher
		fakeVolumeDriver *imagepullerfakes.FakeVolumeDriver
		expectedImgDesc  specsv1.Image

		imagePuller *imagepuller.ImagePuller
		layerInfos  []imagepuller.LayerInfo

		tmpVolumesDir string
	)

	BeforeEach(func() {
		fakeFetcher = new(imagepullerfakes.FakeFetcher)
		expectedImgDesc = specsv1.Image{Author: "Groot"}
		layerInfos = []imagepuller.LayerInfo{
			{BlobID: "i-am-a-layer", ChainID: "layer-111", ParentChainID: "", Size: 111},
			{BlobID: "i-am-another-layer", ChainID: "chain-222", ParentChainID: "layer-111", Size: 222},
			{BlobID: "i-am-the-last-layer", ChainID: "chain-333", ParentChainID: "chain-222", Size: 333},
		}
		fakeFetcher.ImageInfoReturns(
			imagepuller.ImageInfo{
				LayerInfos: layerInfos,
				Config:     expectedImgDesc,
			}, nil)

		fakeFetcher.StreamBlobStub = func(_ lager.Logger, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
			buffer := bytes.NewBuffer([]byte{})
			stream := gzip.NewWriter(buffer)
			defer stream.Close()
			return ioutil.NopCloser(buffer), 0, nil
		}

		tmpVolumesDir = tempDir("", "volumes")

		fakeVolumeDriver = new(imagepullerfakes.FakeVolumeDriver)
		count := 0
		fakeVolumeDriver.UnpackStub = func(_ lager.Logger, layerID string, parentIDs []string, layerTar io.Reader) (int64, error) {
			size := layerInfos[count].Size
			count++
			return size, nil
		}
		imagePuller = imagepuller.NewImagePuller(fakeFetcher, fakeVolumeDriver)
		logger = lagertest.NewTestLogger("image-puller")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpVolumesDir)).To(Succeed())
	})

	It("returns the image description", func() {
		image, err := imagePuller.Pull(logger, imagepuller.ImageSpec{})
		Expect(err).NotTo(HaveOccurred())

		Expect(image.Config).To(Equal(expectedImgDesc))
	})

	It("returns the chain ids in the order specified by the image", func() {
		image, _ := imagePuller.Pull(logger, imagepuller.ImageSpec{})
		Expect(image.ChainIDs).To(Equal([]string{"layer-111", "chain-222", "chain-333"}))
	})

	It("returns the total size of the base image", func() {
		image, _ := imagePuller.Pull(logger, imagepuller.ImageSpec{})
		Expect(image.Size).To(Equal(int64(666)))
	})

	It("passes the correct parentIDs to Unpack", func() {
		imagePuller.Pull(logger, imagepuller.ImageSpec{})

		Expect(fakeVolumeDriver.UnpackCallCount()).To(Equal(3))

		_, _, parentIDs, _ := fakeVolumeDriver.UnpackArgsForCall(0)
		Expect(parentIDs).To(BeEmpty())
		_, _, parentIDs, _ = fakeVolumeDriver.UnpackArgsForCall(1)
		Expect(parentIDs).To(Equal([]string{"layer-111"}))
		_, _, parentIDs, _ = fakeVolumeDriver.UnpackArgsForCall(2)
		Expect(parentIDs).To(Equal([]string{"layer-111", "chain-222"}))
	})

	It("unpacks the layers got from the fetcher", func() {
		fakeFetcher.StreamBlobStub = func(_ lager.Logger, layerInfo imagepuller.LayerInfo) (io.ReadCloser, int64, error) {
			buffer := bytes.NewBuffer([]byte{})
			stream := gzip.NewWriter(buffer)
			defer stream.Close()
			writeString(stream, fmt.Sprintf("layer-%s-contents", layerInfo.BlobID))
			return ioutil.NopCloser(buffer), 1200, nil
		}

		imagePuller.Pull(logger, imagepuller.ImageSpec{})

		Expect(fakeVolumeDriver.UnpackCallCount()).To(Equal(3))

		validateLayer := func(idx int, expected string) {
			_, _, _, stream := fakeVolumeDriver.UnpackArgsForCall(idx)
			Expect(readAll(gzipNewReader(stream))).To(Equal(expected))
		}

		validateLayer(0, "layer-i-am-a-layer-contents")
		validateLayer(1, "layer-i-am-another-layer-contents")
		validateLayer(2, "layer-i-am-the-last-layer-contents")
	})

	Context("when the layers size in the manifest will exceed the limit", func() {
		Context("when including the image size in the limit", func() {
			It("returns an error", func() {
				fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
					LayerInfos: []imagepuller.LayerInfo{
						{Size: 1000},
						{Size: 201},
					},
				}, nil)

				_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
					DiskLimit:             1200,
					ExcludeImageFromQuota: false,
				})
				Expect(err).To(MatchError(ContainSubstring("layers exceed disk quota")))
			})

			Context("when the disk limit is zero", func() {
				It("doesn't fail", func() {
					fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
						LayerInfos: []imagepuller.LayerInfo{
							{Size: 1000},
							{Size: 201},
						},
					}, nil)

					_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
						DiskLimit:             0,
						ExcludeImageFromQuota: false,
					})

					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when not including the image size in the limit", func() {
			It("doesn't fail", func() {
				fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
					LayerInfos: []imagepuller.LayerInfo{
						{Size: 1000},
						{Size: 201},
					},
				}, nil)

				_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{
					DiskLimit:             1024,
					ExcludeImageFromQuota: true,
				})

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("when fetching the list of layers fails", func() {
		BeforeEach(func() {
			fakeFetcher.ImageInfoReturns(imagepuller.ImageInfo{
				LayerInfos: []imagepuller.LayerInfo{},
				Config:     specsv1.Image{},
			}, errors.New("failed to get list of layers"))
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{})
			Expect(err).To(MatchError(ContainSubstring("failed to get list of layers")))
		})
	})

	Context("when unpacking a volume fails", func() {
		BeforeEach(func() {
			fakeVolumeDriver.UnpackReturns(0, errors.New("failed to create volume"))
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{})
			Expect(err).To(MatchError(ContainSubstring("failed to create volume")))
		})
	})

	Context("when streaming a blob fails", func() {
		BeforeEach(func() {
			fakeFetcher.StreamBlobReturns(nil, 0, errors.New("failed to stream blob"))
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{})
			Expect(err).To(MatchError(ContainSubstring("failed to stream blob")))
		})
	})

	Context("when unpacking a child blob fails", func() {
		BeforeEach(func() {
			count := 0
			fakeVolumeDriver.UnpackStub = func(_ lager.Logger, id string, parentIDs []string, stream io.Reader) (int64, error) {
				count++
				if count == 3 {
					return 0, errors.New("failed to unpack the blob")
				}

				return 0, nil
			}
		})

		It("returns an error", func() {
			_, err := imagePuller.Pull(logger, imagepuller.ImageSpec{})
			Expect(err).To(MatchError(ContainSubstring("failed to unpack the blob")))
		})
	})
})
