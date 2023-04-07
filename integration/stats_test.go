package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("stats", func() {
	var (
		footCmd        *exec.Cmd
		driverStoreDir string
	)

	BeforeEach(func() {
		driverStoreDir = tempDir("", "groot-integration-tests")
		footCmd = newFootCommand("", driverStoreDir, "stats", "some-handle")
	})

	JustBeforeEach(func() {
		var out []byte
		out, footCmdError = footCmd.Output()
		footCmdOutput = gbytes.BufferWithBytes(out)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(driverStoreDir)).To(Succeed())
	})

	Describe("success", func() {
		It("calls driver.Stats() with expected args", func() {
			Expect(footCmdError).NotTo(HaveOccurred())

			var statsArgs foot.StatsCalls
			unmarshalFile(filepath.Join(driverStoreDir, foot.StatsArgsFileName), &statsArgs)
			Expect(statsArgs[0].ID).To(Equal("some-handle"))
		})

		It("returns the stats json on stdout", func() {
			Expect(footCmdError).NotTo(HaveOccurred())

			var stats groot.VolumeStats
			Expect(json.Unmarshal(footCmdOutput.Contents(), &stats)).To(Succeed())
			Expect(stats).To(Equal(foot.ReturnedVolumeStats))
		})
	})

	Describe("failure", func() {
		Context("when driver.Stats() returns an error", func() {
			BeforeEach(func() {
				footCmd.Env = append(os.Environ(), "FOOT_STATS_ERROR=true")
			})

			It("prints the error", func() {
				expectErrorOutput("stats-err")
			})
		})

		Context("when the incorrect number of args is given", func() {
			BeforeEach(func() {
				footCmd = newFootCommand("", driverStoreDir, "stats")
			})

			It("prints an error", func() {
				Expect(footCmdError).To(HaveOccurred())
				Expect(footCmdOutput).To(gbytes.Say("Incorrect number of args. Expect 1, got 0"))
			})
		})
	})
})
