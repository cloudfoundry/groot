package layerfetcher // import "code.cloudfoundry.org/groot/fetcher/layerfetcher"

import (
	"fmt"
	"io"
	"os"
)

type BlobReader struct {
	reader   io.ReadCloser
	filePath string
}

func NewBlobReader(blobPath string) (*BlobReader, error) {
	reader, err := os.Open(blobPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open blob: %w", err)
	}

	return &BlobReader{
		filePath: blobPath,
		reader:   reader,
	}, nil
}

func (d *BlobReader) Read(p []byte) (int, error) {
	return d.reader.Read(p)
}

func (d *BlobReader) Close() error {
	// #nosec G104 - ignore the Close() error here because we prefer to know if we could delete the file, and have no other logging options in the code
	d.reader.Close()
	return os.Remove(d.filePath)
}
