// Package ollamahelper provides a shared Ollama chat model constructor used by
// the examples. It reads the standard $OLLAMA_HOST (e.g. "127.0.0.1:11434" or
// a full "http(s)://host:port" URL) and $OLLAMA_MODEL variables.
package ollamahelper

import (
	"context"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ollama"
)

// NewChatModel builds an Ollama chat model from $OLLAMA_HOST and $OLLAMA_MODEL,
// defaulting to http://localhost:11434 and "qwen3.8:latest".
func NewChatModel(ctx context.Context) (*ollama.ChatModel, error) {
	baseURL := os.Getenv("OLLAMA_HOST")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen3.8:latest"
	}

	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   model,
	})
}
