package crawler

import (
	"context"
	"sync"
)

func Run[T any](ctx context.Context,
	input <-chan T,
	f func(e T),
	workerCount int) {
	wg := new(sync.WaitGroup)
	for i := 0; i < workerCount; i++ {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-input:
					if !ok {
						return
					}

					select {
					case <-ctx.Done():
						return
					default:
						f(v)
					}
				}
			}
		})
	}
	wg.Wait()
}
