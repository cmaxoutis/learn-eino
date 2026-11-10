// Package agentutil provides shared helpers for the examples' ADK event loops.
package agentutil

import (
	"log"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// PrintMessageOutput prints a message output, consuming the stream when the
// runner is in streaming mode. The agent name and role make it clear which
// agent produced the text and whether it is a model reply or a tool result.
func PrintMessageOutput(agentName string, mv *adk.TypedMessageVariant[*schema.Message]) {
	if !mv.IsStreaming {
		log.Printf("[%s/%s] %s", agentName, mv.Role, mv.Message.Content)
		return
	}
	var sb strings.Builder
	for {
		msg, recvErr := mv.MessageStream.Recv()
		if recvErr != nil {
			break // io.EOF means the stream finished.
		}
		sb.WriteString(msg.Content)
	}
	if sb.Len() > 0 {
		log.Printf("[%s/%s] %s", agentName, mv.Role, sb.String())
	}
}
