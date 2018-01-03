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
// If Client is nil, the DefaultClient will be used.
type IsgdShortener struct {
	*baseShortener
}

// Shorten requests the short URL of longURL.
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

	body := readBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, &IsgdShortenerError{resp.StatusCode, trimIsgdShortenerErrorMessagePrefix(body)}
	}
	shortURL, err := url.Parse(body)
	if err != nil {
		return nil, err
	}
	return shortURL, nil
}

// Expand determines the long URL of shortURL.
func (shortener *IsgdShortener) Expand(shortURL *url.URL) (*url.URL, error) {
	u := isgdExpandEndpoint
	query := u.Query()
	query.Add("shorturl", shortURL.String())
	u.RawQuery = query.Encode()

	resp, err := shortener.client().Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body := readBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, &IsgdShortenerError{resp.StatusCode, trimIsgdShortenerErrorMessagePrefix(body)}
	}
	longURL, err := url.Parse(body)
	if err != nil {
		return nil, err
	}
	return longURL, nil
}

// IsShortURL determines whether the URL is shortened.
func (*IsgdShortener) IsShortURL(u *url.URL) bool {
	return u.Hostname() == "is.gd"
}

// NewIsgdShortener returns an with client initilized IsgdShortener.
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
		return fmt.Sprintf("Isgd URl shortener reported failure: %s", e.Message)
	}
	return "Isgd URl shortener reported failure"
}
