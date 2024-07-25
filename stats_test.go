package groot_test

import (
	"errors"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/grootfakes"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stats", func() {
	var (
		driver *grootfakes.FakeDriver

		logger *lagertest.TestLogger
		g      *groot.Groot

		expectedStats groot.VolumeStats
	)

	BeforeEach(func() {
		driver = new(grootfakes.FakeDriver)

		logger = lagertest.NewTestLogger("groot")
		g = &groot.Groot{
			Driver: driver,
			Logger: logger,
		}

		expectedStats = groot.VolumeStats{
			DiskUsage: groot.DiskUsage{
				TotalBytesUsed:     1234,
				ExclusiveBytesUsed: 12,
			},
		}
		driver.StatsReturnsOnCall(0, expectedStats, nil)
	})

	It("calls driver.Stats() with the expected args", func() {
		stats, err := g.Stats("image")
		Expect(err).NotTo(HaveOccurred())
		Expect(stats).To(Equal(expectedStats))

		Expect(driver.StatsCallCount()).To(Equal(1))
		_, bundleID := driver.StatsArgsForCall(0)
		Expect(bundleID).To(Equal("image"))
	})

	Context("when driver fails to get the stats", func() {
		BeforeEach(func() {
			driver.StatsReturnsOnCall(0, groot.VolumeStats{}, errors.New("failed"))
		})

		It("returns the error", func() {
			_, err := g.Stats("image")
			Expect(err).To(MatchError("failed"))
		})
	})
})
