package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	nurl "net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"

	"github.com/tagirmukail/ldtester/internal/config"
	"github.com/tagirmukail/ldtester/internal/logger"
	"github.com/tagirmukail/ldtester/internal/router"
	"github.com/tagirmukail/ldtester/internal/tester"
	"github.com/tagirmukail/ldtester/internal/url_item"
)

const (
	configFlagName  = "config"
	loadCSVFlagName = "loadcsv"
	urlFlagName     = "url"
	methodFlagName  = "method"
)

func main() {
	app := cli.App{
		Name: "ldtester",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"cfg", "conf", "c"},
				Usage:       "application configuration",
				DefaultText: "config.yaml",
				Required:    false,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "server",
				Action: runServer,
			},
			{
				Name: "load",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    loadCSVFlagName,
						Aliases: []string{"f"},
						Usage: "Get from this csv file urls and run load test for all. File data format:" +
							"url",
					},
					&cli.StringFlag{
						Name:    urlFlagName,
						Aliases: []string{"u"},
						Usage:   "Load test url",
					},
					&cli.StringFlag{
						Name:    methodFlagName,
						Aliases: []string{"m"},
						Usage:   "Method for load test url",
					},
				},
				Action: runLoad,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}

func runLoad(c *cli.Context) error {
	ctx, cancel := context.WithCancel(c.Context)
	defer cancel()

	fmt.Println("setup configuration...")

	cfg := initConfig(c.String(configFlagName))

	fmt.Printf("===============\n%+v\n===============\n", cfg)

	fmt.Println("setup configuration done.")

	log := logger.New(ctx, cfg.LogLevel, os.Stdout)

	csvFile := c.String(loadCSVFlagName)
	url := c.String(urlFlagName)
	method := c.String(methodFlagName)

	items, err := loadTestItems(csvFile, url)
	if err != nil {
		return err
	}

	conf := tester.FromGlobalConfig(cfg.LoadTest)

	if method != "" {
		conf.Method = method
	}

	t := tester.New(ctx, cancel, log, conf, items)

	t.Run()

	report := t.Report()

	formattedOutputReport(report)

	return nil
}

func runServer(c *cli.Context) error {
	ctx, cancel := context.WithCancel(c.Context)
	defer cancel()

	fmt.Println("setup configuration...")

	cfg := initConfig(c.String(configFlagName))

	fmt.Printf("===============\n%+v\n===============\n", cfg)

	fmt.Println("setup configuration done.")

	log := logger.New(ctx, cfg.LogLevel, os.Stdout)

	fmt.Println("setup router...")

	options := &router.Options{
		Cfg: &cfg,
		Log: log,
	}

	r := router.New(options)

	defer options.Cache.Close()

	fmt.Println("setup router done...")

	fmt.Printf("tester is ready to accept connections for 0.0.0.0:%d/sites?search= ...\n", cfg.Port)

	return r.Serve()
}

func initConfig(configFile string) config.Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetConfigFile(configFile)
	v.SetEnvPrefix("core")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.AddConfigPath(".")

	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}

	cfg := config.DefaultConfig()

	err = v.Unmarshal(&cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}

func loadTestItems(csvFile, url string) ([]url_item.Item, error) {
	if url != "" {
		parsedURL, err := nurl.Parse(url)
		if err != nil {
			return nil, err
		}

		return append([]url_item.Item{}, url_item.Item{
			Host: parsedURL.Hostname(),
			Url:  parsedURL.String(),
		}), nil
	}

	f, err := os.Open(csvFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make([]url_item.Item, 0)

	reader := csv.NewReader(f)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		if len(record) == 0 {
			return nil, fmt.Errorf("invalid row: %+v", record)
		}

		parsedURL, err := nurl.Parse(record[0])
		if err != nil {
			return nil, err
		}

		result = append(result, url_item.Item{
			Host: parsedURL.Hostname(),
			Url:  parsedURL.String(),
		})
	}

	return result, nil
}

const (
	reportSplitResultRow = "---------------------------------------"
	reportSplitRow       = "======================================="
)

func formattedOutputReport(report map[tester.Key]tester.Item) {
	fmt.Println(reportSplitRow)

	for key, item := range report {
		fmt.Printf("Load test for %s.\n", key.URL)
		fmt.Printf("Total sends requests %d.\n", item.TotalReqCount)
		fmt.Printf("Failed requests %d.\n", item.ErrRequestCount)
		fmt.Printf("Slow requests %d.\n", item.SlowReqCount)
		fmt.Printf("Max request time %v s.\n", item.MaxReqTime)
		fmt.Println(reportSplitResultRow)
		fmt.Printf("Recommended requests count %d\n", item.RecommendReqCount)
		fmt.Println(reportSplitRow)
	}
}
