package monstrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

var googleEndpoint *url.URL

func init() {
	var err error
	googleEndpoint, err = url.Parse("https://www.googleapis.com/urlshortener/v1/url")
	if err != nil {
		panic(err)
	}
}

// GoogleShortener communicates with the Google URL shortener API.
// If Client is nil, the DefaultClient will be used.
type GoogleShortener struct {
	APIKey string
	*baseShortener
}

func (shortener *GoogleShortener) addAPIKey(query *url.Values) {
	if shortener.APIKey != "" {
		query.Set("key", shortener.APIKey)
	}
}

type shortenResponse struct {
	ShortURL string                `json:"id"`
	Error    *GoogleShortenerError `json:"error"`
}

type expandResponse struct {
	LongURL string                `json:"longUrl"`
	Error   *GoogleShortenerError `json:"error"`
}

// Shorten requests the short URL of longURL.
func (shortener *GoogleShortener) Shorten(longURL *url.URL) (*url.URL, error) {
	u := googleEndpoint
	query := u.Query()
	shortener.addAPIKey(&query)
	u.RawQuery = query.Encode()

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(map[string]string{"longUrl": longURL.String()})
	resp, err := shortener.client().Post(u.String(), "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var res shortenResponse
	err = dec.Decode(&res)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, res.Error
	}
	shortURL, err := url.Parse(res.ShortURL)
	if err != nil {
		return nil, err
	}
	return shortURL, nil
}

// Expand determines the long URL of shortURL.
func (shortener *GoogleShortener) Expand(shortURL *url.URL) (*url.URL, error) {
	u := googleEndpoint
	query := u.Query()
	query.Set("shortUrl", shortURL.String())
	shortener.addAPIKey(&query)
	u.RawQuery = query.Encode()

	resp, err := shortener.client().Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var res expandResponse
	err = dec.Decode(&res)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, res.Error
	}
	longURL, err := url.Parse(res.LongURL)
	if err != nil {
		return nil, err
	}
	return longURL, nil
}

// IsShortURL determines whether the URL is shortened.
func (*GoogleShortener) IsShortURL(u *url.URL) bool {
	return u.Hostname() == "goog.le"
}

// NewGoogleShortener returns an initialized GoogleShortener instance.
// http.DefaultClient will be used if client is nil.
// Google highly recommends to use an API key.
func NewGoogleShortener(apiKey string, client *http.Client) *GoogleShortener {
	return &GoogleShortener{apiKey, &baseShortener{client}}
}

// GoogleShortenerError represents an error returned by the Google URL shortener API.
type GoogleShortenerError struct {
	Code    int
	Message string
	Errors  []interface{}
}

func (e *GoogleShortenerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Google URL shortener reported failure: %s", e.Message)
	}
	return "Google URL shortener reported failure"
}
