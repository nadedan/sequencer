package sequencer

import (
	"time"

	"github.com/nadedan/sequencer/pkg/cache"
)

type s[T any] struct {
	jitter time.Duration

	packets *cache.Cache[SeqNum, T]

	next chan T
}

type (
	SeqNum int
)

func New[T any](jitter time.Duration) *s[T] {
	newS := &s[T]{
		jitter:  jitter,
		packets: cache.New[SeqNum, T](),
		next:    make(chan T, 10),
	}

	return newS
}

func (s *s[T]) Add(n SeqNum, p T) {
}

func (s *s[T]) Next() (T, error) {
	n := <-s.next
	return n, nil
}
