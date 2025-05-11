package sequencer

import (
	"time"

	"github.com/nadedan/sequencer/pkg/cache"
	"golang.org/x/exp/constraints"
)

type s[N constraints.Ordered, P any] struct {
	jitter time.Duration

	packets *cache.Cache[N, P]

	next chan P
}

func New[N constraints.Ordered, P any](jitter time.Duration) *s[N, P] {
	newS := &s[N, P]{
		jitter:  jitter,
		packets: cache.New[N, P](),
		next:    make(chan P, 10),
	}

	return newS
}

func (s *s[N, P]) Add(n N, p P) {
	s.packets.Store(n, p)
}

func (s *s[N, P]) Next() (P, error) {
	n := <-s.next
	return n, nil
}
