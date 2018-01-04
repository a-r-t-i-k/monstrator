package main

import (
	"encoding/json"
	"net/url"
)

var baseURL *url.URL

const (
	answerInlineQueryMethod string = "answerInlineQuery"
	sendMessageMethod       string = "sendMessage"
)

type update struct {
	InlineQuery *inlineQuery `json:"inline_query"`
}

type inlineQuery struct {
	ID   string `json:"id"`
	Text string `json:"query"`
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

type inputTextMessageContent struct {
	Text string `json:"message_text"`
}
