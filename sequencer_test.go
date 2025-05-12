package sequencer_test

import (
	"context"
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
		stepTime   time.Duration
		seqNums    []int
		expSeqNums []int
	}

	testCases := []testCase{
		{
			name:       "normal in order",
			jitter:     10 * time.Millisecond,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{0, 1, 2, 3, 4, 5},
			expSeqNums: []int{0, 1, 2, 3, 4, 5},
		},
		{
			name:       "out of order with enough time",
			jitter:     10 * time.Millisecond,
			stepTime:   1 * time.Millisecond,
			seqNums:    []int{2, 1, 0, 3, 4, 5},
			expSeqNums: []int{0, 1, 2, 3, 4, 5},
		},
	}

	for id, thisTest := range testCases {
		name := thisTest.name
		if len(name) == 0 {
			name = fmt.Sprint(id)
		}
		t.Run(name, func(t *testing.T) {
			s := sequencer.New[int, int](thisTest.jitter)
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
				s.Add(n, n)

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
