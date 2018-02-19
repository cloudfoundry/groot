package filefetcher_test

import (
	"net/url"
	"testing"

	. "github.com/onsi/ginkgo"
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
