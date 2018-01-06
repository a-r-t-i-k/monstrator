package main

import (
	"encoding"
	"encoding/json"
	"io"
	"os"
	"reflect"
)

var config struct {
	Token                string   `json:"token" env:"TOKEN"`
	Address              string   `json:"address" env:"ADDRESS"`
	ReadTimeout          duration `json:"readTimeout" env:"READ_TIMEOUT"`
	WriteTimeout         duration `json:"writeTimeout" env:"WRITE_TIMEOUT"`
	InlineQueryCacheTime duration `json:"inlineQueryCacheTime" env:"INLINE_QUERY_CACHE_TIME"`
	TLS                  struct {
		Certificate string `json:"certificate" env:"TLS_CERTIFICATE"`
		Key         string `json:"key" env:"TLS_KEY"`
	} `json:"TLS"`
	Shorteners struct {
		Google struct {
			APIKey  string   `json:"APIKey" env:"GOOGLE_SHORTENER_API_KEY"`
			Timeout duration `json:"timeout" env:"GOOGLE_SHORTENER_TIMEOUT"`
		} `json:"Google"`
		Isgd struct {
			Timeout duration `json:"timeout" env:"ISGD_SHORTENER_TIMEOUT"`
		} `json:"isgd"`
		TinyURL struct {
			Timeout duration `json:"timeout" env:"TINYURL_SHORTENER_TIMEOUT"`
		} `json:"TinyURL"`
	} `json:"shorteners"`
}

func loadConfigFromJSON(r io.Reader) error {
	dec := json.NewDecoder(r)
	err := dec.Decode(config)
	return err
}

func loadConfigFromEnv() error {
	var unmarshal func(structValue reflect.Value) error
	unmarshal = func(structValue reflect.Value) error {
		structType := structValue.Type()
		for i := 0; i < structValue.NumField(); i++ {
			fieldValue := structValue.Field(i)
			field := structType.Field(i)
			kind := field.Type.Kind()
			if !fieldValue.CanSet() {
				continue
			}
			if key, ok := field.Tag.Lookup("env"); ok {
				if value, ok := os.LookupEnv(key); ok {
					switch kind {
					case reflect.String:
						fieldValue.SetString(value)
					default:
						if u, ok := fieldValue.Addr().Interface().(encoding.TextUnmarshaler); ok {
							err := u.UnmarshalText([]byte(value))
							if err != nil {
								return err
							}
						} else {
							return &envUnmarshalTypeError{structType, field}
						}
					}
				}
			} else if kind == reflect.Struct {
				return unmarshal(fieldValue)
			}
		}
		return nil
	}
	return unmarshal(reflect.ValueOf(config))
}

type envUnmarshalTypeError struct {
	Struct reflect.Type
	Field  reflect.StructField
}

func (e *envUnmarshalTypeError) Error() string {
	return "cannot unmarshal environment variable into Go struct field " + e.Struct.Name() + "." + e.Field.Name + " of type " + e.Field.Type.String()
}
