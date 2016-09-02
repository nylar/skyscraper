package skyscraper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxSize        = 1 << 22 // 4mb
	capacity       = 100
	requestTimeout = time.Second * 10
)

// ErrNoSuccessfulResponse indicates than a non 2/3XX status code was received
var ErrNoSuccessfulResponse = errors.New("Received non successful response")

// Error shadows the built in error interface
type Error interface {
	Error() string
}

// StatusError returns an error and a corresponding status code
type StatusError struct {
	Code int
	Err  error
}

func (se StatusError) Error() string {
	return fmt.Sprintf("%s, got %d status code", se.Err, se.Code)
}

// Response is returned by the scraper, a non-nil error indicates that
// something went wrong and should be checked before inspecting the Body
type Response struct {
	Domain     string
	Body       io.Reader
	Err        Error
	StatusCode int
}

// Scraper represents a web scraper
type Scraper struct {
	delay   time.Duration
	domains chan string
	client  *http.Client
	stop    chan struct{}
	Out     chan *Response
	wg      *sync.WaitGroup
}

// New returns an instantiated web scraper
func New(workers int) *Scraper {
	wg := &sync.WaitGroup{}
	wg.Add(workers)

	return &Scraper{
		delay:   time.Second * 2,
		domains: make(chan string),
		client: &http.Client{
			Timeout: requestTimeout,
		},
		stop: make(chan struct{}),
		Out:  make(chan *Response),
		wg:   wg,
	}
}

// Close safely closes the scraper processors
func (s *Scraper) Close() {
	defer close(s.domains)

	log.Println("Closing")
	close(s.stop)

	s.wg.Wait()

	log.Println("Closed")
}

// Add enqueues a list of domains into the worker queue, stop if the client
// tells us to close
func (s *Scraper) Add(domains ...string) {
	for _, domain := range domains {
		select {
		case <-s.stop:
			return
		default:
			s.domains <- domain
		}
	}
}

// Process drains the domains queue and sleeps for a fixed period after
// the processing has finished. Process will block indefinitely so
// each Process call should be ran in a goroutine
func (s *Scraper) Process() {
	for {
		select {
		case <-s.stop:
			s.wg.Done()
			log.Println("Closed worker")
			return
		case domain := <-s.domains:
			s.process(domain)
			time.Sleep(s.delay)
		}
	}
}

func (s *Scraper) process(domain string) {
	response := &Response{
		Domain: domain,
	}
	defer func() { s.Out <- response }()

	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") && !strings.HasPrefix(domain, "//") {
		domain = "http://" + domain
	}

	resp, err := s.client.Get(domain)
	if err != nil {
		response.Err = err
		return
	}
	if resp.StatusCode >= http.StatusBadRequest {
		response.Err = StatusError{Err: ErrNoSuccessfulResponse, Code: resp.StatusCode}
		return
	}
	defer resp.Body.Close()

	response.StatusCode = resp.StatusCode

	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		response.Err = err
		return
	}

	response.Body = bytes.NewReader(body)
}
