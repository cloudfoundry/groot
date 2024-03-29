// Code generated by counterfeiter. DO NOT EDIT.
package grootfakes

import (
	"sync"

	"code.cloudfoundry.org/groot"
	"code.cloudfoundry.org/groot/imagepuller"
	"code.cloudfoundry.org/lager/v3"
)

type FakeImagePuller struct {
	PullStub        func(lager.Logger, imagepuller.ImageSpec) (imagepuller.Image, error)
	pullMutex       sync.RWMutex
	pullArgsForCall []struct {
		arg1 lager.Logger
		arg2 imagepuller.ImageSpec
	}
	pullReturns struct {
		result1 imagepuller.Image
		result2 error
	}
	pullReturnsOnCall map[int]struct {
		result1 imagepuller.Image
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeImagePuller) Pull(arg1 lager.Logger, arg2 imagepuller.ImageSpec) (imagepuller.Image, error) {
	fake.pullMutex.Lock()
	ret, specificReturn := fake.pullReturnsOnCall[len(fake.pullArgsForCall)]
	fake.pullArgsForCall = append(fake.pullArgsForCall, struct {
		arg1 lager.Logger
		arg2 imagepuller.ImageSpec
	}{arg1, arg2})
	stub := fake.PullStub
	fakeReturns := fake.pullReturns
	fake.recordInvocation("Pull", []interface{}{arg1, arg2})
	fake.pullMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeImagePuller) PullCallCount() int {
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	return len(fake.pullArgsForCall)
}

func (fake *FakeImagePuller) PullCalls(stub func(lager.Logger, imagepuller.ImageSpec) (imagepuller.Image, error)) {
	fake.pullMutex.Lock()
	defer fake.pullMutex.Unlock()
	fake.PullStub = stub
}

func (fake *FakeImagePuller) PullArgsForCall(i int) (lager.Logger, imagepuller.ImageSpec) {
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	argsForCall := fake.pullArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeImagePuller) PullReturns(result1 imagepuller.Image, result2 error) {
	fake.pullMutex.Lock()
	defer fake.pullMutex.Unlock()
	fake.PullStub = nil
	fake.pullReturns = struct {
		result1 imagepuller.Image
		result2 error
	}{result1, result2}
}

func (fake *FakeImagePuller) PullReturnsOnCall(i int, result1 imagepuller.Image, result2 error) {
	fake.pullMutex.Lock()
	defer fake.pullMutex.Unlock()
	fake.PullStub = nil
	if fake.pullReturnsOnCall == nil {
		fake.pullReturnsOnCall = make(map[int]struct {
			result1 imagepuller.Image
			result2 error
		})
	}
	fake.pullReturnsOnCall[i] = struct {
		result1 imagepuller.Image
		result2 error
	}{result1, result2}
}

func (fake *FakeImagePuller) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.pullMutex.RLock()
	defer fake.pullMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeImagePuller) recordInvocation(key string, args []interface{}) {
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

var _ groot.ImagePuller = new(FakeImagePuller)
