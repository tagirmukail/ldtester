package router

import (
	"context"
	"crypto/sha256"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/tagirmukail/ldtester/internal/tester"
	"github.com/tagirmukail/ldtester/internal/url_item"
)

const (
	cacheDefaultExpiration = 6 * time.Hour
)

// loadHandler handles urls with load testing every url and returns report {"url": {data}}
func (r *Router) loadHandler(w http.ResponseWriter, req *http.Request) {
	resp := &response{}

	items := make([]url_item.Item, 0)
	err := jsoniter.NewDecoder(req.Body).Decode(items)
	if err != nil {
		resp.Message = err.Error()
		r.json(w, http.StatusBadRequest, resp)
		return
	}

	conf := r.testerConfiguration(req)
	b, _ := jsoniter.Marshal(conf)
	confHashSum := sha256.Sum256(b)
	confHash := string(confHashSum[:])

	result := r.getFromCache(confHash, items)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.options.Cfg.StressTestTimeout)*time.Second)
	ctx = context.WithValue(ctx, tester.IsHandlerKey, true)

	t := tester.New(ctx, cancel, r.options.Log, conf, result.notFoundItems)
	defer t.Stop()

	t.Run()

	report := t.Report()
	for k, item := range report {
		r.options.Cache.Set(confHash, k, item, cacheDefaultExpiration)

		result.report[k] = item
	}

	resultResp := make(map[string]tester.Item)
	for k, item := range report {
		resultResp[k.URL] = item
	}

	r.json(w, http.StatusOK, &response{
		Message:        "successfully",
		LoadTestConfig: conf,
		Data:           resultResp,
	})
}

type getFromCacheResult struct {
	notFoundItems []url_item.Item
	report        map[tester.Key]tester.Item
}

// getFromCache get already load tested urls from the cache,
// this method is necessary for load testing only urls that are missing in the cache
func (r *Router) getFromCache(confHash string, items []url_item.Item) *getFromCacheResult {
	result := &getFromCacheResult{
		report: map[tester.Key]tester.Item{},
	}

	for _, item := range items {
		key := tester.Key{
			Host: item.Host,
			URL:  item.Url,
		}

		existItem, ok := r.options.Cache.Get(confHash, key)
		if !ok {
			result.notFoundItems = append(result.notFoundItems, item)
			continue
		}

		if existItem.GetTesterItem().RecommendReqCount == 0 {
			result.notFoundItems = append(result.notFoundItems, item)
			continue
		}

		result.report[key] = existItem.GetTesterItem()
	}

	return result
}

// testerConfiguration sets tester configuration from request headers and url query params
func (r *Router) testerConfiguration(req *http.Request) tester.Configuration {
	tConf := tester.DefaultConfiguration()

	if r.options.Cfg.LoadTest.MaxIdleConnPerHost > 0 {
		tConf.MaxIdleConnPerHost = r.options.Cfg.LoadTest.MaxIdleConnPerHost
	}

	if r.options.Cfg.LoadTest.Timeout > 0 {
		tConf.Timeout = time.Duration(r.options.Cfg.LoadTest.Timeout) * time.Second
	}

	if r.options.Cfg.LoadTest.Method != "" {
		tConf.Method = r.options.Cfg.LoadTest.Method
	}

	if r.options.Cfg.LoadTest.UseHTTP2 {
		tConf.UseHTTP2 = true
	}

	if r.options.Cfg.LoadTest.DisableCompression {
		tConf.DisableCompression = true
	}

	if r.options.Cfg.LoadTest.DisableKeepAlive {
		tConf.DisableKeepAlive = true
	}

	if r.options.Cfg.LoadTest.AcceptHeaderRequest != "" {
		tConf.AcceptHeaderRequest = r.options.Cfg.LoadTest.AcceptHeaderRequest
	}

	if r.options.Cfg.LoadTest.UserAgent != "" {
		tConf.UserAgent = r.options.Cfg.LoadTest.UserAgent
	}

	tConf = r.testerConfFromReq(tConf, req)

	return tConf
}
