//go:build ignore
// +build ignore

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
)

// MBRequest is a Multiple-host HTTP(s) Benchmarking tool request.
type MBRequest struct {
	Clients           int    `json:"clients"`
	Host              string `json:"host"`
	KeepAliveRequests int    `json:"keep-alive-requests"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Port              int    `json:"port"`
	Scheme            string `json:"scheme"`
	TLSSessionReuse   bool   `json:"tls-session-reuse"`
}

type Globals struct {
	Duration    time.Duration `help:"Test duration" short:"d" default:"60s"`
	RequestFile string        `help:"Request file." short:"i" type:"existingfile"`
	TLSReuse    bool          `help:"Enable TLS session reuse" default:"true"`
}

type ProgramCtx struct {
	Context context.Context
	Globals
}

type CLI struct {
	Globals
}

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

func Run(ctx *kong.Context, p *ProgramCtx) error {
	data, err := os.ReadFile(p.Globals.RequestFile)
	if err != nil {
		return fmt.Errorf("cannot open file %q: %w", p.Globals.RequestFile, err)
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
			request := <-requestCh
			resultCh <- fetch(request, client)
		}
	}

	pendingRequests := []*http.Request{}

	for i := 0; i < requests[0].Clients; i++ {
		go fetcher(newHTTPClient())
		for j := range requests {
			url := fmt.Sprintf("%v://%v:%v%v",
				requests[j].Scheme,
				requests[j].Host,
				requests[j].Port,
				requests[j].Path)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			pendingRequests = append(pendingRequests, req)
		}
	}

	fetchErrors := 0
	fetchBadStatus := 0
	hits := 0
	progressTicker := time.Tick(1 * time.Second)
	testComplete := time.After(60 * time.Second)

	for {
		var sendCh chan<- *http.Request
		var link *http.Request

		if len(pendingRequests) > 0 {
			sendCh = requestCh
			link = pendingRequests[0]
		}

		select {
		case <-p.Context.Done():
			return errors.New("test interrupted")

		case <-testComplete:
			log.Printf("hits: %v errors: %v bad_status: %v request/s: %.0f", hits, fetchErrors, fetchBadStatus, float64(hits)/float64(p.Globals.Duration.Seconds()))
			return nil

		case <-progressTicker:
			log.Printf("hits: %v errors: %v", hits, fetchErrors)

		case sendCh <- link:
			pendingRequests = pendingRequests[1:]

		case result := <-resultCh:
			pendingRequests = append(pendingRequests, result.req)
			hits += 1 // should we record a hit if there was an error?
			if result.err != nil {
				fetchErrors += 1
				log.Printf("%s %q failed: %v", result.req.Method, result.req.URL, result.err)
				continue
			}
			if result.resp.StatusCode != http.StatusOK {
				fetchBadStatus += 1
				log.Printf("%s %q bad_status: %v", result.req.Method, result.req.URL, result.resp.StatusCode)
			}
			io.Copy(io.Discard, result.resp.Body)
			result.resp.Body.Close()
		}
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	cli := CLI{
		Globals: Globals{},
	}

	ktx := kong.Parse(&cli,
		kong.Name("gomb"),
		kong.Description("An incomplete implementation of https://github.com/jmencak/mb"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		kong.Vars{
			"version": "0.0.1",
		},
	)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	p := &ProgramCtx{Globals: cli.Globals, Context: signalCtx}
	if err := Run(ktx, p); err != nil {
		log.Fatal(err)
	}
}
