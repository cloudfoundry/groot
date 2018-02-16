package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("stats", func() {
	BeforeEach(func() {
		driverStoreDir = tempDir("", "groot-integration-tests")
		env = []string{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
	})

	Describe("success", func() {
		It("calls driver.Stats() with expected args", func() {
			_, err := runFoot("", driverStoreDir, "stats", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			var statsArgs foot.StatsCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.StatsArgsFileName), &statsArgs)
			Expect(statsArgs[0].ID).To(Equal("some-handle"))
		})

		It("returns the stats json on stdout", func() {
			stdout, err := runFoot("", driverStoreDir, "stats", "some-handle")
			Expect(err).NotTo(HaveOccurred())

			var stats groot.VolumeStats
			Expect(json.Unmarshal([]byte(stdout), &stats)).To(Succeed())
			Expect(stats).To(Equal(foot.ReturnedVolumeStats))
		})
	})

	Describe("failure", func() {
		Context("when driver.Stats() returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_STATS_ERROR=true")
			})

			It("prints the error", func() {
				stdout, err := runFoot("", driverStoreDir, "stats", "some-handle")
				Expect(err).To(HaveOccurred())
				Expect(stdout).To(ContainSubstring("stats-err"))
			})
		})
	})
})
