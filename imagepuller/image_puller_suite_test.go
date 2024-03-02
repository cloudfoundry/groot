package imagepuller_test

import (
	"compress/gzip"
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImagePuller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Imagepuller suite")
}

func tempDir(dir, prefix string) string {
	path, err := os.MkdirTemp(dir, prefix)
	Expect(err).NotTo(HaveOccurred())
	return path
}

func writeString(w io.Writer, s string) {
	n, err := io.WriteString(w, s)
	Expect(err).NotTo(HaveOccurred())
	Expect(n).To(Equal(len(s)))
}

func readAll(r io.Reader) string {
	content, err := io.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

func gzipNewReader(r io.Reader) *gzip.Reader {
	g, err := gzip.NewReader(r)
	Expect(err).NotTo(HaveOccurred())
	return g
}
