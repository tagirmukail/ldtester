package tester

import (
	"context"
	"time"
)

type report struct {
	shutdownCtx context.Context

	results chan *requestResult

	done chan struct{}

	globResult *GlobResult

	maxReqDuration time.Duration
}

func newReport(shutdownCtx context.Context, resultsCh chan *requestResult, maxReqDuration time.Duration) *report {
	return &report{
		shutdownCtx: shutdownCtx,
		results:     resultsCh,
		globResult:  NewResult(),
		done:        make(chan struct{}),

		maxReqDuration: maxReqDuration,
	}
}

func (r *report) runReport() {
	for reqResult := range r.results {
		key := Key{
			Host: reqResult.host,
			URL:  reqResult.urlKey,
		}

		r.globResult.ProcessItem(key, func(m map[Key]Item, i Item) {
			defer func() { m[key] = i }()

			i.TotalReqCount++

			if reqResult.err != nil {
				i.ErrRequestCount++

				return
			}

			if reqResult.finishDuration.Seconds() > i.MaxReqTime {
				i.MaxReqTime = reqResult.finishDuration.Seconds()
			}

			if reqResult.finishDuration >= r.maxReqDuration {
				i.SlowReqCount++

				return
			}

			// increment i.RecommendReqCount only if no err and finish duration less than max request duration
			//and not exist any error and all request is fast
			if i.ErrRequestCount == 0 && i.SlowReqCount == 0 {
				i.RecommendReqCount++
			}
		})
	}

	r.stop()
}

func (r *report) stop() {
	select {
	case <-r.shutdownCtx.Done():
		return
	case r.done <- struct{}{}:
		return
	}
}
