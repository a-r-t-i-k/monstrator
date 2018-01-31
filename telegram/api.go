package main

import (
	"encoding/json"
	"net/http"
)

const (
	sendMessageMethod       string = "sendMessage"
	answerInlineQueryMethod string = "answerInlineQuery"

	defaultParseMode  string = ""
	markdownParseMode string = "Markdown"
)

type update struct {
	InlineQuery *inlineQuery `json:"inline_query"`
	Message     *message     `json:"message"`
}

type inlineQuery struct {
	ID   string `json:"id"`
	Text string `json:"query"`
}

type inputTextMessageContent struct {
	Text string `json:"message_text"`
}

type message struct {
	Sender *user  `json:"from"`
	Chat   *chat  `json:"chat"`
	Text   string `json:"text"`
}

type user struct {
	FirstName string `json:"first_name"`
}

type chat struct {
	ID int64 `json:"id"`
}

type inlineQueryResultArticle struct {
	ID                  string      `json:"id"`
	Title               string      `json:"title"`
	InputMessageContent interface{} `json:"input_message_content"`
	URL                 string      `json:"url,ommitempty"`
}

func (article *inlineQueryResultArticle) MarshalJSON() ([]byte, error) {
	type alias inlineQueryResultArticle
	return json.Marshal(&struct {
		Type string `json:"type"`
		*alias
	}{"article", (*alias)(article)})
}

func answerInlineQuery(w http.ResponseWriter, ID string, results []interface{}) {
	if len(results) == 0 {
		panic("attempting to answer inline query without results")
	}
	answerUpdate(w, answerInlineQueryMethod, map[string]interface{}{
		"inline_query_id": ID,
		"results":         results,
		"cache_time":      inlineQueryCacheTimeSeconds})
}

func sendMessage(w http.ResponseWriter, chatID string, text string, parseMode string) {
	if len(text) == 0 {
		panic("attempting to send empty message")
	}
	params := map[string]interface{}{
		"chat_id": chatID,
		"text":    text}
	if parseMode != "" {
		params["parse_mode"] = parseMode
	}
	answerUpdate(w, sendMessageMethod, params)
}

func answerUpdate(w http.ResponseWriter, method string, params map[string]interface{}) {
	if method == "" {
		panic("attempting to answer update without method")
	}
	params["method"] = method
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(params)
	if err != nil {
		panic(err)
	}
}
