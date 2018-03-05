package monstrator

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func trimIsgdShortenerErrorMessagePrefix(message string) string {
	return strings.TrimPrefix(message, "Error: ")
}

// IsgdShortener communicates with the is.gd URL shortener API.
type IsgdShortener struct {
	Client *http.Client
}

// Shorten requests the shortened URL.
func (shortener *IsgdShortener) Shorten(longURL *url.URL) (*url.URL, error) {
	query := url.Values{"format": []string{"simple"}, "url": []string{longURL.String()}}
	endpoint := &url.URL{Scheme: "https", Host: "is.gd", Path: "/create.php", RawQuery: query.Encode()}

	resp, err := shortener.Client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &IsgdShortenerError{resp.StatusCode, trimIsgdShortenerErrorMessagePrefix(body)}
	}
	shortenedURL, err := url.Parse(body)
	if err != nil {
		return nil, err
	}
	if !shortener.IsShortenedURL(shortenedURL) {
		return nil, NotShortenedURLError{shortenedURL}
	}
	return shortenedURL, nil
}

// Expand requests the long URL.
func (shortener *IsgdShortener) Expand(shortenedURL *url.URL) (*url.URL, error) {
	if !shortener.IsShortenedURL(shortenedURL) {
		return nil, NotShortenedURLError{shortenedURL}
	}
	query := url.Values{"format": []string{"simple"}, "shorturl": []string{shortenedURL.String()}}
	endpoint := &url.URL{Scheme: "https", Host: "is.gd", Path: "/forward.php", RawQuery: query.Encode()}

	resp, err := shortener.Client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &IsgdShortenerError{resp.StatusCode, trimIsgdShortenerErrorMessagePrefix(body)}
	}
	longURL, err := url.Parse(body)
	if err != nil {
		return nil, err
	}
	return longURL, nil
}

// IsShortenedURL determines whether the URL is shortened.
func (*IsgdShortener) IsShortenedURL(u *url.URL) bool {
	return u.IsAbs() && hasHTTPScheme(u) && u.Hostname() == "is.gd" && len(u.Path) > 1
}

// NewIsgdShortener returns an initialized IsgdShortener instance.
// If client is nil, http.DefaultClient will be used.
func NewIsgdShortener(client *http.Client) *IsgdShortener {
	if client == nil {
		return &IsgdShortener{Client: http.DefaultClient}
	}
	return &IsgdShortener{Client: client}
}

// IsgdShortenerError represents an error returned by the is.gd URL shortener.
type IsgdShortenerError struct {
	Code    int
	Message string
}

func (e *IsgdShortenerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("shortener reported failure: %s", e.Message)
	}
	return "shortener reported failure"
}
