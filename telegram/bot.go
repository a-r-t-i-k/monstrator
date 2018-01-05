package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/r3turnz/monstrator"
)

var (
	googleShortener  *monstrator.GoogleShortener
	tinyURLShortener *monstrator.TinyURLShortener
	isgdShortener    *monstrator.IsgdShortener
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-c path]\n", os.Args[0])
		fmt.Fprint(os.Stderr, "Runs the Monstrator Telegram bot.\n")
		flag.PrintDefaults()
	}
	var configPath string
	flag.StringVar(&configPath, "c", "", "`path` to read configuration from")
	flag.Parse()

	if configPath != "" {
		configFile, err := os.Open(configPath)
		if err != nil {
			log.Fatal(err)
		}
		err = loadConfigFromJSON(configFile)
		if err != nil {
			log.Fatalf("failed to load configuration from file: %s", err)
		}
	}
	err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to load configuration from environment variables: %s", err)
	}

	if config.Address == "" {
		log.Fatal("no address to listen on for updates")
	}
	if config.Token == "" {
		log.Fatal("no authentication token")
	}
	baseURL, err = url.Parse(fmt.Sprintf("https://api.telegram.org/bot%s/", url.PathEscape(config.Token)))
	if err != nil {
		panic(err)
	}
	inlineQueryCacheTimeSeconds = int(config.InlineQueryCacheTime.Duration.Seconds())

	googleShortener = monstrator.NewGoogleShortener(config.Shorteners.Google.APIKey,
		&http.Client{Timeout: config.Shorteners.Google.Timeout.Duration})
	isgdShortener = monstrator.NewIsgdShortener(&http.Client{Timeout: config.Shorteners.Isgd.Timeout.Duration})
	tinyURLShortener = monstrator.NewTinyURLShortener(&http.Client{Timeout: config.Shorteners.TinyURL.Timeout.Duration})

	server := &http.Server{ReadTimeout: config.ReadTimeout.Duration, WriteTimeout: config.WriteTimeout.Duration,
		Handler: http.HandlerFunc(handleUpdate), Addr: config.Address}
	log.Printf("about to listen for updates on %s", config.Address)
	if config.TLS == nil || config.TLS.Certificate == "" || config.TLS.Key == "" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(config.TLS.Certificate, config.TLS.Key))
	}
}

var shortenerNames = map[monstrator.Shortener]string{
	googleShortener:  "Google",
	isgdShortener:    "isgd",
	tinyURLShortener: "TinyURL"}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	// Ensure update comes from Telegram.
	if r.URL.Path != config.Token {
		w.WriteHeader(http.StatusForbidden)
		log.Printf("unauthorized update from %s", r.RemoteAddr)
		return
	}

	var upd update
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&upd)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if upd.InlineQuery != nil {
		handleInlineQuery(w, upd.InlineQuery)
	} else {
		w.WriteHeader(http.StatusNotFound)
		log.Print("webhook configured to receive unnecessary updates")
	}
}

func isShortURL(u *url.URL) (bool, monstrator.Shortener) {
	switch {
	case googleShortener.IsShortURL(u):
		return true, googleShortener
	case isgdShortener.IsShortURL(u):
		return true, isgdShortener
	case tinyURLShortener.IsShortURL(u):
		return true, tinyURLShortener
	}
	return false, nil
}

func handleInlineQuery(w http.ResponseWriter, q *inlineQuery) {
	if q.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		log.Print("empty inline query ID")
		return
	}
	if q.Text == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	u, err := url.Parse(q.Text)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if ok, shortener := isShortURL(u); ok {
		longURL, err := shortener.Expand(u)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			logError := func() {
				log.Printf("failed to expand %s with %s: %s", u, shortenerNames[shortener], err)
			}
			switch err := err.(type) {
			case *monstrator.GoogleShortenerError:
				if err.Code != 404 {
					logError()
				}
			case *monstrator.IsgdShortenerError:
				if err.Code != 400 {
					logError()
				}
			case monstrator.TinyURLShortenerError:
				if err != 404 {
					logError()
				}
			}
			return
		}

		shortenerName := shortenerNames[shortener]
		encodedURL := longURL.String()
		results := []interface{}{
			&inlineQueryResultArticle{ID: shortenerName, Title: shortenerName, URL: encodedURL,
				InputMessageContent: &inputTextMessageContent{Text: encodedURL}}}
		answerInlineQuery(w, results)
		return
	}

	shortURLs := make(map[monstrator.Shortener]*url.URL)
	wg := sync.WaitGroup{}
	wg.Add(3)
	var shorten = func(shortener monstrator.Shortener) {
		defer wg.Done()
		shortURL, err := shortener.Shorten(u)
		if err != nil {
			log.Printf("failed to shorten %s with the %s shortener: %s", u, shortenerNames[shortener], err)
		} else {
			shortURLs[shortener] = shortURL
		}
	}
	go shorten(googleShortener)
	go shorten(isgdShortener)
	go shorten(tinyURLShortener)
	wg.Wait()

	if len(shortURLs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	results := make([]interface{}, len(shortURLs))
	i := 0
	for shortener, shortURL := range shortURLs {
		encodedURL := shortURL.String()
		shortenerName := shortenerNames[shortener]
		results[i] = &inlineQueryResultArticle{ID: shortenerName, Title: shortenerName, URL: encodedURL,
			InputMessageContent: &inputTextMessageContent{Text: encodedURL}}
		i++
	}
	answerInlineQuery(w, results)
}

var inlineQueryCacheTimeSeconds int

func answerInlineQuery(w http.ResponseWriter, results []interface{}) {
	if len(results) == 0 {
		panic("attempting to answer inline query without results")
	}
	enc := json.NewEncoder(w)
	err := enc.Encode(map[string]interface{}{
		"method":     answerInlineQueryMethod,
		"results":    results,
		"cache_time": inlineQueryCacheTimeSeconds})
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		switch err := err.(type) {
		case *json.MarshalerError:
			panic(err)
		case *json.UnsupportedTypeError:
			panic(err)
		case *json.UnsupportedValueError:
			panic(err)
		default:
			log.Printf("failed to answer inline query: %s", err)
		}
	}
}
