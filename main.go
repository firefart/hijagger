package main

import (
	"context"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	whoisparser "github.com/likexian/whois-parser"
	"github.com/projectdiscovery/retryabledns"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
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
	localFile := flag.String("localfile", "_all_docs", "downloaded package index")
	dnsRetries := flag.Int("retries", 3, "dns retries")
	dnsServers := flag.String("dnsserver", "1.1.1.1:53,8.8.8.8:53,8.8.4.4:53,1.0.0.1:53,208.67.222.222:53,208.67.220.220:53", "dns servers to use, separate with ,")
	checkExpired := flag.Bool("expired", false, "list domains that expire in 7 days")
	debug := flag.Bool("debug", false, "enable debug output")
	threads := flag.Int64("threads", 10, "number of parallel checks")
	logfile := flag.String("log", "output.txt", "logfile for output")
	flag.Parse()

	log := logrus.New()

	logfileHandle, err := os.OpenFile(*logfile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("could not open %s: %v", *logfile, err)
	}

	mw := io.MultiWriter(os.Stdout, logfileHandle)
	log.SetOutput(mw)
	log.Level = logrus.InfoLevel
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	if *debug {
		log.Level = logrus.DebugLevel
	}

	dnsServersSlice := strings.Split(*dnsServers, ",")

	if err := run(log, *localFile, *threads, dnsServersSlice, *dnsRetries, *checkExpired); err != nil {
		log.Fatalf("[ERROR]: %v", err)
	}
}

func run(log *logrus.Logger, localFile string, threads int64, dnsServers []string, dnsRetries int, checkExpired bool) error {
	httpClient := http.Client{
		Timeout: 1 * time.Minute,
	}

	dnsClient := retryabledns.New(dnsServers, dnsRetries)

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

	log.Infof("loading all packages")
	packages, err := app.getAllPackageNames(localFile)
	if err != nil {
		return err
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
			if err := app.checkPackage(p, checkExpired); err != nil {
				app.log.WithError(err).Error("")
			}
			sem.Release(1)
		}(p)
	}

	return nil
}
