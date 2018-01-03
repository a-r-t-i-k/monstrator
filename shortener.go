package monstrator

import (
	"net/http"
	"net/url"
)

// Shortener is the interface all URL shortening services implement.
type Shortener interface {
	Shorten(longURL *url.URL) (*url.URL, error)
	Expand(shortURL *url.URL) (*url.URL, error)
	IsShortURL(u *url.URL) bool
}

// baseShortener implements common shortener facilities.
type baseShortener struct {
	Client *http.Client
}

func (shortener *baseShortener) client() *http.Client {
	if shortener.Client == nil {
		return http.DefaultClient
	}
	return shortener.Client
}
