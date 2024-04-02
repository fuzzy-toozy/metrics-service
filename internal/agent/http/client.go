package http

import (
	"net/http"
	"time"
)

// HTTPClient http client interface to send metrics to server
type HTTPClient interface {
	Send(r *http.Request) (*http.Response, error)
}

// DefaultHTTPClient default HTTPClient implementation
type DefaultHTTPClient struct {
	client http.Client
}

// Send sends http request to server
func (c *DefaultHTTPClient) Send(r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}

func NewDefaultHTTPClient() *DefaultHTTPClient {
	c := DefaultHTTPClient{client: http.Client{
		Timeout: 30 * time.Second,
	}}
	return &c
}
