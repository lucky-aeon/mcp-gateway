package xlog

import "sync"

type asyncSinkDispatcher struct {
	mu    sync.RWMutex
	sinks []Sink
	ch    chan Entry
}

func newAsyncSinkDispatcher(buffer int) *asyncSinkDispatcher {
	d := &asyncSinkDispatcher{
		sinks: make([]Sink, 0),
		ch:    make(chan Entry, buffer),
	}
	go d.run()
	return d
}

func (d *asyncSinkDispatcher) add(sink Sink) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sinks = append(d.sinks, sink)
}

func (d *asyncSinkDispatcher) publish(entry Entry) {
	select {
	case d.ch <- entry:
	default:
		// Drop rather than blocking request paths. The primary zap output still exists.
	}
}

func (d *asyncSinkDispatcher) run() {
	for entry := range d.ch {
		d.mu.RLock()
		sinks := append([]Sink(nil), d.sinks...)
		d.mu.RUnlock()
		for _, sink := range sinks {
			sink.Write(entry)
		}
	}
}
