// Code generated by counterfeiter. DO NOT EDIT.
package grootfakes

import (
	"io"
	"sync"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type FakeDriver struct {
	UnpackStub        func(logger lager.Logger, layerId, parentID string, layerTar io.Reader) error
	unpackMutex       sync.RWMutex
	unpackArgsForCall []struct {
		logger   lager.Logger
		layerId  string
		parentID string
		layerTar io.Reader
	}
	unpackReturns struct {
		result1 error
	}
	unpackReturnsOnCall map[int]struct {
		result1 error
	}
	BundleStub        func(logger lager.Logger, bundleId string, layerIDs []string) (specs.Spec, error)
	bundleMutex       sync.RWMutex
	bundleArgsForCall []struct {
		logger   lager.Logger
		bundleId string
		layerIDs []string
	}
	bundleReturns struct {
		result1 specs.Spec
		result2 error
	}
	bundleReturnsOnCall map[int]struct {
		result1 specs.Spec
		result2 error
	}
	ExistsStub        func(logger lager.Logger, layerId string) bool
	existsMutex       sync.RWMutex
	existsArgsForCall []struct {
		logger  lager.Logger
		layerId string
	}
	existsReturns struct {
		result1 bool
	}
	existsReturnsOnCall map[int]struct {
		result1 bool
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeDriver) Unpack(logger lager.Logger, layerId string, parentID string, layerTar io.Reader) error {
	fake.unpackMutex.Lock()
	ret, specificReturn := fake.unpackReturnsOnCall[len(fake.unpackArgsForCall)]
	fake.unpackArgsForCall = append(fake.unpackArgsForCall, struct {
		logger   lager.Logger
		layerId  string
		parentID string
		layerTar io.Reader
	}{logger, layerId, parentID, layerTar})
	fake.recordInvocation("Unpack", []interface{}{logger, layerId, parentID, layerTar})
	fake.unpackMutex.Unlock()
	if fake.UnpackStub != nil {
		return fake.UnpackStub(logger, layerId, parentID, layerTar)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.unpackReturns.result1
}

func (fake *FakeDriver) UnpackCallCount() int {
	fake.unpackMutex.RLock()
	defer fake.unpackMutex.RUnlock()
	return len(fake.unpackArgsForCall)
}

func (fake *FakeDriver) UnpackArgsForCall(i int) (lager.Logger, string, string, io.Reader) {
	fake.unpackMutex.RLock()
	defer fake.unpackMutex.RUnlock()
	return fake.unpackArgsForCall[i].logger, fake.unpackArgsForCall[i].layerId, fake.unpackArgsForCall[i].parentID, fake.unpackArgsForCall[i].layerTar
}

func (fake *FakeDriver) UnpackReturns(result1 error) {
	fake.UnpackStub = nil
	fake.unpackReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeDriver) UnpackReturnsOnCall(i int, result1 error) {
	fake.UnpackStub = nil
	if fake.unpackReturnsOnCall == nil {
		fake.unpackReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.unpackReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeDriver) Bundle(logger lager.Logger, bundleId string, layerIDs []string) (specs.Spec, error) {
	var layerIDsCopy []string
	if layerIDs != nil {
		layerIDsCopy = make([]string, len(layerIDs))
		copy(layerIDsCopy, layerIDs)
	}
	fake.bundleMutex.Lock()
	ret, specificReturn := fake.bundleReturnsOnCall[len(fake.bundleArgsForCall)]
	fake.bundleArgsForCall = append(fake.bundleArgsForCall, struct {
		logger   lager.Logger
		bundleId string
		layerIDs []string
	}{logger, bundleId, layerIDsCopy})
	fake.recordInvocation("Bundle", []interface{}{logger, bundleId, layerIDsCopy})
	fake.bundleMutex.Unlock()
	if fake.BundleStub != nil {
		return fake.BundleStub(logger, bundleId, layerIDs)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.bundleReturns.result1, fake.bundleReturns.result2
}

func (fake *FakeDriver) BundleCallCount() int {
	fake.bundleMutex.RLock()
	defer fake.bundleMutex.RUnlock()
	return len(fake.bundleArgsForCall)
}

func (fake *FakeDriver) BundleArgsForCall(i int) (lager.Logger, string, []string) {
	fake.bundleMutex.RLock()
	defer fake.bundleMutex.RUnlock()
	return fake.bundleArgsForCall[i].logger, fake.bundleArgsForCall[i].bundleId, fake.bundleArgsForCall[i].layerIDs
}

func (fake *FakeDriver) BundleReturns(result1 specs.Spec, result2 error) {
	fake.BundleStub = nil
	fake.bundleReturns = struct {
		result1 specs.Spec
		result2 error
	}{result1, result2}
}

func (fake *FakeDriver) BundleReturnsOnCall(i int, result1 specs.Spec, result2 error) {
	fake.BundleStub = nil
	if fake.bundleReturnsOnCall == nil {
		fake.bundleReturnsOnCall = make(map[int]struct {
			result1 specs.Spec
			result2 error
		})
	}
	fake.bundleReturnsOnCall[i] = struct {
		result1 specs.Spec
		result2 error
	}{result1, result2}
}

func (fake *FakeDriver) Exists(logger lager.Logger, layerId string) bool {
	fake.existsMutex.Lock()
	ret, specificReturn := fake.existsReturnsOnCall[len(fake.existsArgsForCall)]
	fake.existsArgsForCall = append(fake.existsArgsForCall, struct {
		logger  lager.Logger
		layerId string
	}{logger, layerId})
	fake.recordInvocation("Exists", []interface{}{logger, layerId})
	fake.existsMutex.Unlock()
	if fake.ExistsStub != nil {
		return fake.ExistsStub(logger, layerId)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.existsReturns.result1
}

func (fake *FakeDriver) ExistsCallCount() int {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return len(fake.existsArgsForCall)
}

func (fake *FakeDriver) ExistsArgsForCall(i int) (lager.Logger, string) {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return fake.existsArgsForCall[i].logger, fake.existsArgsForCall[i].layerId
}

func (fake *FakeDriver) ExistsReturns(result1 bool) {
	fake.ExistsStub = nil
	fake.existsReturns = struct {
		result1 bool
	}{result1}
}

func (fake *FakeDriver) ExistsReturnsOnCall(i int, result1 bool) {
	fake.ExistsStub = nil
	if fake.existsReturnsOnCall == nil {
		fake.existsReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.existsReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *FakeDriver) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.unpackMutex.RLock()
	defer fake.unpackMutex.RUnlock()
	fake.bundleMutex.RLock()
	defer fake.bundleMutex.RUnlock()
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeDriver) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ groot.Driver = new(FakeDriver)