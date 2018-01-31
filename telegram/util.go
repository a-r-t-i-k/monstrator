package main

import (
	"encoding/json"
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

// isDomainName checks if a string is a presentation-format domain name.
// Currently only length is considered because of the many edge cases.
func isDomainName(s string) bool {
	if len(s) == 0 || len(s) > 255 {
		return false
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) > 63 {
			return false
		}
	}
	return true
}
