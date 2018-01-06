package monstrator

import "net/http"
import "io"
import "io/ioutil"

func readBody(resp *http.Response) (string, error) {
	if resp.ContentLength > -1 {
		body := make([]byte, 0, resp.ContentLength)
		_, err := io.ReadFull(resp.Body, body)
		return string(body), err
	}
	// Content length is unknown.
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}
