package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("stats", func() {
	var (
		rootfsURI string
		handle    = "some-handle"
	)

	BeforeEach(func() {
		tmpDir = tempDir("", "groot-integration-tests")
		rootfsURI = filepath.Join(tmpDir, "rootfs.tar")

		logLevel = ""
		env = []string{}
		stdout = new(bytes.Buffer)
		stderr = new(bytes.Buffer)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	runStatsCmd := func() error {
		footArgv := []string{"--driver-store", tmpDir, "stats", handle}
		footCmd := exec.Command(footBinPath, footArgv...)
		footCmd.Stdout = io.MultiWriter(stdout, GinkgoWriter)
		footCmd.Stderr = io.MultiWriter(stderr, GinkgoWriter)
		footCmd.Env = append(os.Environ(), env...)
		return footCmd.Run()
	}

	Describe("success", func() {
		It("calls driver.Stats() with expected args", func() {
			Expect(runStatsCmd()).To(Succeed())
			var statsArgs foot.StatsCalls
			readTestArgsFile(foot.StatsArgsFileName, &statsArgs)
			Expect(statsArgs[0].ID).To(Equal(handle))
		})

		It("returns the stats json on stdout", func() {
			Expect(runStatsCmd()).To(Succeed())
			var stats groot.VolumeStats
			Expect(json.Unmarshal(stdout.Bytes(), &stats)).To(Succeed())
			Expect(stats).To(Equal(foot.ReturnedVolumeStats))
		})
	})

	Describe("failure", func() {
		Context("when driver.Stats() returns an error", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_STATS_ERROR=true")
			})

			It("prints the error", func() {
				Expect(runStatsCmd()).NotTo(Succeed())
				Expect(stdout.String()).To(ContainSubstring("stats-err\n"))
			})
		})
	})
})
