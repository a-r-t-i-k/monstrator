package monstrator

import (
	"fmt"
	"net/url"
)

// NotShortenedURLError respresents a not shortened URL.
type NotShortenedURLError struct {
	URL *url.URL
}

func (e NotShortenedURLError) Error() string {
	return fmt.Sprintf("not shortened URL: %v", e.URL)
}
