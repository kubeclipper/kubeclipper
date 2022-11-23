package restplus

import (
	"sync"

	"k8s.io/apimachinery/pkg/watch"
)

type multiWatcher struct {
	watchers []watch.Interface
	stop     chan struct{}

	ch      chan watch.Event
	stopped bool
	sync.Mutex
}

func NewMultiWatcher(watchers []watch.Interface) watch.Interface {
	m := &multiWatcher{
		watchers: watchers,
		ch:       make(chan watch.Event),
		stop:     make(chan struct{}),
	}

	// merge watchers
	var wg sync.WaitGroup
	wg.Add(len(m.watchers))

	for _, watcher := range m.watchers {
		go func(w watch.Interface) {
			defer wg.Done()
			events := w.ResultChan()
			for event := range events {
				m.ch <- event
			}
		}(watcher)
	}
	go func() {
		wg.Wait()
		// when all watcher closed,send a signal to stop chan
		m.stop <- struct{}{}
	}()
	return m
}

func (m *multiWatcher) Stop() {
	m.Lock()
	defer m.Unlock()

	if m.stopped {
		return
	}

	m.stopped = true
	for _, watcher := range m.watchers {
		watcher.Stop()
	}

	// waiting for all watcher closed
	<-m.stop
	close(m.ch)
}

func (m *multiWatcher) ResultChan() <-chan watch.Event {
	return m.ch
}
