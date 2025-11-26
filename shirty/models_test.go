package shirty

import (
	"fmt"
	"os"
	"testing"

	"github.com/sandialabs/bibcheck/openai"
)

func TestShirtyChatGptOss120b(t *testing.T) {

	apiKey, ok := os.LookupEnv("SHIRTY_API_KEY")
	if !ok {
		t.Skip("SHIRTY_API_KEY not provided")
	}

	client := openai.NewClient(apiKey, openai.WithBaseUrl("https://shirty.sandia.gov/api/v1"))

	req := &openai.ChatRequest{
		Model: "openai/gpt-oss-120b",
		Messages: []openai.Message{
			{
				Role:    openai.RoleUser,
				Content: "What are you?",
			},
		},
	}

	resp, err := client.Chat(req)
	if err != nil {
		t.Fatalf("openai client error: %v", err)
	}

	if len(resp.Choices) < 1 {
		t.Fatalf("expected at least one response choice")
	}
	choice := resp.Choices[0]

	if choice.Message.Role != openai.RoleAssistant {
		t.Fatalf("expected assistant role in response")
	}

	fmt.Println(choice.Message.Content)
}
