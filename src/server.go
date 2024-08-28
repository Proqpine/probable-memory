// https://discord.com/api/webhooks/1277902137254350920/MI3_f0XSDWL7pE_J4tHtcWU_QI7lK22aw5QlwjSXX_KqoM8iPFLO0npb05oPizarEuOA

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type EmbedType int

const (
	Rich EmbedType = iota
	Image
	Video
	Gifv
	Article
	Link
)

func (e EmbedType) String() string {
	return [...]string{"rich", "image", "video", "gifv", "article", "link"}[e]
}

type Author struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	IconUrl string `json:"icon_url"`
}

type Embed struct {
	Title       string    `json:"title"`
	Type        EmbedType `json:"type"`
	Description string    `json:"description"`
	Url         string    `json:"url"`
	Timestamp   time.Time `json:"timestamp"`
	Color       int       `json:"color"`
	Author      Author    `json:"author"`
}

type WebhookData struct {
	Content   string  `json:"content"`
	Username  string  `json:"username"`
	AvatarUrl string  `json:"avatar_url"`
	Embeds    []Embed `json:"embeds"`
}

type EmbedOption func(*Embed)
type WebhookOption func(*WebhookData)

func NewWebHookData(content, username, avatarUrl string, options ...WebhookOption) WebhookData {
	hookData := WebhookData{
		Content:   content,
		Username:  username,
		AvatarUrl: avatarUrl,
	}
	for _, option := range options {
		option(&hookData)
	}
	return hookData
}

func WithEmbed(embedOptions ...EmbedOption) WebhookOption {
	return func(wd *WebhookData) {
		embed := Embed{}
		for _, option := range embedOptions {
			option(&embed)
		}
		wd.Embeds = append(wd.Embeds, embed)
	}
}

func WithTitle(title string) EmbedOption {
	return func(e *Embed) {
		e.Title = title
	}
}

func WithDescription(description string) EmbedOption {
	return func(e *Embed) {
		e.Description = description
	}
}

func WithUrl(url string) EmbedOption {
	return func(e *Embed) {
		e.Url = url
	}
}

func WithTimestamp(timestamp time.Time) EmbedOption {
	return func(e *Embed) {
		e.Timestamp = timestamp
	}
}

func WithColor(color int) EmbedOption {
	return func(e *Embed) {
		e.Color = color
	}
}

func WithAuthor(name, url, iconUrl string) EmbedOption {
	return func(e *Embed) {
		e.Author = Author{
			Name:    name,
			Url:     url,
			IconUrl: iconUrl,
		}
	}
}

func WithType(embedType EmbedType) EmbedOption {
	return func(e *Embed) {
		e.Type = embedType
	}
}

func ExecuteWebHook(data WebhookData) {
	webHook := data
	marshalledJson, err := json.Marshal(webHook)
	if err != nil {
		log.Fatalf("Impossible to marshall teacher: %s", err)
	}

	postUrl := "https://discord.com/api/webhooks/1277902137254350920/MI3_f0XSDWL7pE_J4tHtcWU_QI7lK22aw5QlwjSXX_KqoM8iPFLO0npb05oPizarEuOA"
	resp, err := http.Post(postUrl, "application/json", bytes.NewReader(marshalledJson))
	if err != nil {
		log.Fatalf("Error with webhook: %s", err)
	}
	defer resp.Body.Close()
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Impossible to read all body of response: %s", err)
	}
	log.Printf("Response: %s, %v", string(resBody), resp.StatusCode)
}

func main() {
	data := NewWebHookData(
		"Captain hook says hi",
		"Hook and Eye",
		"https://example.com/go-icon.png",
	)
	ExecuteWebHook(data)
}
