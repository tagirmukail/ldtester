package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/tagirmukail/ldtester/internal/cache"
	"github.com/tagirmukail/ldtester/internal/config"
	"github.com/tagirmukail/ldtester/internal/logger"
)

const timeout = 10 * time.Second

type Options struct {
	Cfg     *config.Config
	Log     logger.Logger
	HTTPCli *http.Client
	Cache   *cache.Cache
}

type Router struct {
	options *Options
}

func New(opts *Options) *Router {
	if opts == nil {
		panic("router options is empty")
	}

	if opts.Cfg == nil {
		panic("router options configuration is emty")
	}

	if opts.Log == nil {
		opts.Log = logrus.StandardLogger()
	}

	if opts.HTTPCli == nil {
		opts.HTTPCli = &http.Client{}
		opts.HTTPCli.Timeout = timeout

	}

	if opts.Cache == nil {
		opts.Cache = cache.New()
	}

	return &Router{options: opts}
}

func (r *Router) router() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/load", r.loadHandler).Methods(http.MethodPost)

	return router
}

func (r *Router) Serve() error {
	srv := &http.Server{
		Handler:      r.router(),
		Addr:         fmt.Sprintf(":%d", r.options.Cfg.Port),
		WriteTimeout: time.Duration(r.options.Cfg.WriteTimeout) * time.Second,
		ReadTimeout:  time.Duration(r.options.Cfg.ReadTimeout) * time.Second,
	}

	return srv.ListenAndServe()
}
