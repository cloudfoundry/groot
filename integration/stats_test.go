package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("stats", func() {
	BeforeEach(func() {
		tmpDir = tempDir("", "groot-integration-tests")
		env = []string{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Describe("success", func() {
		It("calls driver.Stats() with expected args", func() {
			Expect(runFoot("", tmpDir, "stats", "some-handle")).To(gexec.Exit(0))

			var statsArgs foot.StatsCalls
			unmarshalFile(filepath.Join(tmpDir, foot.StatsArgsFileName), &statsArgs)
			Expect(statsArgs[0].ID).To(Equal("some-handle"))
		})

		It("returns the stats json on stdout", func() {
			session := runFoot("", tmpDir, "stats", "some-handle")
			Expect(session).To(gexec.Exit(0))

			var stats groot.VolumeStats
			Expect(json.Unmarshal(session.Out.Contents(), &stats)).To(Succeed())
			Expect(stats).To(Equal(foot.ReturnedVolumeStats))
		})
	})

	Describe("failure", func() {
		Context("when driver.Stats() returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_STATS_ERROR=true")
			})

			It("prints the error", func() {
				session := runFoot("", tmpDir, "stats", "some-handle")
				Expect(session).To(gexec.Exit(1))
				Expect(session.Out).To(gbytes.Say("stats-err"))
			})
		})
	})
})
