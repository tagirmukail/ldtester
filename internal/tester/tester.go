package tester

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptrace"
	"os"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"

	"github.com/tagirmukail/ldtester/internal/config"
	"github.com/tagirmukail/ldtester/internal/url_item"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
)

const (
	IsHandlerKey int = 1

	httpClientTimeout = 15 * time.Second

	acceptHeader    = "accept"
	userAgentHeader = "user-agent"
)

type Configuration struct {
	MaxIdleConnPerHost  int           `json:"max_idle_conn_per_host"`
	DisableCompression  bool          `json:"disable_compression"`
	DisableKeepAlive    bool          `json:"disable_keep_alive"`
	UseHTTP2            bool          `json:"use_http_2"`
	Timeout             time.Duration `json:"timeout"`
	Method              string        `json:"method"`
	AcceptHeaderRequest string        `json:"accept_header_request"`
	UserAgent           string        `json:"user_agent"`
}

// DefaultConfiguration sets default configuration for load testing
func DefaultConfiguration() Configuration {
	conf := Configuration{
		MaxIdleConnPerHost: 200,
		DisableCompression: false,
		DisableKeepAlive:   false,
		UseHTTP2:           false,
		Timeout:            3 * time.Second,
		Method:             http.MethodGet,
	}

	return conf
}

func FromGlobalConfig(loadTestConf config.LoadTest) Configuration {
	conf := DefaultConfiguration()

	if loadTestConf.Timeout > 0 {
		conf.Timeout = time.Duration(loadTestConf.Timeout) * time.Second
	}

	if loadTestConf.DisableCompression {
		conf.DisableCompression = true
	}

	if loadTestConf.DisableKeepAlive {
		conf.DisableKeepAlive = true
	}

	if loadTestConf.UserAgent != "" {
		conf.UserAgent = loadTestConf.UserAgent
	}

	if loadTestConf.AcceptHeaderRequest != "" {
		conf.AcceptHeaderRequest = loadTestConf.AcceptHeaderRequest
	}

	if loadTestConf.Method != "" {
		conf.Method = loadTestConf.Method
	}

	if loadTestConf.MaxIdleConnPerHost > 0 {
		conf.MaxIdleConnPerHost = loadTestConf.MaxIdleConnPerHost
	}

	return conf
}

// Tester represents load testing struct
type Tester struct {
	shutdownCtx context.Context
	cancel      context.CancelFunc

	log logrus.FieldLogger

	conf Configuration

	items []url_item.Item

	reqResultCh       chan *requestResult
	throttlingChecker *throttlingChecker

	stopCh chan struct{}

	report *report
}

func New(shutdownCtx context.Context, cancel context.CancelFunc,
	log logrus.FieldLogger, conf Configuration, items []url_item.Item) *Tester {
	t := &Tester{
		shutdownCtx: shutdownCtx,
		cancel:      cancel,

		log: log,

		items: items,

		conf: conf,

		throttlingChecker: &throttlingChecker{
			mx: sync.Mutex{},
			m:  make(map[string]int),
		},
	}

	t.reqResultCh = make(chan *requestResult, len(items)*2)

	t.report = newReport(shutdownCtx, t.reqResultCh, conf.Timeout)

	return t
}

func (t *Tester) Run() {
	if len(t.items) == 0 {
		return
	}

	go t.report.runReport()

	t.runWorkers()
	t.finalize()
}

func (t *Tester) Stop() {
	t.cancel()
}

func (t *Tester) Report() map[Key]Item {
	return t.report.globResult.GetResult()
}

func (t *Tester) finalize() {
	close(t.reqResultCh)
	select {
	case <-t.report.done:
		return
	case <-t.shutdownCtx.Done():
		return
	}
}

// runWorkers runs load testing workers for every url
func (t *Tester) runWorkers() {

	wg := sync.WaitGroup{}

	for i, item := range t.items {
		i := i
		item := item

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         item.Host,
			},
			MaxIdleConnsPerHost: t.conf.MaxIdleConnPerHost,
			DisableCompression:  t.conf.DisableCompression,
			DisableKeepAlives:   t.conf.DisableKeepAlive,
		}

		if t.conf.UseHTTP2 {
			_ = http2.ConfigureTransport(tr)
		} else {
			tr.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
		}

		client := &http.Client{
			Transport: tr,
			Timeout:   httpClientTimeout,
		}

		wg.Add(1)
		go func() {
			t.runWorker(i, client, item)
			wg.Done()
		}()
	}

	wg.Wait()
}

// runWorker runs one worker for url
func (t *Tester) runWorker(workerNum int, client *http.Client, item url_item.Item) {
	var (
		numRequests = 1
		isHandler   bool
	)

	isHandlerVal := t.shutdownCtx.Value(IsHandlerKey)
	if isHandlerVal != nil {
		isHandler = isHandlerVal.(bool)
	}

	for {
		select {
		case <-t.shutdownCtx.Done():
			t.log.WithField("url", item.Url).WithField("worker_num", workerNum).Info("worker canceled")

			return
		default:
			count := t.throttlingChecker.Check(item.Url)
			if count > 0 {
				t.log.WithField("url", item.Url).WithField("worker_num", workerNum).
					WithField("throttling_requests", count).Info("worker stopped")
				return
			}

			numRequests++
		}

		var bar *pb.ProgressBar
		if !isHandler {
			t.log.WithField("url", item.Url).WithField("req_num", numRequests).Info("started")
			bar = pb.StartNew(numRequests)
		}

		wg := sync.WaitGroup{}
		for i := 0; i < numRequests; i++ {
			i := i

			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer func() {
					wg.Done()

					if bar != nil {
						bar.Increment()
					}
				}()

				select {
				case <-t.shutdownCtx.Done():
					t.log.WithField("url", item.Url).WithField("worker_num", workerNum).
						WithField("req_num", i).
						Info("worker req num canceled")

					return
				default:
				}

				t.doRequest(client, item, numRequests)
			}(&wg)
		}

		wg.Wait()

		if bar != nil {
			t.log.WithField("url", item.Url).WithField("req_num", numRequests).Info("finished")
			bar.Finish()
		}
	}

}

// doRequest does request with analyze
func (t *Tester) doRequest(client *http.Client, item url_item.Item, numRequests int) {
	var (
		now        = time.Now()
		nowSince   = since(now)
		dnsStart   time.Duration
		startConn  time.Duration
		reqStart   time.Duration
		delayStart time.Duration
		respStart  time.Duration

		result = &requestResult{
			urlKey: item.Url,
			host:   item.Host,
			offset: nowSince,
		}
	)

	req, _ := http.NewRequestWithContext(t.shutdownCtx, t.conf.Method, item.Url, nil)

	req.Header.Set(acceptHeader, t.conf.AcceptHeaderRequest)
	req.Header.Set(userAgentHeader, t.conf.UserAgent)

	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = since(now)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			result.dnsDuration = since(now) - dnsStart
		},
		GetConn: func(h string) {
			startConn = since(now)
		},
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				result.connDuration = since(now) - startConn
			}

			reqStart = since(now)
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			result.reqDuration = since(now) - reqStart
			delayStart = since(now)
		},
		GotFirstResponseByte: func() {
			result.delayDuration = since(now) - delayStart
			respStart = since(now)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := client.Do(req)
	result.err = err
	switch {
	case err == nil:
		result.statusCode = resp.StatusCode
		_ = resp.Body.Close()
	case errors.Is(err, context.DeadlineExceeded):
		t.throttlingChecker.Throttle(item.Url, numRequests)
	case os.IsTimeout(err):
		t.throttlingChecker.Throttle(item.Url, numRequests)
	default:
		t.throttlingChecker.Throttle(item.Url, numRequests)
		t.log.
			WithError(err).
			WithField("url", item.Url).
			Error("do request failed")
	}

	finishedDuration := since(now)
	result.respDuration = finishedDuration - respStart

	result.finishDuration = finishedDuration - nowSince

	t.reqResultCh <- result
}

func since(t time.Time) time.Duration {
	return time.Since(t)
}
