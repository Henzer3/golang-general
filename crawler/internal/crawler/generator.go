package crawler

import "context"

func Generate[T, R any](ctx context.Context, data []T, f func(index int, e T) R, size int) <-chan R {
	result := make(chan R, size)

	go func() {
		defer close(result)
		for i := 0; i < len(data); i++ {
			select {
			case <-ctx.Done():
				return
			case result <- f(i, data[i]):
			}
		}
	}()

	return result
}
