package sequencer

import (
	"context"
	"sync"
	"time"

	"github.com/nadedan/sequencer/pkg/cache"

	"golang.org/x/exp/constraints"
)

type S[N constraints.Integer, P any] struct {
	jitter time.Duration

	packets *cache.Cache[N, P]

	// nextSeqId is the next sequence id that we need to emit.
	// once this sequence id is received, we can emit more packets.
	nextSeqId N
	// maxSeqId is the largest sequence id that we have received
	// and is currenly sitting in our packets cache
	maxSeqId N

	gotNext chan struct{}
	next    chan P

	mut sync.RWMutex
}

func New[N constraints.Integer, P any](jitter time.Duration) *S[N, P] {
	s := &S[N, P]{
		jitter:  jitter,
		packets: cache.New[N, P](),
		gotNext: make(chan struct{}, 10),
		next:    make(chan P, 10),
	}

	go s.waitForNext()
	return s
}

func (s *S[N, P]) Add(n N, p P) {
	s.packets.Store(n, p)
	gotNext := false

	s.mut.Lock()
	switch {
	case n > s.maxSeqId:
		s.maxSeqId = n
		fallthrough
	case n == s.nextSeqId:
		gotNext = true
	}
	s.mut.Unlock()

	if gotNext {
		s.gotNext <- struct{}{}
	}
}

func (s *S[N, P]) Next() (*P, error) {
	select {
	case p, ok := <-s.next:
		if !ok {
			panic("next channel has been closed")
		}
		return &p, nil
	default:
		return nil, ErrNoPacketReady
	}
}

func (s *S[N, P]) WaitForNext(ctx context.Context) (*P, error) {
	select {
	case p, ok := <-s.next:
		if !ok {
			panic("next channel has been closed")
		}
		return &p, nil
	case <-ctx.Done():
		return nil, ErrCtxCanceled
	}
}

func (s *S[N, P]) waitForNext() {
	timer := time.NewTimer(s.jitter)
	defer timer.Stop()

	timedOut := false
	for {
		select {
		case <-timer.C:
			// we hit the end of our jitter window
			// emit the maxSeqId and reset the cache
			timedOut = true
		case _, ok := <-s.gotNext:
			if !ok {
				return
			}
			timedOut = false
		}
		s.emitPackets(timedOut)
	}
}

func (s *S[N, P]) emitPackets(timedOut bool) {
	s.mut.Lock()
	defer s.mut.Unlock()
	if timedOut {
		s.nextSeqId = s.maxSeqId
	}

	for ; s.nextSeqId <= s.maxSeqId; s.nextSeqId++ {
		p, ok := s.packets.Load(s.nextSeqId)
		if !ok {
			return
		}
		s.next <- p
		s.packets.Delete(s.nextSeqId)
	}

	if timedOut {
		s.packets.Clear()
	}
}
