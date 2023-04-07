package imagepuller_test

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/url"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImagePuller(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Imagepuller suite")
}

func tempDir(dir, prefix string) string {
	path, err := ioutil.TempDir(dir, prefix)
	Expect(err).NotTo(HaveOccurred())
	return path
}

func urlParse(rawURL string) *url.URL {
	parsed, err := url.Parse(rawURL)
	Expect(err).NotTo(HaveOccurred())
	return parsed
}

func writeString(w io.Writer, s string) {
	n, err := io.WriteString(w, s)
	Expect(err).NotTo(HaveOccurred())
	Expect(n).To(Equal(len(s)))
}

func readAll(r io.Reader) string {
	content, err := ioutil.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}

func gzipNewReader(r io.Reader) *gzip.Reader {
	g, err := gzip.NewReader(r)
	Expect(err).NotTo(HaveOccurred())
	return g
}
