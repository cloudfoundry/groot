package integration_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"code.cloudfoundry.org/groot/integration/cmd/foot/foot"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	footBinPath          string
	tempDir              string
	env                  []string
	logLevel             string
	configFilePath       string
	stdout               *bytes.Buffer
	stderr               *bytes.Buffer
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

func readTestArgsFile(filename string, ptr interface{}) {
	content, err := ioutil.ReadFile(argFilePath(filename))
	Expect(err).NotTo(HaveOccurred())
	Expect(json.Unmarshal(content, ptr)).To(Succeed())
}

func argFilePath(filename string) string {
	return filepath.Join(tempDir, filename)
}

func unpackIsSuccessful(runFootCmd func() error) {
	It("calls driver.Unpack() with the expected args", func() {
		var args foot.UnpackCalls
		readTestArgsFile(foot.UnpackArgsFileName, &args)
		Expect(args[0].ID).NotTo(BeEmpty())
		Expect(args[0].ParentIDs).To(BeEmpty())
	})

	Describe("subsequent invocations", func() {
		Context("when the rootfs file has not changed", func() {
			It("gernerates the same layer ID", func() {
				var unpackArgs foot.UnpackCalls
				readTestArgsFile(foot.UnpackArgsFileName, &unpackArgs)
				firstInvocationLayerID := unpackArgs[0].ID

				Expect(runFootCmd()).To(Succeed())

				readTestArgsFile(foot.UnpackArgsFileName, &unpackArgs)
				secondInvocationLayerID := unpackArgs[0].ID

				Expect(secondInvocationLayerID).To(Equal(firstInvocationLayerID))
			})
		})
	})

	Describe("layer caching", func() {
		It("calls exists", func() {
			var existsArgs foot.ExistsCalls
			readTestArgsFile(foot.ExistsArgsFileName, &existsArgs)
			Expect(existsArgs[0].LayerID).ToNot(BeEmpty())
		})

		Context("when the layer is not cached", func() {
			It("calls driver.Unpack() with the layerID", func() {
				var existsArgs foot.ExistsCalls
				readTestArgsFile(foot.ExistsArgsFileName, &existsArgs)
				Expect(existsArgs[0].LayerID).ToNot(BeEmpty())

				Expect(argFilePath(foot.UnpackArgsFileName)).To(BeAnExistingFile())

				var unpackArgs foot.UnpackCalls
				readTestArgsFile(foot.UnpackArgsFileName, &unpackArgs)
				Expect(len(unpackArgs)).To(Equal(len(existsArgs)))

				lastCall := len(unpackArgs) - 1
				for i := range unpackArgs {
					Expect(unpackArgs[i].ID).To(Equal(existsArgs[lastCall-i].LayerID))
				}
			})
		})

		Context("when the layer is cached", func() {
			BeforeEach(func() {
				env = append(env, "FOOT_LAYER_EXISTS=true")
			})

			It("doesn't call driver.Unpack()", func() {
				Expect(argFilePath(foot.UnpackArgsFileName)).ToNot(BeAnExistingFile())
			})
		})
	})
}

func whenUnpackIsUnsuccessful(runFootCmd func() error) {
	var writeConfigFile bool

	BeforeEach(func() {
		writeConfigFile = true
	})

	JustBeforeEach(func() {
		if writeConfigFile {
			writeFile(configFilePath, "log_level: "+logLevel)
		}

		exitErr := runFootCmd()
		Expect(exitErr).To(HaveOccurred())
		Expect(exitErr.(*exec.ExitError).Sys().(syscall.WaitStatus).ExitStatus()).To(Equal(1))
	})

	Context("when driver.Unpack() returns an error", func() {
		BeforeEach(func() {
			env = append(env, "FOOT_UNPACK_ERROR=true")
		})

		It("prints the error", func() {
			Expect(stdout.String()).To(ContainSubstring("unpack-err\n"))
		})
	})

	Context("when the config file path is not an existing file", func() {
		BeforeEach(func() {
			writeConfigFile = false
		})

		It("prints an error", func() {
			Expect(stdout.String()).To(ContainSubstring(notFoundRuntimeError[runtime.GOOS]))
		})
	})

	Context("when the config file is invalid yaml", func() {
		BeforeEach(func() {
			writeConfigFile = false
			writeFile(configFilePath, "%haha")
		})

		It("prints an error", func() {
			Expect(stdout.String()).To(ContainSubstring("yaml"))
		})
	})

	Context("when the specified log level is invalid", func() {
		BeforeEach(func() {
			logLevel = "lol"
		})

		It("prints an error", func() {
			Expect(stdout.String()).To(ContainSubstring("lol"))
		})
	})
}

func writeFile(path, content string) {
	Expect(ioutil.WriteFile(path, []byte(content), 0600)).To(Succeed())
}
