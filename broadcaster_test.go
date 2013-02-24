package broadcast

import (
	"sync"
	"testing"
)

func TestBroadcast(t *testing.T) {
	wg := sync.WaitGroup{}

	b := NewBroadcaster(100)
	defer b.Close()

	for i := 0; i < 5; i++ {
		wg.Add(1)

		cch := make(chan interface{})

		b.Register(cch)

		go func() {
			defer wg.Done()
			defer b.Unregister(cch)
			<-cch
		}()

	}

	b.Submit(1)

	wg.Wait()
}

func TestBroadcastCleanup(t *testing.T) {
	b := NewBroadcaster(100)
	b.Register(make(chan interface{}))
	b.Close()
}
