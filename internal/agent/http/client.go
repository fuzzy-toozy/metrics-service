package http

import (
	"net/http"
	"time"
)

type HTTPClient interface {
	Send(r *http.Request) (*http.Response, error)
}

type DefaultHTTPClient struct {
	client http.Client
}

func (c *DefaultHTTPClient) Send(r *http.Request) (*http.Response, error) {
	return c.client.Do(r)
}

func NewDefaultHTTPClient() *DefaultHTTPClient {
	c := DefaultHTTPClient{client: http.Client{
		Timeout: 30 * time.Second,
	}}
	return &c
}
