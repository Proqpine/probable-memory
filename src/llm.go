package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sanity-io/litter"
	"github.com/subosito/gotenv"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type AIResponse struct {
	ID                string    `json:"id"`
	Object            string    `json:"object"`
	Created           int       `json:"created"`
	Model             string    `json:"model"`
	Choices           []Choices `json:"choices"`
	Usage             Usage     `json:"usage"`
	SystemFingerprint string    `json:"system_fingerprint"`
}
type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Refusal any    `json:"refusal"`
}
type Choices struct {
	Index           int             `json:"index"`
	ResponseMessage ResponseMessage `json:"message"`
	Logprobs        any             `json:"logprobs"`
	FinishReason    string          `json:"finish_reason"`
}
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func SummariseActivities() {
	err := gotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env: %s", err)
	}

	OPENAI_API_KEY := os.Getenv("OPENAI_API_KEY")

	request := AIRequest{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Write a haiku that explains dependency injection.",
			},
		},
	}

	requestByte, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}

	req, err := http.NewRequest(
		"POST",
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(requestByte),
	)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(
		"Authorization",
		fmt.Sprintf("Bearer %s", OPENAI_API_KEY))

	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	var response AIResponse

	body, err := io.ReadAll(resp.Body)
	fmt.Println(string(body))
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: %s", resp.Status)
	}
	data := NewWebHookData(
		response.Choices[0].ResponseMessage.Content,
		"Haiku Naiku",
		"https://gravatar.com/avatar/344ff2b0f7ecff02ad9050696059866c?s=400&d=robohash&r=x",
	)
	ExecuteWebHook(data)
	litter.Dump(response.Choices[0].ResponseMessage)
}
