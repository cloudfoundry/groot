package groot_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/grootfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("Groot", func() {
	var (
		driver            *grootfakes.FakeDriver
		driverRuntimeSpec = specs.Spec{Version: "some-version"}

		layerIDGenerator *grootfakes.FakeLayerIDGenerator
		logger           *lagertest.TestLogger
		g                *groot.Groot

		rootfsFilePath string
	)

	BeforeEach(func() {
		tempFile, err := ioutil.TempFile("", "groot-unit-tests")
		Expect(err).NotTo(HaveOccurred())
		fmt.Fprint(tempFile, "afile")
		rootfsFilePath = tempFile.Name()
		Expect(tempFile.Close()).To(Succeed())

		driver = new(grootfakes.FakeDriver)
		driver.BundleReturns(driverRuntimeSpec, nil)

		layerIDGenerator = new(grootfakes.FakeLayerIDGenerator)
		layerIDGenerator.GenerateLayerIDReturns("checksum", nil)

		logger = lagertest.NewTestLogger("groot")
		g = &groot.Groot{Driver: driver, LayerIDGenerator: layerIDGenerator, Logger: logger}
	})

	AfterEach(func() {
		Expect(os.Remove(rootfsFilePath)).To(Succeed())
	})

	Describe("Create succeeding", func() {
		var (
			returnedRuntimeSpec specs.Spec
			rootfsFileBuffer    *bytes.Buffer
		)

		BeforeEach(func() {
			rootfsFileBuffer = bytes.NewBuffer([]byte{})
			driver.UnpackStub = func(logger lager.Logger, id, parentID string, layerTar io.Reader) error {
				_, err := io.Copy(rootfsFileBuffer, layerTar)
				Expect(err).NotTo(HaveOccurred())
				return nil
			}
		})

		JustBeforeEach(func() {
			var err error
			returnedRuntimeSpec, err = g.Create("some-handle", rootfsFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("generates a layer ID for the local rootfs", func() {
			Expect(layerIDGenerator.GenerateLayerIDCallCount()).To(Equal(1))
			Expect(layerIDGenerator.GenerateLayerIDArgsForCall(0)).To(Equal(rootfsFilePath))
		})

		It("calls driver.Unpack with the expected args", func() {
			Expect(driver.UnpackCallCount()).To(Equal(1))
			_, id, parentID, _ := driver.UnpackArgsForCall(0)
			Expect(id).To(Equal("checksum"))
			Expect(parentID).To(Equal(""))

			tarContents, err := ioutil.ReadAll(rootfsFileBuffer)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(tarContents)).To(Equal("afile"))
		})

		It("returns the runtime spec from driver.Bundle", func() {
			Expect(returnedRuntimeSpec).To(Equal(driverRuntimeSpec))
		})

		It("calls driver.Bundle with the expected args", func() {
			Expect(driver.BundleCallCount()).To(Equal(1))
			_, id, layerIDs := driver.BundleArgsForCall(0)
			Expect(id).To(Equal("some-handle"))
			Expect(layerIDs).To(Equal([]string{"checksum"}))
		})
	})

	Describe("Create failing", func() {
		var (
			rootfsURI string
			createErr error
		)

		BeforeEach(func() {
			rootfsURI = rootfsFilePath
		})

		JustBeforeEach(func() {
			_, createErr = g.Create("some-handle", rootfsURI)
		})

		Context("when the rootfsURI is not a file", func() {
			BeforeEach(func() {
				rootfsURI = "idontexist"
			})

			It("returns an error", func() {
				Expect(createErr).To(BeAssignableToTypeOf(&os.PathError{}))
			})
		})

		Context("when the layer ID generator returns an error", func() {
			BeforeEach(func() {
				layerIDGenerator.GenerateLayerIDReturns("", errors.New("generating-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError("generating-failed"))
			})
		})

		Context("when driver.Unpack returns an error", func() {
			BeforeEach(func() {
				driver.UnpackReturns(errors.New("unpack-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError("unpack-failed"))
			})
		})

		Context("when driver.Bundle returns an error", func() {
			BeforeEach(func() {
				driver.BundleReturns(specs.Spec{}, errors.New("bundle-failed"))
			})

			It("returns the error", func() {
				Expect(createErr).To(MatchError("bundle-failed"))
			})
		})
	})
})
