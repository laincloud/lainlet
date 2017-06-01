package mocks

import (
	"github.com/laincloud/lainlet/store"
	"testing"
)

func TestPut(t *testing.T) {
	testObj := new(Store)
	testObj.On("Put", "key", []byte("value")).Return(nil)
	testObj.Put("key", []byte("value"))
	testObj.AssertExpectations(t)
}

func TestGet(t *testing.T) {
	testObj := new(Store)
	testObj.On("Get", "key").Return(&store.KVPair{
		Key:       "key",
		Value:     []byte("value"),
		LastIndex: 1,
	}, nil)
	testObj.Get("key")
	testObj.AssertExpectations(t)
}

func TestExists(t *testing.T) {
	testObj := new(Store)
	testObj.On("Exists", "key").Return(true, nil)
	testObj.Exists("key")
	testObj.AssertExpectations(t)
}
