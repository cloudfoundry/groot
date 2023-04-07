package filefetcher_test

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/groot/fetcher/filefetcher"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager/v3/lagertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

var _ = Describe("File Fetcher", func() {
	var (
		fetcher *filefetcher.FileFetcher

		sourceImagePath string
		imagePath       string
		logger          *lagertest.TestLogger
		imageURL        *url.URL
	)

	BeforeEach(func() {
		sourceImagePath = tempDir()
		imagePath = filepath.Join(sourceImagePath, "a_file")
		imageURL = urlParse(imagePath)

		Expect(ioutil.WriteFile(path.Join(sourceImagePath, "a_file"), []byte("hello-world"), 0600)).To(Succeed())
		logger = lagertest.NewTestLogger("file-fetcher")
	})

	JustBeforeEach(func() {
		fetcher = filefetcher.NewFileFetcher(imageURL)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(imagePath)).To(Succeed())
		Expect(os.RemoveAll(sourceImagePath)).To(Succeed())
	})

	Describe("StreamBlob", func() {
		var (
			stream    io.ReadCloser
			streamErr error
		)

		JustBeforeEach(func() {
			stream, _, streamErr = fetcher.StreamBlob(logger, imagepuller.LayerInfo{})
		})

		AfterEach(func() {
			if stream != nil {
				Expect(stream.Close()).To(Succeed())
			}
		})

		It("returns the contents of the source file", func() {
			Expect(readAll(stream)).To(Equal("hello-world"))
		})

		Context("when the source is a directory", func() {
			var tmpDir string

			BeforeEach(func() {
				tmpDir = tempDir()
				imageURL = urlParse(tmpDir)
			})

			AfterEach(func() {
				Expect(os.RemoveAll(tmpDir)).To(Succeed())
			})

			It("returns an error message", func() {
				Expect(streamErr).To(MatchError(ContainSubstring("invalid base image: directory provided instead of a tar file")))
			})
		})

		Context("when the source does not exist", func() {
			BeforeEach(func() {
				imageURL = urlParse("/nothing/here")
			})

			It("returns an error", func() {
				Expect(streamErr).To(MatchError(ContainSubstring("local image not found in `/nothing/here`")))
			})
		})
	})

	Describe("LayersDigest", func() {
		var (
			imageInfo imagepuller.ImageInfo
			infoErr   error
		)

		JustBeforeEach(func() {
			imageInfo, infoErr = fetcher.ImageInfo(logger)
		})

		It("does not return an error", func() {
			Expect(infoErr).NotTo(HaveOccurred())
		})

		It("returns the correct image", func() {
			layers := imageInfo.LayerInfos

			Expect(len(layers)).To(Equal(1))
			Expect(strings.EqualFold(layers[0].BlobID, imagePath)).To(BeTrue())
			Expect(layers[0].ChainID).NotTo(BeEmpty())
			Expect(layers[0].ParentChainID).To(BeEmpty())
			Expect(layers[0].Size).To(Equal(int64(len("hello-world"))))

			Expect(imageInfo.Config).To(Equal(v1.Image{}))
		})

		Context("when image timestamp changes", func() {
			JustBeforeEach(func() {
				Expect(os.Chtimes(imagePath, time.Now().Add(time.Hour), time.Now().Add(time.Hour))).To(Succeed())
			})

			It("generates another chain id", func() {
				newImageInfo, err := fetcher.ImageInfo(logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(imageInfo.LayerInfos[0].ChainID).NotTo(Equal(newImageInfo.LayerInfos[0].ChainID))
			})
		})

		Context("when the image doesn't exist", func() {
			BeforeEach(func() {
				imageURL = urlParse("/not-here")
			})

			It("returns an error", func() {
				Expect(infoErr).To(MatchError(ContainSubstring("fetching image timestamp")))
			})
		})
	})
})

func tempDir() string {
	dir, err := ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())
	return dir
}

type tarEntry struct {
	header   *tar.Header
	contents []byte
}

func streamTar(r *tar.Reader) []tarEntry {
	l := []tarEntry{}
	for {
		header, err := r.Next()
		if err != nil {
			Expect(err).To(Equal(io.EOF))
			return l
		}

		contents := make([]byte, header.Size)
		_, _ = r.Read(contents)
		l = append(l, tarEntry{
			header:   header,
			contents: contents,
		})
	}
}
