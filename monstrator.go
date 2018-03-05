// Package monstrator provides abstraction over URL shorteners.
package monstrator

import (
	"net/url"
)

// Shortener is the interface all URL shorteners implement.
type Shortener interface {
	Shorten(longURL *url.URL) (*url.URL, error)
	Expand(shortURL *url.URL) (*url.URL, error)
	IsShortenedURL(u *url.URL) bool
}
