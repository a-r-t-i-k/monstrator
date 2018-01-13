package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/r3turnz/monstrator"
)

var shorteners []monstrator.Shortener

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
			log.Fatalf("failed to load configuration from file: %v", err)
		}
	}
	err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to load configuration from environment variables: %v", err)
	}

	if config.Address == "" {
		if config.Port != "" {
			config.Address = ":" + config.Port
		} else {
			log.Fatal("no address to listen on for updates")
		}
	}
	if config.Token == "" {
		log.Fatal("no authentication token")
	}
	inlineQueryCacheTimeSeconds = int(config.InlineQueryCacheTime.Duration.Seconds())

	shorteners = make([]monstrator.Shortener, 2)
	shorteners[0] = monstrator.NewGoogleShortener(config.Shorteners.Google.APIKey,
		&http.Client{Timeout: config.Shorteners.Google.Timeout.Duration})
	shorteners[1] = monstrator.NewIsgdShortener(&http.Client{Timeout: config.Shorteners.Isgd.Timeout.Duration})

	server := &http.Server{ReadTimeout: config.ReadTimeout.Duration, WriteTimeout: config.WriteTimeout.Duration,
		Handler: http.HandlerFunc(handleUpdate), Addr: config.Address}
	log.Printf("about to listen for updates on %v", config.Address)
	if config.TLS.Certificate == "" || config.TLS.Key == "" {
		log.Fatal(server.ListenAndServe())
	} else {
		log.Fatal(server.ListenAndServeTLS(config.TLS.Certificate, config.TLS.Key))
	}
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	// Ensure update comes from Telegram.
	if strings.TrimPrefix(r.URL.Path, "/") != config.Token {
		w.WriteHeader(http.StatusForbidden)
		log.Printf("unauthorized update from %v", r.RemoteAddr)
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

func isShortenedURL(u *url.URL) (bool, monstrator.Shortener) {
	for _, shortener := range shorteners {
		if shortener.IsShortenedURL(u) {
			return true, shortener
		}
	}
	return false, nil
}

func handleInlineQuery(w http.ResponseWriter, q *inlineQuery) {
	q.ID = strings.TrimSpace(q.ID)
	if q.ID == "" {
		w.WriteHeader(http.StatusBadRequest)
		log.Print("empty inline query ID")
		return
	}
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	u, err := url.Parse(q.Text)
	if err != nil || !u.IsAbs() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if ok, shortener := isShortenedURL(u); ok {
		longURL, err := shortener.Expand(u)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("failed to expand %v with %s: %v", u, shortener.Name(), err)
			return
		}

		encodedURL := longURL.String()
		results := []interface{}{
			&inlineQueryResultArticle{ID: shortener.Name(), Title: shortener.Name(), URL: encodedURL,
				InputMessageContent: &inputTextMessageContent{Text: encodedURL}}}
		answerInlineQuery(w, q.ID, results)
		return
	}

	results := make([]interface{}, 0, len(shorteners))
	m := sync.Mutex{}
	wg := sync.WaitGroup{}
	var shorten = func(shortener monstrator.Shortener) {
		defer wg.Done()
		shortenedURL, err := shortener.Shorten(u)
		if err != nil {
			log.Printf("failed to shorten %v with the %s shortener: %v", u, shortener.Name(), err)
		} else {
			encodedURL := shortenedURL.String()
			result := &inlineQueryResultArticle{ID: shortener.Name(), Title: shortener.Name(), URL: encodedURL,
				InputMessageContent: &inputTextMessageContent{Text: encodedURL}}
			m.Lock()
			results = append(results, result)
			m.Unlock()
		}
	}
	wg.Add(len(shorteners))
	for _, shortener := range shorteners {
		go shorten(shortener)
	}
	wg.Wait()

	if len(results) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	answerInlineQuery(w, q.ID, results)
}

var inlineQueryCacheTimeSeconds int

func answerInlineQuery(w http.ResponseWriter, ID string, results []interface{}) {
	if len(results) == 0 {
		panic("attempting to answer inline query without results")
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(map[string]interface{}{
		"method":          answerInlineQueryMethod,
		"inline_query_id": ID,
		"results":         results,
		"cache_time":      inlineQueryCacheTimeSeconds})
	if err != nil {
		panic(err)
	}
}
