package router

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/tagirmukail/ldtester/internal/tester"

	jsoniter "github.com/json-iterator/go"
)

const (
	contentTypeHeader = "Content-Type"
	contentTypeJson   = "application/json"
)

type response struct {
	Message        string               `json:"message,omitempty"`
	LoadTestConfig tester.Configuration `json:"load_test_config"`
	Data           interface{}          `json:"data,omitempty"`
}

func (r *Router) json(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set(contentTypeHeader, contentTypeJson)
	w.WriteHeader(status)

	err := jsoniter.NewEncoder(w).Encode(data)
	if err != nil {
		r.options.Log.WithError(err).Error("write response failed")
	}
}

func (r *Router) testerConfFromReq(c tester.Configuration, req *http.Request) tester.Configuration {
	maxIdleConn, _ := r.testerConfSetParamInt(maxIdleConnPerHostHeader, maxIdleConnPerHostParam, req)
	if maxIdleConn > 0 {
		c.MaxIdleConnPerHost = maxIdleConn
	}

	reqTimeout, _ := r.testerConfSetParamInt(reqTimeoutHeader, reqTimeoutParam, req)
	if reqTimeout > 0 {
		c.Timeout = time.Duration(reqTimeout) * time.Second
	}

	disableCompress := r.testerConfReqBool(disableCompressionHeader, disableCompressionParam, req)
	if disableCompress {
		c.DisableCompression = disableCompress
	}

	disableKeepAlive := r.testerConfReqBool(disableKeepAliveHeader, disableKeepAliveParam, req)
	if disableCompress {
		c.DisableKeepAlive = disableKeepAlive
	}

	reqMethod := r.testerConfReqString(reqMethodHeader, reqMethodParam, req)
	if reqMethod != "" {
		c.Method = reqMethod
	}

	accept := req.Header.Get(reqAcceptHeader)
	if accept != "" {
		c.AcceptHeaderRequest = accept
	}

	userAgent := req.Header.Get(reqUserAgentHeader)
	if userAgent != "" {
		c.UserAgent = userAgent
	}

	return c
}

func (r *Router) testerConfSetParamInt(header, queryParam string, req *http.Request) (int, error) {
	val := req.Header.Get(header)
	if val != "" {
		convVal, err := strconv.Atoi(val)
		if err != nil {
			return 0, err
		}

		return convVal, nil
	}

	val = req.URL.Query().Get(queryParam)
	if val != "" {
		convVal, err := strconv.Atoi(val)
		if err != nil {
			return 0, err
		}

		return convVal, nil
	}

	return 0, nil
}

func (r *Router) testerConfReqBool(header, queryParam string, req *http.Request) bool {
	val := req.Header.Get(header)
	if val != "" {
		return true
	}

	val = req.URL.Query().Get(queryParam)
	if val != "" {
		return true
	}

	return false
}

func (r *Router) testerConfReqString(header, queryParam string, req *http.Request) string {
	val := req.Header.Get(header)
	if val != "" {
		return val
	}

	val = req.URL.Query().Get(queryParam)
	if val != "" {
		return val
	}

	return val
}

// reportMergeByHost merges all reports items by hosts
func reportMergeByHost(report map[tester.Key]tester.Item) map[string]tester.Item {
	var result = make(map[string]tester.Item)

	hostURLsCounters := make(map[string]int)

	for key, item := range report {
		resultItemByHost := result[key.Host]

		if item.MaxReqTime > resultItemByHost.MaxReqTime {
			resultItemByHost.MaxReqTime = item.MaxReqTime
		}

		resultItemByHost.SlowReqCount += item.SlowReqCount
		resultItemByHost.ErrRequestCount += item.ErrRequestCount
		resultItemByHost.TotalReqCount += item.TotalReqCount

		count := hostURLsCounters[key.Host]
		resultItemByHost.RecommendReqCount = calcAVG(
			count,
			float64(resultItemByHost.RecommendReqCount),
			float64(item.RecommendReqCount),
		)
		hostURLsCounters[key.Host] = count + 1

		result[key.Host] = resultItemByHost
	}

	return result
}

func calcAVG(currentCount int, currentVal, val float64) int {
	if currentCount < 1 {
		return int(math.Round(val))
	}

	currentSum := float64(currentCount) * currentVal

	result := (currentSum + val) / float64(currentCount+1)

	return int(math.Round(result))
}
