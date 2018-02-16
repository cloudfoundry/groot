package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	footBinPath          string
	driverStoreDir       string
	env                  []string
	configFilePath       string
	notFoundRuntimeError = map[string]string{
		"linux":   "no such file or directory",
		"windows": "The system cannot find the file specified.",
	}
)

var _ = SynchronizedBeforeSuite(func() []byte {
	binPath, err := gexec.Build("code.cloudfoundry.org/groot/integration/cmd/foot")
	Expect(err).NotTo(HaveOccurred())
	return []byte(binPath)
}, func(footBinPathBytes []byte) {
	footBinPath = string(footBinPathBytes)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func writeFile(path, content string) {
	Expect(ioutil.WriteFile(path, []byte(content), 0600)).To(Succeed())
}

func tempDir(dir, prefix string) string {
	name, err := ioutil.TempDir(dir, prefix)
	Expect(err).NotTo(HaveOccurred())
	return name
}

func readFile(filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	Expect(err).NotTo(HaveOccurred())
	return content
}

func unmarshalFile(filename string, data interface{}) {
	content := readFile(filename)
	Expect(json.Unmarshal(content, data)).To(Succeed())
}

func runFoot(configFilePath, driverStore string, args ...string) (string, error) {
	footCmd := exec.Command(footBinPath, "--config", configFilePath, "--driver-store", driverStore)
	footCmd.Args = append(footCmd.Args, args...)
	footCmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	footCmd.Stdout = io.MultiWriter(&stdout, GinkgoWriter)
	footCmd.Stderr = GinkgoWriter

	err := footCmd.Run()
	return stdout.String(), err
}
