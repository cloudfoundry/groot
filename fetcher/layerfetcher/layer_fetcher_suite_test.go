package layerfetcher_test

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLayerFetcher(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "LayerFetcher Suite")
}

func readAll(reader io.Reader) string {
	content, err := io.ReadAll(reader)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

func tempFile() *os.File {
	file, err := os.CreateTemp("", "")
	Expect(err).NotTo(HaveOccurred())
	return file
}

func writeString(writer io.Writer, contents string) {
	size, err := io.WriteString(writer, contents)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(contents)).To(Equal(size))
}
