// Code generated by counterfeiter. DO NOT EDIT.
package imagepullerfakes

import (
	"io"
	"sync"

	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager/v3"
)

type FakeVolumeDriver struct {
	UnpackStub        func(lager.Logger, string, []string, io.Reader) (int64, error)
	unpackMutex       sync.RWMutex
	unpackArgsForCall []struct {
		arg1 lager.Logger
		arg2 string
		arg3 []string
		arg4 io.Reader
	}
	unpackReturns struct {
		result1 int64
		result2 error
	}
	unpackReturnsOnCall map[int]struct {
		result1 int64
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeVolumeDriver) Unpack(arg1 lager.Logger, arg2 string, arg3 []string, arg4 io.Reader) (int64, error) {
	var arg3Copy []string
	if arg3 != nil {
		arg3Copy = make([]string, len(arg3))
		copy(arg3Copy, arg3)
	}
	fake.unpackMutex.Lock()
	ret, specificReturn := fake.unpackReturnsOnCall[len(fake.unpackArgsForCall)]
	fake.unpackArgsForCall = append(fake.unpackArgsForCall, struct {
		arg1 lager.Logger
		arg2 string
		arg3 []string
		arg4 io.Reader
	}{arg1, arg2, arg3Copy, arg4})
	stub := fake.UnpackStub
	fakeReturns := fake.unpackReturns
	fake.recordInvocation("Unpack", []interface{}{arg1, arg2, arg3Copy, arg4})
	fake.unpackMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeVolumeDriver) UnpackCallCount() int {
	fake.unpackMutex.RLock()
	defer fake.unpackMutex.RUnlock()
	return len(fake.unpackArgsForCall)
}

func (fake *FakeVolumeDriver) UnpackCalls(stub func(lager.Logger, string, []string, io.Reader) (int64, error)) {
	fake.unpackMutex.Lock()
	defer fake.unpackMutex.Unlock()
	fake.UnpackStub = stub
}

func (fake *FakeVolumeDriver) UnpackArgsForCall(i int) (lager.Logger, string, []string, io.Reader) {
	fake.unpackMutex.RLock()
	defer fake.unpackMutex.RUnlock()
	argsForCall := fake.unpackArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *FakeVolumeDriver) UnpackReturns(result1 int64, result2 error) {
	fake.unpackMutex.Lock()
	defer fake.unpackMutex.Unlock()
	fake.UnpackStub = nil
	fake.unpackReturns = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) UnpackReturnsOnCall(i int, result1 int64, result2 error) {
	fake.unpackMutex.Lock()
	defer fake.unpackMutex.Unlock()
	fake.UnpackStub = nil
	if fake.unpackReturnsOnCall == nil {
		fake.unpackReturnsOnCall = make(map[int]struct {
			result1 int64
			result2 error
		})
	}
	fake.unpackReturnsOnCall[i] = struct {
		result1 int64
		result2 error
	}{result1, result2}
}

func (fake *FakeVolumeDriver) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.unpackMutex.RLock()
	defer fake.unpackMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeVolumeDriver) recordInvocation(key string, args []interface{}) {
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

var _ imagepuller.VolumeDriver = new(FakeVolumeDriver)
