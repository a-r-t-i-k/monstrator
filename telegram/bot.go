package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/r3turnz/monstrator"
)

var shorteners []monstrator.Shortener
var shortenerNames = make(map[monstrator.Shortener]string)
var thumbnailURLs = make(map[monstrator.Shortener]*url.URL)
var inlineQueryCacheTimeSeconds int

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
	googleShortener := monstrator.NewGoogleShortener(config.Shorteners.Google.APIKey,
		&http.Client{Timeout: config.Shorteners.Google.Timeout.Duration})
	shortenerNames[googleShortener] = "Google"
	thumbnailURLs[googleShortener], err = url.Parse("/thumbnails/google.png")
	if err != nil {
		panic(err)
	}
	shorteners[0] = googleShortener
	isgdShortener := monstrator.NewIsgdShortener(&http.Client{Timeout: config.Shorteners.Isgd.Timeout.Duration})
	shortenerNames[isgdShortener] = "is.gd"
	thumbnailURLs[isgdShortener], err = url.Parse("/thumbnails/is.gd.jpg")
	if err != nil {
		panic(err)
	}
	shorteners[1] = isgdShortener

	http.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir("thumbnails"))))
	http.HandleFunc("/", handleUpdate)

	server := &http.Server{ReadTimeout: config.ReadTimeout.Duration, WriteTimeout: config.WriteTimeout.Duration, Addr: config.Address}
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
		return
	}

	var upd update
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&upd)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("invalid update: %v", err)
		return
	}

	switch {
	case upd.InlineQuery != nil:
		handleInlineQuery(w, r, upd.InlineQuery)
	case upd.Message != nil:
		handleMessage(w, upd.Message)
	default:
		w.WriteHeader(http.StatusNoContent)
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

func buildInlineQueryResultArticle(shortener monstrator.Shortener, u *url.URL, r *http.Request) *inlineQueryResultArticle {
	encodedLongURL := u.String()
	article := &inlineQueryResultArticle{ID: shortenerNames[shortener], Title: shortenerNames[shortener], URL: encodedLongURL,
		InputMessageContent: &inputTextMessageContent{Text: encodedLongURL}}
	thumbnailURL := assembleAbsoluteURL(thumbnailURLs[shortener], r)
	if thumbnailURL != nil {
		article.Thumbnail = thumbnailURL.String()
	}
	return article
}

func handleInlineQuery(w http.ResponseWriter, r *http.Request, q *inlineQuery) {
	if q.Text == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	u, err := url.Parse(q.Text)
	if err != nil || !u.IsAbs() || !isDomainName(u.Hostname()) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var results []interface{}
	if ok, shortener := isShortenedURL(u); ok {
		longURL, err := shortener.Expand(u)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("failed to expand %v with %s: %v", u, shortenerNames[shortener], err)
			return
		}
		results = []interface{}{buildInlineQueryResultArticle(shortener, longURL, r)}
	} else {
		wg := sync.WaitGroup{}
		c := make(chan interface{}, len(shorteners))
		shorten := func(shortener monstrator.Shortener) {
			defer wg.Done()
			shortenedURL, err := shortener.Shorten(u)
			if err != nil {
				log.Printf("failed to shorten %v with the %s shortener: %v", u, shortenerNames[shortener], err)
			} else {
				c <- buildInlineQueryResultArticle(shortener, shortenedURL, r)
			}
		}
		wg.Add(len(shorteners))
		for _, shortener := range shorteners {
			go shorten(shortener)
		}
		wg.Wait()
		close(c)

		results = make([]interface{}, 0, len(c))
		for result := range c {
			results = append(results, result)
		}
		if len(results) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	answerInlineQuery(w, q.ID, results)
}

func handleMessage(w http.ResponseWriter, m *message) {
	if m.Chat == nil {
		log.Print("message from no chat")
		w.WriteHeader(http.StatusBadRequest)
	}

	var text string
	parseMode := defaultParseMode
	disableWebPagePreview := false
	switch {
	case m.Text == "":
		text = "Sorry, I only understand text messages."
	case !strings.HasPrefix(m.Text, "/"):
		text = "Sorry, I only can interact with commands."
	default:
		command := strings.TrimPrefix(strings.Split(m.Text, " ")[0], "/")
		switch command {
		case "start":
			if m.Sender == nil || m.Sender.FirstName == "" {
				text = "Hello!\nI can shorten and expand URLs with [inline queries](https://core.telegram.org/bots/inline)."
			} else {
				text = fmt.Sprintf("Hello %s!\nI can shorten and expand URLs with [inline queries](https://core.telegram.org/bots/inline).", m.Sender.FirstName)
			}
			parseMode = markdownParseMode
			disableWebPagePreview = true
		default:
			text = "Sorry, I don't recognize that command."
		}
	}
	sendMessage(w, strconv.FormatInt(m.Chat.ID, 10), text, parseMode, disableWebPagePreview)
}
