package main

import (
	"net/http"
	"net/http/cookiejar"
	"time"
)

var cookieJar, _ = cookiejar.New(nil)
var httpClient = http.Client{
	Transport: &customAgentTransport{http.Transport{}, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36"},
	Timeout:   60 * time.Second,
	Jar:       cookieJar,
}

type customAgentTransport struct {
	http.Transport
	agent string
}

func (t *customAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("user-agent", t.agent)
	return t.Transport.RoundTrip(r)
}
