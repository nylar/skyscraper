package skyscraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mockServer(body []byte, code int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write(body)
	}))
}

func TestScraper_StatusError_Error(t *testing.T) {
	se := StatusError{
		Code: http.StatusTeapot,
		Err:  fmt.Errorf("This isn't coffee"),
	}

	assert.Equal(t, "This isn't coffee, got 418 status code", se.Error())
}

func TestScraper_Scraper_Process(t *testing.T) {
	scraper := New(1)
	defer scraper.Close()

	data := []byte("Hello, World")

	ms := mockServer(data, http.StatusOK)

	go scraper.Process()

	scraper.Add(ms.URL)

	response := <-scraper.Out

	assert.NoError(t, response.Err)
	assert.Equal(t, ms.URL, response.Domain)
}

func TestScraper_Scraper_GracefulClose(t *testing.T) {
	scraper := New(1)

	go scraper.Process()

	scraper.Close()

	assert.NotPanics(t, func() {
		scraper.Add("test")
	})
}

func TestScraper_Scraper_process_GetError(t *testing.T) {
	scraper := New(0)

	go scraper.process("invalid-domain")

	response := <-scraper.Out

	assert.Error(t, response.Err)
}

func TestScraper_Scraper_process_NonSuccessfulResponseError(t *testing.T) {
	scraper := New(0)

	ms := mockServer([]byte(""), http.StatusTeapot)

	go scraper.process(ms.URL)

	response := <-scraper.Out

	err, ok := response.Err.(StatusError)

	assert.Error(t, response.Err)
	assert.Equal(t, ErrNoSuccessfulResponse, err.Err)
	assert.Equal(t, http.StatusTeapot, err.Code)
	assert.True(t, ok)
}
