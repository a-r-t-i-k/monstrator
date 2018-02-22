package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// duration embeds time.Duration but implements encoding.TextUnmarshaler and json.Unmarshaler.
type duration struct {
	time.Duration
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	var err error
	d.Duration, err = time.ParseDuration(s)
	return err
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

func assembleAbsoluteURL(rel *url.URL, r *http.Request) *url.URL {
	u := new(url.URL)
	*u = *rel

	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		u.Scheme = proto
	}
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		u.Host = host
	} else if r.Host != "" {
		u.Host = r.Host
	}

	if !u.IsAbs() {
		return nil
	}
	return u
}

// isDomainName checks if a string is a presentation-format domain name.
// Currently only length is considered due to the many edge cases.
func isDomainName(s string) bool {
	if s == "" || len(s) > 255 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
	}
	return true
}
