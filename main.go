package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	whoisparser "github.com/likexian/whois-parser"
	"github.com/projectdiscovery/retryabledns"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/semaphore"
)

type registryType int

const (
	RegistryNPM  registryType = 1
	RegistryPypi registryType = 2
)

type app struct {
	httpClient               *http.Client
	dnsClient                *retryabledns.Client
	ctx                      context.Context
	muMX                     sync.RWMutex
	alreadyCheckedMX         map[string][]string
	muNS                     sync.RWMutex
	alreadyCheckedNS         map[string][]string
	muRS                     sync.RWMutex
	alreadyCheckedRecords    map[string][]string
	muWhois                  sync.RWMutex
	alreadyCheckedWhois      map[string]*whoisparser.WhoisInfo
	muWhoisError             sync.RWMutex
	alreadyCheckedWhoisError map[string]error
	log                      *logrus.Logger
}

func main() {
	log := logrus.New()
	log.Level = logrus.InfoLevel
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	commonFlags := []cli.Flag{
		&cli.BoolFlag{Name: "debug", Aliases: []string{"d"}, Value: false, Usage: "enable debug output"},
		&cli.IntFlag{Name: "retries", Value: 3, Usage: "dns retries"},
		&cli.StringSliceFlag{Name: "dnsserver", Value: cli.NewStringSlice("1.1.1.1:53", "8.8.8.8:53", "8.8.4.4:53", "1.0.0.1:53", "208.67.222.222:53", "208.67.220.220:53"), Usage: "dns servers to use"},
		&cli.BoolFlag{Name: "expired", Value: false, Usage: "list domains that expire in 7 days"},
		&cli.Int64Flag{Name: "threads", Value: 10, Usage: "number of parallel checks"},
		&cli.StringFlag{Name: "log", Value: "output.txt", Usage: "logfile for output"},
	}

	app := &cli.App{
		Name:  "hijagger",
		Usage: "check package registries for hijackable packages",
		Authors: []*cli.Author{
			{
				Name:  "Christian Mehlmauer",
				Email: "firefart@gmail.com",
			},
		},
		Commands: []*cli.Command{
			{
				Name:        "npm",
				Usage:       "runs against the npm registry",
				Description: "Checks the npm registry for hijackable packages",
				Flags: append(commonFlags,
					&cli.StringFlag{Name: "localfile", Value: "_all_docs", Usage: "downloaded package index"},
				),
				Before: func(ctx *cli.Context) error {
					if ctx.Bool("debug") {
						log.SetLevel(logrus.DebugLevel)
					}
					logFile := ctx.String("log")
					logfileHandle, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0o644)
					if err != nil {
						log.Fatalf("could not open %s: %v", logFile, err)
					}

					mw := io.MultiWriter(os.Stdout, logfileHandle)
					log.SetOutput(mw)
					return nil
				},
				Action: func(c *cli.Context) error {
					localFile := c.String("localfile")
					threads := c.Int64("threads")
					dnsServers := c.StringSlice("dnsserver")
					dnsRetries := c.Int("retries")
					checkExpired := c.Bool("expired")
					return run(log,
						RegistryNPM,
						localFile,
						threads,
						dnsServers,
						dnsRetries,
						checkExpired,
					)
				},
			},
			{
				Name:        "pypi",
				Usage:       "runs against the pypi registry",
				Description: "Checks the python pypi registry for hijackable packages",
				Flags:       commonFlags,
				Before: func(ctx *cli.Context) error {
					if ctx.Bool("debug") {
						log.SetLevel(logrus.DebugLevel)
					}
					logFile := ctx.String("log")
					logfileHandle, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE, 0o644)
					if err != nil {
						log.Fatalf("could not open %s: %v", logFile, err)
					}

					mw := io.MultiWriter(os.Stdout, logfileHandle)
					log.SetOutput(mw)
					return nil
				},
				Action: func(c *cli.Context) error {
					localFile := c.String("localfile")
					threads := c.Int64("threads")
					dnsServers := c.StringSlice("dnsserver")
					dnsRetries := c.Int("retries")
					checkExpired := c.Bool("expired")
					return run(log,
						RegistryPypi,
						localFile,
						threads,
						dnsServers,
						dnsRetries,
						checkExpired,
					)
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(log *logrus.Logger, registryType registryType, localFile string, threads int64, dnsServers []string, dnsRetries int, checkExpired bool) error {
	httpClient := http.Client{
		Timeout: 1 * time.Minute,
	}

	var err error

	dnsClient, err := retryabledns.New(dnsServers, dnsRetries)
	if err != nil {
		return err
	}

	app := app{
		httpClient:               &httpClient,
		dnsClient:                dnsClient,
		ctx:                      context.Background(),
		alreadyCheckedMX:         make(map[string][]string),
		alreadyCheckedNS:         make(map[string][]string),
		alreadyCheckedRecords:    make(map[string][]string),
		alreadyCheckedWhois:      make(map[string]*whoisparser.WhoisInfo),
		alreadyCheckedWhoisError: make(map[string]error),
		log:                      log,
	}

	var packages []string
	switch registryType {
	case RegistryNPM:
		log.Infof("loading all packages")
		packages, err = app.npmGetAllPackageNames(localFile)
		if err != nil {
			return err
		}
	case RegistryPypi:
		packages, err = app.pypiGetAllPackageNames(localFile)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid registry type")
	}

	lenPackages := len(packages)
	log.Infof("got %d packages", lenPackages)

	sem := semaphore.NewWeighted(threads)
	for i, p := range packages {
		if i%1000 == 0 {
			log.Infof("%d / %d", i, lenPackages)
		}

		// Acquire only returns context related errors so safe to ignore
		if err := sem.Acquire(context.Background(), 1); err != nil {
			continue
		}
		go func(p string) {
			if err := app.checkPackage(registryType, p, checkExpired); err != nil {
				app.log.WithError(err).Error("")
			}
			sem.Release(1)
		}(p)
	}

	return nil
}
