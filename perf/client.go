package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type fetchResult struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          0, // no limit
			MaxIdleConnsPerHost:   0, // no limit
			MaxConnsPerHost:       0, // no limit
			DisableKeepAlives:     false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func (c *TestCmd) Run(p *ProgramCtx) error {
	data, err := os.ReadFile(c.RequestFile)
	if err != nil {
		return err
	}

	var requests []MBRequest
	if err := json.Unmarshal(data, &requests); err != nil {
		return err
	}
	if len(requests) == 0 {
		return nil
	}

	resultCh := make(chan *fetchResult)
	requestCh := make(chan *http.Request)

	fetch := func(req *http.Request, client *http.Client) *fetchResult {
		result := &fetchResult{req: req}
		result.resp, result.err = client.Do(req)
		return result
	}

	fetcher := func(client *http.Client) {
		for {
			select {
			case request := <-requestCh:
				resultCh <- fetch(request, client)
			}
		}
	}

	port := func(scheme string) int {
		switch scheme {
		case "http":
			return p.HTTPPort
		default:
			return p.HTTPSPort
		}
	}

	pendingRequests := []*http.Request{}

	for i := 0; i < requests[0].Clients; i++ {
		go fetcher(newHTTPClient())
		for j := range requests {
			url := fmt.Sprintf("%v://%v:%v%v",
				requests[j].Scheme,
				requests[j].Host,
				port(requests[j].Scheme),
				requests[j].Path)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			pendingRequests = append(pendingRequests, req)
		}
	}

	fetchErrors := 0
	hits := 0
	progressTicker := time.Tick(1 * time.Second)
	testComplete := time.After(c.Duration)

	for {
		select {
		case <-p.Context.Done():
			return errors.New("test interrupted")

		case <-testComplete:
			log.Printf("hits: %v errors: %v request/s: %.0f", hits, fetchErrors, float64(hits)/float64(c.Duration.Seconds()))
			return nil

		case <-progressTicker:
			log.Printf("hits: %v errors: %v", hits, fetchErrors)

		case requestCh <- pendingRequests[0]:
			pendingRequests = pendingRequests[1:]

		case result := <-resultCh:
			hits += 1 // should we record a hit if there was an error?
			if result.err != nil {
				fetchErrors += 1
				log.Printf("%s %q failed: %v", result.req.Method, result.req.URL, result.err)
			} else {
				io.Copy(ioutil.Discard, result.resp.Body)
				result.resp.Body.Close()
			}
			pendingRequests = append(pendingRequests, result.req)
		}
	}
}
