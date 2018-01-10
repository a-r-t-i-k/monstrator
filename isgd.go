package monstrator

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var isgdShortenEndpoint *url.URL
var isgdExpandEndpoint *url.URL

func init() {
	var err error
	isgdShortenEndpoint, err = url.Parse("https://is.gd/create.php?format=simple")
	if err != nil {
		panic(err)
	}
	isgdExpandEndpoint, err = url.Parse("https://is.gd/forward.php?format=simple")
	if err != nil {
		panic(err)
	}
}

func trimIsgdShortenerErrorMessagePrefix(message string) string {
	return strings.TrimPrefix(message, "Error: ")
}

// IsgdShortener communicates with the is.gd URL shortener API.
// If Client is nil, http.DefaultClient will be used.
type IsgdShortener struct {
	*baseShortener
}

// Shorten requests the shortened URL.
func (shortener *IsgdShortener) Shorten(longURL *url.URL) (*url.URL, error) {
	u := isgdShortenEndpoint
	query := u.Query()
	query.Add("url", longURL.String())
	u.RawQuery = query.Encode()

	resp, err := shortener.client().Get(u.String())
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
	u := isgdExpandEndpoint
	query := u.Query()
	query.Add("shorturl", shortenedURL.String())
	u.RawQuery = query.Encode()

	resp, err := shortener.client().Get(u.String())
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

// Name returns the name of IsgdShortener.
func (*IsgdShortener) Name() string {
	return "is.gd"
}

// NewIsgdShortener returns an initialized IsgdShortener instance.
func NewIsgdShortener(client *http.Client) *IsgdShortener {
	return &IsgdShortener{&baseShortener{client}}
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
