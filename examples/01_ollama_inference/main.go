// Command ollama_inference shows how to run inference against a local Ollama
// server with a specified model, both directly and through an Eino ADK agent.
package main

import (
	"context"
	"log"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/cmaxoutis/learn-eino/examples/internal/ollamahelper"
)

func main() {
	ctx := context.Background()

	chatModel, err := ollamahelper.NewChatModel(ctx)
	if err != nil {
		log.Fatalf("NewChatModel failed, err=%v", err)
	}

	// 1. Direct single-shot inference.
	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{Role: schema.System, Content: "You are a terse assistant."},
		{Role: schema.User, Content: "What is the capital of France?"},
	})
	if err != nil {
		log.Fatalf("Generate failed, err=%v", err)
	}
	log.Printf("direct: %s", resp.Content)

	// 2. Streaming inference. Chunks arrive token-by-token, so accumulate them
	// and print the full reply once instead of one line per token.
	stream, err := chatModel.Stream(ctx, []*schema.Message{
		{Role: schema.User, Content: "Count from 1 to 3."},
	})
	if err != nil {
		log.Fatalf("Stream failed, err=%v", err)
	}
	defer stream.Close()

	var sb strings.Builder
	for {
		chunk, recvErr := stream.Recv()
		if recvErr != nil {
			break // io.EOF means the stream finished.
		}
		sb.WriteString(chunk.Content)
	}
	log.Printf("streamed: %s", sb.String())
}
