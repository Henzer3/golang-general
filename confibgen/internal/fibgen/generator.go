package fibonacci

import (
	"errors"
	"math"
	"runtime"
	"sync/atomic"
)

var ErrOverflow = errors.New("fibonacci sequence overflow")

const (
	fistNumberFibonacci   = 0
	secondNumberFibonacci = 1
)

type Generator interface {
	Next() uint64
}

var _ Generator = (*generatorImpl)(nil)

type generatorImpl struct {
	a    uint64
	b    uint64
	lock atomic.Bool
}

func NewGenerator() *generatorImpl {
	c := &generatorImpl{
		a: secondNumberFibonacci,
		b: fistNumberFibonacci,
	}
	c.lock.Store(false)
	return c
}

func (g *generatorImpl) Lock() {
	for {
		if g.lock.CompareAndSwap(false, true) {
			return
		}
		runtime.Gosched()
	}
}

func (g *generatorImpl) Unlock() {
	g.lock.Store(false)
}

func (g *generatorImpl) Next() uint64 {
	g.Lock()
	defer g.Unlock()

	if g.b == math.MaxUint64 {
		panic(ErrOverflow)
	}

	oldB := g.b

	if oldB > math.MaxUint64-g.a {
		g.a = oldB
		g.b = math.MaxUint64
	} else {
		g.b = g.a + oldB
		g.a = oldB
	}
	return g.a
}
