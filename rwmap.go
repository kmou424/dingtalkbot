package dingtalkbot

import (
	"encoding/json"
	"fmt"
	"sync"
)

type RWMap[T comparable, R any] struct {
	mutex *sync.RWMutex
	data  map[T]R
}

func NewRWMap[T comparable, R any]() *RWMap[T, R] {
	return &RWMap[T, R]{
		mutex: &sync.RWMutex{},
		data:  make(map[T]R),
	}
}

func NewRWValueMap[T comparable]() *RWMap[T, *Value] {
	return NewRWMap[T, *Value]()
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &rw.data)
}

//goland:noinspection GoMixedReceiverTypes
func (rw RWMap[T, R]) MarshalJSON() ([]byte, error) {
	return json.Marshal(rw.data)
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) Put(key T, value R) {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()
	rw.data[key] = value
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) Get(key T) (R, bool) {
	rw.mutex.RLock()
	defer rw.mutex.RUnlock()
	value, ok := rw.data[key]
	return value, ok
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) MustGet(key T) R {
	value, ok := rw.Get(key)
	if !ok {
		panic(fmt.Sprintf(`can't get %v from RWMap`, key))
	}
	return value
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) Delete(key T) {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()
	delete(rw.data, key)
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) Size() int {
	rw.mutex.RLock()
	defer rw.mutex.RUnlock()
	return len(rw.data)
}

//goland:noinspection GoMixedReceiverTypes
func (rw *RWMap[T, R]) Each(f func(T, R) bool) {
	rw.mutex.RLock()
	defer rw.mutex.RUnlock()
	for k, v := range rw.data {
		if !f(k, v) {
			return
		}
	}
}
