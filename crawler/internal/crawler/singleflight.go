package crawler

import "golang.org/x/sync/singleflight"

type singleflightImpl[T any] struct {
	x singleflight.Group
}

func (s *singleflightImpl[T]) Do(key string, fn func() (T, error)) (T, error, bool) {
	v, err, shared := s.x.Do(key, func() (any, error) {
		return fn()
	})
	return v.(T), err, shared
}
