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
	errors "github.com/pkg/errors"
)

var _ = Describe("Pull", func() {
	var (
		imagePuller *grootfakes.FakeImagePuller
		driver      *grootfakes.FakeDriver

		logger *lagertest.TestLogger
		g      *groot.Groot
	)

	BeforeEach(func() {

		imagePuller = new(grootfakes.FakeImagePuller)
		driver = new(grootfakes.FakeDriver)

		imagePuller.PullReturns(imagepuller.Image{
			ChainIDs: []string{"checksum"},
		}, nil)

		logger = lagertest.NewTestLogger("groot")
		g = &groot.Groot{
			Driver:      driver,
			Logger:      logger,
			ImagePuller: imagePuller,
		}
	})

	Describe("Pull succeeding", func() {
		var (
			rootfsFileBuffer *bytes.Buffer
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
			Expect(g.Pull()).To(Succeed())
		})

		It("calls the image puller with the expected args", func() {
			Expect(imagePuller.PullCallCount()).To(Equal(1))
			_, spec := imagePuller.PullArgsForCall(0)
			Expect(spec).To(Equal(imagepuller.ImageSpec{}))
		})
	})

	Describe("Pull failing", func() {
		var (
			pullErr error
		)

		JustBeforeEach(func() {
			pullErr = g.Pull()
		})

		Context("when image puller returns an error", func() {
			BeforeEach(func() {
				imagePuller.PullReturns(imagepuller.Image{}, errors.New("pull-failed"))
			})

			It("returns the error", func() {
				Expect(pullErr).To(MatchError(ContainSubstring("pull-failed")))
			})
		})
	})
})
