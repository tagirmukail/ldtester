package tester

import (
	"sync"
	"time"
)

type Key struct {
	Host string
	URL  string
}

type Error struct {
	err         error
	url         string
	maxRequests int64
}

type GlobResult struct {
	mx sync.Mutex
	m  map[Key]Item
}

type Item struct {
	RecommendReqCount int     `json:"recommend_req_count"`
	TotalReqCount     int     `json:"total_req_count"`
	ErrRequestCount   int     `json:"err_request_count"`
	MaxReqTime        float64 `json:"max_req_time"`
	SlowReqCount      int     `json:"slow_req_count"`
}

func NewResult() *GlobResult {
	return &GlobResult{
		mx: sync.Mutex{},
		m:  make(map[Key]Item),
	}
}

func (r *GlobResult) ProcessItem(key Key, fn func(m map[Key]Item, i Item)) {
	r.mx.Lock()
	defer r.mx.Unlock()

	i := r.m[key]

	fn(r.m, i)
}

func (r *GlobResult) Set(key Key, item Item) {
	r.mx.Lock()
	defer r.mx.Unlock()

	r.m[key] = item
}

func (r *GlobResult) GetResult() map[Key]Item {
	return r.m
}

type requestResult struct {
	urlKey         string
	host           string
	offset         time.Duration
	statusCode     int
	finishDuration time.Duration
	err            error
	connDuration   time.Duration
	dnsDuration    time.Duration
	reqDuration    time.Duration
	respDuration   time.Duration
	delayDuration  time.Duration
}

type throttlingChecker struct {
	mx sync.Mutex
	m  map[string]int
}

func (t *throttlingChecker) Throttle(url string, count int) {
	t.mx.Lock()
	defer t.mx.Unlock()
	haveCount := t.m[url]
	if haveCount > 0 {
		return
	}

	t.m[url] = count
}

func (t *throttlingChecker) Check(url string) int {
	t.mx.Lock()
	defer t.mx.Unlock()

	return t.m[url]
}
