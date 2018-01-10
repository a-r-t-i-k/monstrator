package monstrator

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

func readBody(resp *http.Response) (string, error) {
	if resp.ContentLength > -1 {
		body := make([]byte, 0, resp.ContentLength)
		_, err := io.ReadFull(resp.Body, body)
		return string(body), err
	}
	// Content length is unknown.
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}

// hasHTTPScheme determines whether the URL has a HTTP or HTTPS scheme.
func hasHTTPScheme(u *url.URL) bool {
	return u.Scheme == "http" || u.Scheme == "https"
}
