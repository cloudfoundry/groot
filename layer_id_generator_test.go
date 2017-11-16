package groot_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/grootfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LocalLayerIDGenerator", func() {
	var (
		modTimer *grootfakes.FakeModTimer
		l        *groot.LocalLayerIDGenerator
	)

	BeforeEach(func() {
		modTimer = new(grootfakes.FakeModTimer)
		modTimer.ModTimeReturns(time.Date(2017, time.April, 1, 23, 0, 0, 0, time.UTC), nil)
		l = &groot.LocalLayerIDGenerator{ModTimer: modTimer}
	})

	Describe("successful layer generation", func() {
		var layerID string

		JustBeforeEach(func() {
			var err error
			layerID, err = l.GenerateLayerID("some-path")
			Expect(err).NotTo(HaveOccurred())
		})

		It("generates a layer ID equal to sha256(pathname-mtime)", func() {
			Expect(layerID).To(Equal("1a56a74279b5be2abb76887990636b0d9667f554ceb24c2c315d0e61be97529e"))
		})
	})

	Describe("unsuccessful layer generation", func() {
		Context("when the mod time cannot be determined", func() {
			It("returns an error", func() {
				modTimer.ModTimeReturns(time.Time{}, errors.New("modtime-error"))
				_, err := l.GenerateLayerID("some-path")
				Expect(err).To(MatchError("modtime-error"))
			})
		})
	})
})
