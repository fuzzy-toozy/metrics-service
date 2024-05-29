// Package http Client for sending http requests
package http

import (
	"net/http"
	"time"
)

const DefaultClientTimeout = 30

// HTTPClient http client interface to send metrics to server
type HTTPClient interface {
	Send(r *http.Request) (*http.Response, error)
	SetTransport(t http.RoundTripper)
}

// DefaultHTTPClient default HTTPClient implementation
type DefaultHTTPClient struct {
	client http.Client
}

// Send sends http request to server
func (c *DefaultHTTPClient) Send(r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}

func (c *DefaultHTTPClient) SetTransport(t http.RoundTripper) {
	c.client.Transport = t
}

func NewDefaultHTTPClient() *DefaultHTTPClient {
	c := DefaultHTTPClient{client: http.Client{
		Timeout: DefaultClientTimeout * time.Second,
	}}
	return &c
}
