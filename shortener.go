// Package monstrator provides abstraction over URL shorteners.
package monstrator

import (
	"fmt"
	"net/http"
	"net/url"
)

// Shortener is the interface all URL shorteners implement.
type Shortener interface {
	Shorten(longURL *url.URL) (*url.URL, error)
	Expand(shortURL *url.URL) (*url.URL, error)
	IsShortenedURL(u *url.URL) bool
	Name() string
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

// NotShortenedURLError respresents a not shortened URL.
type NotShortenedURLError struct {
	URL *url.URL
}

func (e NotShortenedURLError) Error() string {
	return fmt.Sprintf("not shortened URL: %v", e.URL)
}
