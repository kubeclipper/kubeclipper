package restplus

import (
	"strconv"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

type testType string

func (obj testType) GetObjectKind() schema.ObjectKind { return schema.EmptyObjectKind }
func (obj testType) DeepCopyObject() runtime.Object   { return obj }

func TestNewMultiWatcher(t *testing.T) {
	w1 := watch.NewFake()
	w2 := watch.NewFake()
	m := NewMultiWatcher([]watch.Interface{w1, w2})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			w1.Add(testType("watcher1" + strconv.Itoa(i)))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			w2.Add(testType("watcher2" + strconv.Itoa(i)))
		}
	}()

	go func() {
		wg.Wait()
		m.Stop()
	}()

	resultChan := m.ResultChan()
	for event := range resultChan {
		t.Log("event:", event)
	}
}
