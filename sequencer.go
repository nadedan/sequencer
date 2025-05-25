package sequencer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nadedan/sequencer/pkg/cache"
)

type S[P any] struct {
	jitter time.Duration

	packets *cache.Cache[int, P]

	// thisSeqId is the last in-order sequence id that we have received
	thisSeqId int
	// maxReceivedSeqId is the largest sequence id that we have received
	// and is currenly sitting in our packets cache
	maxReceivedSeqId int
	// maxSeqId is the highest sequence number before it rolls over
	maxSeqId int

	timedOutSeqId  int
	timedOutFlag   bool
	recoveryWindow int

	gotNext chan struct{}
	next    chan P

	mut sync.RWMutex

	primed bool
}

// New creates a new sequencer for packet type P
//
// Arguments:
//   - jitter: the amount of time we will wait for the next
//     in-order sequence number before we drop our queue and move on
//   - maxSeqId: the maximum sequence id allowed by the protocol
func New[P any](jitter time.Duration, maxSeqId int) *S[P] {
	s := &S[P]{
		jitter:         jitter,
		maxSeqId:       maxSeqId,
		packets:        cache.New[int, P](),
		gotNext:        make(chan struct{}, 10),
		next:           make(chan P, 10),
		recoveryWindow: 10,
	}

	go s.waitForNext()
	return s
}

// SetRecoveryWindow allows us to change the recovery window for the resequencer
//
// The recovery window is the amount of packets we have to receive, after a
// timeout, before we start accepting smaller sequnce ids again
func (s *S[P]) SetRecoveryWindow(w int) {
	s.recoveryWindow = w
}

// Add packets to the sequencer queue
func (s *S[P]) Add(seqId int, packet P) error {
	if s.inRecovery() && seqId < s.thisSeqId {
		return fmt.Errorf("dropping seqId %d: %w", seqId, ErrInRecovery)
	}

	s.packets.Store(seqId, packet)
	gotNext := false

	s.mut.Lock()
	switch {
	case !s.primed:
		s.primed = true
		s.thisSeqId = seqId
		fallthrough
	case seqId > s.maxReceivedSeqId ||
		(s.maxReceivedSeqId-seqId) > s.maxSeqId/2: // detecting a rollover
		s.maxReceivedSeqId = seqId
		fallthrough
	case seqId == s.nextSeqId():
		gotNext = true
	}
	s.mut.Unlock()

	if gotNext {
		s.gotNext <- struct{}{}
	}

	return nil
}

// Next returns the next in-order packet from the queue.
// Returns ErrNoPacketReady if there are no packets
// ready to be returned
func (s *S[P]) Next() (*P, error) {
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

// WaitForNext blocks until a packet can be returned or the context
// is canceled
func (s *S[P]) WaitForNext(ctx context.Context) (*P, error) {
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

func (s *S[P]) waitForNext() {
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
			timer.Reset(s.jitter)
			timedOut = false
		}
		s.emitPackets(timedOut)
	}
}

func (s *S[P]) emitPackets(timedOut bool) {
	//fmt.Printf("emitting packets with timedOut:%v\n", timedOut)
	s.mut.Lock()
	defer s.mut.Unlock()
	if timedOut && s.maxReceivedSeqId > s.nextSeqId() {
		s.thisSeqId = s.maxReceivedSeqId
	}

	for ; s.thisSeqId <= s.maxReceivedSeqId; s.thisSeqId = s.nextSeqId() {
		p, ok := s.packets.Load(s.thisSeqId)
		if !ok {
			return
		}
		s.next <- p
		s.packets.Delete(s.thisSeqId)
	}

	if timedOut {
		s.timedOutFlag = true
		s.timedOutSeqId = s.maxReceivedSeqId
		s.packets.Clear()
	}
}

func (s *S[P]) nextSeqId() int {
	if s.thisSeqId == s.maxSeqId {
		return 0
	}

	return s.thisSeqId + 1
}

func (s *S[P]) seqIdDistance(oldest int, newest int) int {
	if newest < oldest {
		return s.maxSeqId - (oldest - newest)
	}

	return newest - oldest
}

func (s *S[P]) inRecovery() bool {
	if s.timedOutFlag &&
		s.seqIdDistance(s.timedOutSeqId, s.thisSeqId) < s.recoveryWindow {
		return true
	}

	s.timedOutFlag = false
	return false
}
