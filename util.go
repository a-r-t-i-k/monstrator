package monstrator

import "net/http"
import "io"

func readBody(resp *http.Response) string {
	body := make([]byte, 0, resp.ContentLength)
	io.ReadFull(resp.Body, body)
	return string(body)
}
