package config

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type Config struct {
	LogLevel logrus.Level
	Server
	LoadTest
}

type Server struct {
	Port         int
	WriteTimeout int
	ReadTimeout  int
}

type LoadTest struct {
	MaxIdleConnPerHost  int
	DisableCompression  bool
	DisableKeepAlive    bool
	UseHTTP2            bool
	Timeout             int
	Method              string
	StressTestTimeout   int
	AcceptHeaderRequest string
	UserAgent           string
}

func DefaultConfig() Config {
	return Config{
		LogLevel: logrus.DebugLevel,
		Server: Server{
			Port:         8000,
			WriteTimeout: 30,
			ReadTimeout:  30,
		},
		LoadTest: LoadTest{
			MaxIdleConnPerHost:  200,
			DisableCompression:  false,
			DisableKeepAlive:    false,
			UseHTTP2:            false,
			Timeout:             3,
			Method:              http.MethodGet,
			StressTestTimeout:   30,
			AcceptHeaderRequest: "",
			UserAgent:           "",
		},
	}
}
