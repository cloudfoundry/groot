package layerfetcher_test

import (
	"io"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLayerFetcher(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "LayerFetcher Suite")
}

func readAll(reader io.Reader) string {
	content, err := ioutil.ReadAll(reader)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

func tempFile() *os.File {
	file, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	return file
}

func writeString(writer io.Writer, contents string) {
	size, err := io.WriteString(writer, contents)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(contents)).To(Equal(size))
}
