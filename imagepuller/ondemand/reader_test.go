package ondemand_test

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "code.cloudfoundry.org/groot/imagepuller/ondemand"
)

var _ = Describe("Reader", func() {
	It("runs the create function on first read", func() {
		count := 0
		r := &Reader{
			Create: func() (io.ReadCloser, error) {
				count++
				return ioutil.NopCloser(bytes.NewBuffer([]byte{})), nil
			},
		}

		r.Read([]byte{})
		Expect(count).To(Equal(1))
	})

	It("does not run the create function on subsequent reads", func() {
		count := 0
		r := &Reader{
			Create: func() (io.ReadCloser, error) {
				count++
				return ioutil.NopCloser(bytes.NewBuffer([]byte{})), nil
			},
		}

		r.Read([]byte{})
		r.Read([]byte{})
		Expect(count).To(Equal(1))
	})

	It("reads from the created ReadCloser", func() {
		r := &Reader{
			Create: func() (io.ReadCloser, error) {
				return ioutil.NopCloser(bytes.NewBuffer([]byte("cake"))), nil
			},
		}

		buf := make([]byte, 4)
		_, err := r.Read(buf)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(buf)).To(Equal("cake"))
	})

	It("closes the created ReadCloser", func() {
		fakeReadCloser := &FakeReadCloser{}
		r := &Reader{
			Create: func() (io.ReadCloser, error) {
				return fakeReadCloser, nil
			},
		}

		r.Read([]byte{})
		Expect(r.Close()).To(Succeed())
		Expect(fakeReadCloser.closeCount).To(Equal(1))
	})

	It("doesn't error if close is called before read", func() {
		fakeReadCloser := &FakeReadCloser{}
		r := &Reader{
			Create: func() (io.ReadCloser, error) {
				return fakeReadCloser, nil
			},
		}

		Expect(r.Close()).To(Succeed())
	})

	Context("when the create function errors", func() {
		It("bubbles the error on read", func() {
			r := &Reader{
				Create: func() (io.ReadCloser, error) {
					return nil, errors.New("EXPLODE")
				},
			}

			_, err := r.Read([]byte{})
			Expect(err).To(MatchError("EXPLODE"))
		})
	})

})

type FakeReadCloser struct {
	closeCount int
}

func (f *FakeReadCloser) Read(p []byte) (int, error) {
	return 0, nil
}

func (f *FakeReadCloser) Close() error {
	f.closeCount++
	return nil
}
