package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("stats", func() {
	var (
		footCmd *exec.Cmd
		session *gexec.Session
	)

	BeforeEach(func() {
		driverStoreDir = tempDir("", "groot-integration-tests")
		footCmd = newFootCommand("", driverStoreDir, "stats", "some-handle")
	})

	JustBeforeEach(func() {
		session = gexecStart(footCmd).Wait()
	})

	AfterEach(func() {
		Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
	})

	Describe("success", func() {
		It("calls driver.Stats() with expected args", func() {
			Expect(session).To(gexec.Exit(0))

			var statsArgs foot.StatsCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.StatsArgsFileName), &statsArgs)
			Expect(statsArgs[0].ID).To(Equal("some-handle"))
		})

		It("returns the stats json on stdout", func() {
			Expect(session).To(gexec.Exit(0))

			var stats groot.VolumeStats
			Expect(json.Unmarshal(session.Out.Contents(), &stats)).To(Succeed())
			Expect(stats).To(Equal(foot.ReturnedVolumeStats))
		})
	})

	Describe("failure", func() {
		Context("when driver.Stats() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_STATS_ERROR=true")
			})

			It("prints the error", func() {
				Expect(session).NotTo(gexec.Exit(0))
				Expect(session).To(gbytes.Say("stats-err"))
			})
		})
	})
})
