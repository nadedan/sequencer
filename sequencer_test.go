package sequencer_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nadedan/sequencer"
)

func TestSequencer(t *testing.T) {
	type testCase struct {
		name       string
		jitter     time.Duration
		maxSeqId   int
		stepTime   time.Duration
		seqNums    []int
		expSeqNums []int
		expErr     error
	}

	testCases := []testCase{
		{
			name:       "normal in order",
			jitter:     3 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{0, 1, 2, 3, 4, 5, 6, 7},
			expSeqNums: []int{0, 1, 2, 3, 4, 5, 6, 7},
		},
		{
			name:       "out of order with enough time",
			jitter:     3 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{0, 2, 1, 3, 4, 5},
			expSeqNums: []int{0, 1, 2, 3, 4, 5},
		},
		{
			name:       "out of order not enough time",
			jitter:     2 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{0, 4, 3, 2, 5, 6, 7, 8},
			expSeqNums: []int{0, 4, 5, 6, 7, 8},
		},
		{
			name:       "non-zero start",
			jitter:     3 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{1, 2, 3, 4, 5},
			expSeqNums: []int{1, 2, 3, 4, 5},
		},
		{
			name:       "normal in order with seqId rollover",
			jitter:     3 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{97, 98, 99, 100, 0, 1, 2, 3, 4},
			expSeqNums: []int{97, 98, 99, 100, 0, 1, 2, 3, 4},
		},
		{
			name:       "out of order with enough time with seqId rollover",
			jitter:     3 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{97, 98, 99, 100, 1, 0, 2, 3, 4},
			expSeqNums: []int{97, 98, 99, 100, 0, 1, 2, 3, 4},
		},
		{
			name:       "out of order with enough time across rollover boundary",
			jitter:     3 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{97, 98, 99, 1, 100, 0, 2, 3, 4},
			expSeqNums: []int{97, 98, 99, 100, 0, 1, 2, 3, 4},
		},
		{
			name:       "out of order with not enough time across rollover boundary",
			jitter:     2 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{97, 98, 99, 3, 2, 1, 4, 5, 6},
			expSeqNums: []int{97, 98, 99, 3, 4, 5, 6},
		},
		{
			name:       "out of order with not enough time prevent stuck packet",
			jitter:     2 * time.Millisecond,
			maxSeqId:   100,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{97, 98, 99, 3, 2, 1, 0, 4, 5, 6},
			expSeqNums: []int{97, 98, 99, 3, 4, 5, 6},
			expErr:     sequencer.ErrInRecovery,
		},
	}

	for id, thisTest := range testCases {
		name := thisTest.name
		if len(name) == 0 {
			name = fmt.Sprint(id)
		}
		t.Run(name, func(t *testing.T) {
			s := sequencer.New[int](thisTest.jitter, thisTest.maxSeqId)
			gotSeqNums := make([]int, 0)

			ctx, stop := context.WithCancel(context.Background())
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					p, err := s.WaitForNext(ctx)
					switch {
					case err == nil:
					case err == sequencer.ErrCtxCanceled:
						return
					default:
						panic(err)
					}
					gotSeqNums = append(gotSeqNums, *p)
					t.Logf("Got   %d", *p)
				}
			}()

			ticker := time.NewTicker(thisTest.stepTime)
			defer ticker.Stop()

			i := 0
			for range ticker.C {
				if i >= len(thisTest.seqNums) {
					break
				}
				n := thisTest.seqNums[i]

				t.Logf("Add %d", n)
				err := s.Add(n, n)
				if err != nil {
					if !errors.Is(err, thisTest.expErr) {
						t.Errorf("did not get expected error '%s', got '%s'", thisTest.expErr, err)
					}
					t.Logf("got expected error: %s", err)
				}

				i++
			}

			stop()
			wg.Wait()

			if len(gotSeqNums) != len(thisTest.expSeqNums) {
				t.Errorf("got %d seqNums but expected %d", len(gotSeqNums), len(thisTest.expSeqNums))
				return
			}

			for i := range thisTest.expSeqNums {
				if thisTest.expSeqNums[i] != gotSeqNums[i] {
					t.Errorf("expected the %dth seqNum to be %d, but it was %d",
						i,
						thisTest.expSeqNums[i],
						gotSeqNums[i],
					)
					return
				}
			}

		})
	}

}
