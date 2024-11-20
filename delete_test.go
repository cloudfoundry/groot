package groot_test

import (
	"errors"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/grootfakes"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete", func() {
	var (
		driver *grootfakes.FakeDriver

		logger *lagertest.TestLogger
		g      *groot.Groot
	)

	BeforeEach(func() {
		driver = new(grootfakes.FakeDriver)

		logger = lagertest.NewTestLogger("groot")
		g = &groot.Groot{
			Driver: driver,
			Logger: logger,
		}
	})

	It("calls driver.Delete() with the expected args", func() {
		Expect(g.Delete("image")).To(Succeed())

		Expect(driver.DeleteCallCount()).To(Equal(1))
		_, bundleID := driver.DeleteArgsForCall(0)
		Expect(bundleID).To(Equal("image"))
	})

	Context("when driver fails to delete", func() {
		BeforeEach(func() {
			driver.DeleteReturns(errors.New("failed"))
		})

		It("returns the error", func() {
			Expect(g.Delete("image")).To(MatchError(ContainSubstring("failed")))
		})
	})
})
