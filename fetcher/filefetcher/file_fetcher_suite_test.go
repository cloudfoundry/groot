package filefetcher_test

import (
	"io"
	"net/url"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTarFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "File Fetcher Suite")
}

func urlParse(rawURL string) *url.URL {
	parsed, err := url.Parse(rawURL)
	Expect(err).NotTo(HaveOccurred())
	return parsed
}

func readAll(r io.Reader) string {
	content, err := io.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())
	return string(content)
}
