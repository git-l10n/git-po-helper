package util

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func httpClient() *http.Client {
	var (
		timeout = time.Duration(10)
	)

	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeout * time.Second,
			KeepAlive: timeout * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout:   timeout * time.Second,
		ResponseHeaderTimeout: timeout * time.Second,
		ExpectContinueTimeout: timeout * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       timeout * time.Second,
		DisableCompression:    true,
		Proxy:                 http.ProxyFromEnvironment,
	}

	return &http.Client{Transport: tr}
}

func httpDownload(URL string, fileName string, showProgress bool) error {
	var (
		done = make(chan int, 1)
		wg   sync.WaitGroup
	)

	log.Debugf("will download %s to %s", URL, fileName)

	client := httpClient()
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("cannot access %s (status: %d)", URL, resp.StatusCode)
	}

	f, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if showProgress {
		contentLength, err := strconv.Atoi(resp.Header.Get("Content-Length"))
		if err != nil {
			log.Debugf("fail to get content-length: %s", err)
			contentLength = 0
		} else {
			log.Debugf("content-length for %s: %d", URL, contentLength)
		}

		wg.Add(1)
		go func(fullpath string, total int) {
			var (
				maxWidth = 60
				percent  float64
				stop     bool
				name     = filepath.Base(URL)
			)

			defer wg.Done()

			if total == 0 {
				return
			}

			for {
				select {
				case <-done:
					stop = true
				default:
				}

				if !stop {
					fi, err := os.Stat(fullpath)
					if err != nil {
						stop = true
						break
					}
					size := fi.Size()
					if size == 0 {
						size = 1
					}
					percent = float64(size) / float64(total) * 100
				} else {
					percent = 100
				}

				fmt.Printf("Download %s: %s %3.0f%%\r",
					name,
					strings.Repeat("#", int(percent/100*float64(maxWidth))),
					percent)

				if stop {
					break
				}

				time.Sleep(time.Second)
			}
			fmt.Printf("\n")
		}(fileName, contentLength)
	}

	_, err = io.Copy(f, resp.Body)
	if showProgress {
		done <- 1
		wg.Wait()
	}
	return err
}
