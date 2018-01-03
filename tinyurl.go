package monstrator

import (
	"fmt"
	"net/http"
	"net/url"
)

var tinyURLShortenEndpoint *url.URL

func init() {
	var err error
	tinyURLShortenEndpoint, err = url.Parse("http://tinyurl.com/api-create.php")
	if err != nil {
		panic(err)
	}
}

// TinyURLShortener communicates with the TinyURL shortener.
// If Client is nil, the DefaultClient will be used.
type TinyURLShortener struct {
	*baseShortener
}

// Shorten requests the short URL of longURL.
func (shortener *TinyURLShortener) Shorten(longURL *url.URL) (*url.URL, error) {
	u := tinyURLShortenEndpoint
	query := url.Values{}
	query.Add("url", longURL.String())
	u.RawQuery = query.Encode()

	resp, err := shortener.client().Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, TinyURLShortenerError(resp.StatusCode)
	}
	body := readBody(resp)
	shortURL, err := url.Parse(body)
	if err != nil {
		return nil, err
	}
	// HTTPS is supported but not utilized by default.
	shortURL.Scheme = "https"
	return shortURL, nil
}

// Expand determines the long URL of shortURL. There is no API endpoint for expanding and the Location header is therefore utilized.
func (shortener *TinyURLShortener) Expand(shortURL *url.URL) (*url.URL, error) {
	resp, err := shortener.client().Get(shortURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMovedPermanently {
		return nil, TinyURLShortenerError(resp.StatusCode)
	}
	longURL, err := resp.Location()
	if err != nil {
		return nil, err
	}
	return longURL, nil
}

// IsShortURL determines whether the URL is shortened.
func (*TinyURLShortener) IsShortURL(u *url.URL) bool {
	return u.Hostname() == "tinyurl.com"
}

func NewTinyURLShortener(client *http.Client) *TinyURLShortener {
	return &TinyURLShortener{&baseShortener{client}}
}

// TinyURLShortenerError represents an error status code returned by the TinyURL shortener.
type TinyURLShortenerError int

func (e TinyURLShortenerError) Error() string {
	return fmt.Sprintf("TinyURL URL shortener reported failure (%d)", e)
}
