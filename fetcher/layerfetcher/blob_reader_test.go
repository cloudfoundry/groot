package layerfetcher_test

import (
	"os"
	"strings"

	"code.cloudfoundry.org/groot/fetcher/layerfetcher"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BlobReader", func() {
	var (
		blobReader       *layerfetcher.BlobReader
		blobFile         *os.File
		newBlobReaderErr error
	)

	BeforeEach(func() {
		blobFile = tempFile()
		defer blobFile.Close()
		writeString(blobFile, "hello-world")
	})

	AfterEach(func() {
		removeAllIfTemp(blobFile.Name(), blobReader)
	})

	JustBeforeEach(func() {
		blobReader, newBlobReaderErr = layerfetcher.NewBlobReader(blobFile.Name())
		Expect(newBlobReaderErr).NotTo(HaveOccurred())
	})

	Describe("Read", func() {
		It("reads the stream", func() {
			Expect(readAll(blobReader)).To(Equal("hello-world"))
		})
	})

	Context("when the blob doesn't exist", func() {
		Describe("NewBlobReader", func() {
			It("returns an error", func() {
				_, err := layerfetcher.NewBlobReader("not-a-real/file")
				Expect(err).To(MatchError(ContainSubstring("failed to open blob")))
			})
		})
	})

	Describe("Close", func() {
		It("deletes the source blob file", func() {
			Expect(blobFile.Name()).To(BeAnExistingFile())
			Expect(blobReader.Close()).To(Succeed())
			Expect(blobFile.Name()).ToNot(BeAnExistingFile())
		})
	})
})

func removeAllIfTemp(path string, blobReader *layerfetcher.BlobReader) {
	blobReader.Close()
	if !strings.HasPrefix(path, os.TempDir()) {
		Fail("attempt to delete non-temp file: " + path)
	}

	os.RemoveAll(path)
	Expect(path).NotTo(BeAnExistingFile())
}
