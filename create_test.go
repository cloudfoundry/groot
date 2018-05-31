package groot_test

import (
	"bytes"
	"io"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/grootfakes"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	imgspec "github.com/opencontainers/image-spec/specs-go/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	errors "github.com/pkg/errors"
)

var _ = Describe("Create", func() {
	var (
		driver                *grootfakes.FakeDriver
		imagePuller           *grootfakes.FakeImagePuller
		driverRuntimeSpec     = specs.Spec{Version: "some-version"}
		excludeImageFromQuota bool
		diskLimit             int64

		logger *lagertest.TestLogger
		g      *groot.Groot
	)

	BeforeEach(func() {
		driver = new(grootfakes.FakeDriver)
		driver.BundleReturns(driverRuntimeSpec, nil)
		imagePuller = new(grootfakes.FakeImagePuller)

		imagePuller.PullReturns(imagepuller.Image{
			ChainIDs: []string{"checksum"},
			Size:     1000,
		}, nil)

		logger = lagertest.NewTestLogger("groot")
		g = &groot.Groot{
			Driver:      driver,
			Logger:      logger,
			ImagePuller: imagePuller,
		}

		diskLimit = 5000
		excludeImageFromQuota = true
	})

	Describe("Create succeeding", func() {
		var (
			returnedRuntimeSpec specs.Spec
			rootfsFileBuffer    *bytes.Buffer
		)

		BeforeEach(func() {
			rootfsFileBuffer = bytes.NewBuffer([]byte{})
			driver.UnpackStub = func(logger lager.Logger, id string, parentIDs []string, layerTar io.Reader) (int64, error) {
				bytesWritten, err := io.Copy(rootfsFileBuffer, layerTar)
				Expect(err).NotTo(HaveOccurred())
				return bytesWritten, nil
			}
		})

		JustBeforeEach(func() {
			var err error
			returnedRuntimeSpec, err = g.Create("some-handle", diskLimit, excludeImageFromQuota)
			Expect(err).NotTo(HaveOccurred())
		})

		It("calls the image puller with the expected args", func() {
			Expect(imagePuller.PullCallCount()).To(Equal(1))
			_, spec := imagePuller.PullArgsForCall(0)
			Expect(spec).To(Equal(imagepuller.ImageSpec{
				DiskLimit:             diskLimit,
				ExcludeImageFromQuota: excludeImageFromQuota,
			}))
		})

		It("returns the runtime spec from driver.Bundle", func() {
			Expect(returnedRuntimeSpec).To(Equal(driverRuntimeSpec))
		})

		It("calls driver.Bundle with the handle and layer ids", func() {
			Expect(driver.BundleCallCount()).To(Equal(1))
			_, id, layerIDs, _ := driver.BundleArgsForCall(0)
			Expect(id).To(Equal("some-handle"))
			Expect(layerIDs).To(Equal([]string{"checksum"}))
		})

		It("calls driver.WriteMetadata with the handle and metadata", func() {
			Expect(driver.WriteMetadataCallCount()).To(Equal(1))
			_, id, metadata := driver.WriteMetadataArgsForCall(0)
			Expect(id).To(Equal("some-handle"))
			Expect(metadata).To(Equal(groot.ImageMetadata{Size: 1000}))
		})

		Context("image has env vars", func() {
			BeforeEach(func() {
				imagePuller.PullReturns(imagepuller.Image{
					ChainIDs: []string{"checksum"},
					Size:     1000,
					Config: imgspec.Image{
						Config: imgspec.ImageConfig{
							Env: []string{"foo=bar"},
						},
					},
				}, nil)

				driverRuntimeSpec.Process = &specs.Process{
					Env: []string{"foo=bar"},
				}
			})

			It("includes env information in the returned spec", func() {
				Expect(returnedRuntimeSpec).To(Equal(driverRuntimeSpec))
			})
		})

		Context("exclude image from quota is true", func() {
			It("passes the disk limit directly to driver.Bundle", func() {
				Expect(driver.BundleCallCount()).To(Equal(1))
				_, _, _, diskLimit := driver.BundleArgsForCall(0)
				Expect(diskLimit).To(Equal(int64(5000)))
			})
		})

		Context("exclude image from quota is false", func() {
			BeforeEach(func() {
				excludeImageFromQuota = false
			})

			It("subtracts the image size from the disk limit when calling driver.Bundle", func() {
				Expect(driver.BundleCallCount()).To(Equal(1))
				_, _, _, diskLimit := driver.BundleArgsForCall(0)
				Expect(diskLimit).To(Equal(int64(4000)))
			})

			Context("the disk limit is zero", func() {
				BeforeEach(func() {
					diskLimit = 0
				})

				It("passes the disk limit directly to driver.Bundle", func() {
					Expect(driver.BundleCallCount()).To(Equal(1))
					_, _, _, diskLimit := driver.BundleArgsForCall(0)
					Expect(diskLimit).To(Equal(int64(0)))
				})
			})
		})
	})

	Describe("Create failing", func() {
		var (
			createErr error
		)

		JustBeforeEach(func() {
			_, createErr = g.Create("some-handle", diskLimit, excludeImageFromQuota)
		})

		Context("when image puller returns an error", func() {
			BeforeEach(func() {
				imagePuller.PullReturns(imagepuller.Image{}, errors.New("pull-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("pull-failed")))
			})
		})

		Context("the disk limit is negative", func() {
			BeforeEach(func() {
				diskLimit = -500
			})

			It("returns an error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("invalid disk limit: -500")))
			})
		})

		Context("the disk limit is smaller than the base image size", func() {
			BeforeEach(func() {
				excludeImageFromQuota = false
				diskLimit = 500
			})

			It("returns an error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("disk limit 500 must be larger than image size 1000")))
			})
		})

		Context("the disk limit is equal to the base image size", func() {
			BeforeEach(func() {
				excludeImageFromQuota = false
				diskLimit = 1000
			})

			It("returns an error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("disk limit 1000 must be larger than image size 1000")))
			})
		})

		Context("when driver.Bundle returns an error", func() {
			BeforeEach(func() {
				driver.BundleReturns(specs.Spec{}, errors.New("bundle-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("bundle-failed")))
			})
		})
	})
})
