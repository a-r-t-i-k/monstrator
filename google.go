package monstrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GoogleShortener communicates with the Google URL shortener API.
type GoogleShortener struct {
	APIKey string
	Client *http.Client
}

func (shortener *GoogleShortener) addAPIKey(query *url.Values) {
	if shortener.APIKey != "" {
		query.Set("key", shortener.APIKey)
	}
}

// Shorten requests the shortened URL.
func (shortener *GoogleShortener) Shorten(longURL *url.URL) (*url.URL, error) {
	query := url.Values{}
	shortener.addAPIKey(&query)
	endpoint := &url.URL{Scheme: "https", Host: "www.googleapis.com", Path: "/urlshortener/v1/url", RawQuery: query.Encode()}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(map[string]string{"longUrl": longURL.String()})

	resp, err := shortener.Client.Post(endpoint.String(), "application/json", buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var res struct {
		ShortenedURL string                `json:"id"`
		Error        *GoogleShortenerError `json:"error"`
	}
	err = dec.Decode(&res)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, res.Error
	}
	shortenedURL, err := url.Parse(res.ShortenedURL)
	if err != nil {
		return nil, err
	}
	if !shortener.IsShortenedURL(shortenedURL) {
		return nil, NotShortenedURLError{shortenedURL}
	}
	return shortenedURL, nil
}

// Expand requests the long URL.
func (shortener *GoogleShortener) Expand(shortenedURL *url.URL) (*url.URL, error) {
	if !shortener.IsShortenedURL(shortenedURL) {
		return nil, NotShortenedURLError{shortenedURL}
	}
	query := url.Values{"shortUrl": []string{shortenedURL.String()}}
	shortener.addAPIKey(&query)
	endpoint := &url.URL{Scheme: "https", Host: "www.googleapis.com", Path: "/urlshortener/v1/url", RawQuery: query.Encode()}

	resp, err := shortener.Client.Get(endpoint.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var res struct {
		LongURL string                `json:"longUrl"`
		Error   *GoogleShortenerError `json:"error"`
	}
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

// IsShortenedURL determines whether the URL is shortened.
func (*GoogleShortener) IsShortenedURL(u *url.URL) bool {
	return u.IsAbs() && hasHTTPScheme(u) && u.Hostname() == "goo.gl" && len(u.Path) > 4
}

// NewGoogleShortener returns an initialized GoogleShortener instance.
// Google highly recommends to use an API key.
// If client is nil, http.DefaultClient will be used.
func NewGoogleShortener(apiKey string, client *http.Client) *GoogleShortener {
	if client == nil {
		return &GoogleShortener{APIKey: apiKey, Client: http.DefaultClient}
	}
	return &GoogleShortener{APIKey: apiKey, Client: client}
}

// GoogleShortenerError represents an error returned by the Google URL shortener API.
type GoogleShortenerError struct {
	Code    int
	Message string
	Errors  []interface{}
}

func (e *GoogleShortenerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("shortener reported failure: %s", e.Message)
	}
	return "shortener reported failure"
}
