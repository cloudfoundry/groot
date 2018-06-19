package source_test

import (
	"archive/tar"
	"fmt"
	"io"
	"net/url"
	"os"
	"testing"

	"github.com/containers/image/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Layer Fetcher Source Suite")
}

func tarEntries(tarFile io.Reader) []string {
	tr := tar.NewReader(tarFile)
	entries := []string{}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		Expect(err).NotTo(HaveOccurred())
		entries = append(entries, hdr.Name)
	}

	return entries
}

func urlParse(s string) *url.URL {
	u, err := url.Parse(s)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return u
}

func open(path string) *os.File {
	f, err := os.Open(path)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return f
}

func privateDockerAuthConfig() *types.DockerAuthConfig {
	return &types.DockerAuthConfig{
		Username: ensureEnv("REGISTRY_USERNAME"),
		Password: ensureEnv("REGISTRY_PASSWORD"),
	}
}

func ensureEnv(name string) string {
	value, exists := os.LookupEnv(name)
	Expect(exists).To(BeTrue(), fmt.Sprintf("expected env var %s to be set", name))
	return value
}
