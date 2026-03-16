package lfucache

import (
	"errors"
	"iter"
)

var ErrKeyNotFound = errors.New("key not found")

const DefaultCapacity = 5

type Cache[K comparable, V any] interface {
	Get(key K) (V, error)
	Put(key K, value V)
	All() iter.Seq2[K, V]
	Size() int
	Capacity() int
	GetKeyFrequency(key K) (int, error)
}

type cacheImpl[K comparable, V any] struct{}

func New[K comparable, V any](_ ...int) *cacheImpl[K, V] {
	return new(cacheImpl[K, V])
}

func (l *cacheImpl[K, V]) Get(_ K) (V, error) {
	panic("not implemented")
}

func (l *cacheImpl[K, V]) Put(_ K, _ V) {
	panic("not implemented")
}

func (l *cacheImpl[K, V]) All() iter.Seq2[K, V] {
	panic("not implemented")
}

func (l *cacheImpl[K, V]) Size() int {
	panic("not implemented")
}

func (l *cacheImpl[K, V]) Capacity() int {
	panic("not implemented")
}

func (l *cacheImpl[K, V]) GetKeyFrequency(_ K) (int, error) {
	panic("not implemented")
}
