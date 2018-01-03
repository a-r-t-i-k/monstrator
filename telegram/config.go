package main

import (
	"encoding/json"
	"io"
)

const defaultConfigPath string = "configuration.json"

var config *configuration

type configuration struct {
	// TODO: Add customizability of log targets.
	Token                string   `json:"token"`
	Address              string   `json:"address"`
	ReadTimeout          duration `json:"readTimeout"`
	WriteTimeout         duration `json:"writeTimeout"`
	InlineQueryCacheTime duration `json:"inlineQueryCacheTime"`
	TLS                  struct {
		Certificate string `json:"certificate"`
		Key         string `json:"key"`
	} `json:"TLS"`
	Shorteners struct {
		Google struct {
			APIKey  string   `json:"APIKey"`
			Timeout duration `json:"timeout"`
		} `json:"Google"`
		Isgd struct {
			Timeout duration `json:"timeout"`
		} `json:"isgd"`
		TinyURL struct {
			Timeout duration `json:"timeout"`
		} `json:"TinyURL"`
	} `json:"shorteners"`
}

func readJSON(r io.Reader) error {
	dec := json.NewDecoder(r)
	config = new(configuration)
	err := dec.Decode(config)
	return err
}

// TODO: Implement configuration from environment variables.
