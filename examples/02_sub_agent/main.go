// Command sub_agent shows how to run a sub-agent from a parent agent using the
// AgentAsTool pattern: the sub-agent is wrapped as a tool the parent can call.
package main

import (
	"context"
	"log"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cmaxoutis/learn-eino/examples/internal/ollamahelper"
)

func main() {
	ctx := context.Background()

	chatModel, err := ollamahelper.NewChatModel(ctx)
	if err != nil {
		log.Fatalf("newChatModel failed, err=%v", err)
	}

	// 1. Build the sub-agent. It is a normal ChatModelAgent with its own
	// instruction and (optionally) its own tools.
	subAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_expert",
		Description: "Solves arithmetic problems step by step.",
		Instruction: "You are a math expert. Solve the problem and return only the final answer.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent(sub) failed, err=%v", err)
	}

	// 2. Wrap the sub-agent as a tool so the parent can call it.
	subAgentTool := adk.NewAgentTool(ctx, subAgent)

	// 3. Build the parent agent that owns the sub-agent tool.
	parentAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "coordinator",
		Description: "Coordinates specialist sub-agents.",
		Instruction: "You are a coordinator. Delegate math questions to the math_expert tool.",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{subAgentTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent(parent) failed, err=%v", err)
	}

	// 4. Run the parent; it decides when to delegate to the sub-agent.
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           parentAgent,
		EnableStreaming: true,
	})

	iter := runner.Query(ctx, "What is 17 * 23?")
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatalf("event error, err=%v", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			printMessageOutput(event.AgentName, event.Output.MessageOutput)
		}
	}
}

// printMessageOutput prints a message output, consuming the stream when the
// runner is in streaming mode. The agent name and role make it clear which
// agent produced the text and whether it is a model reply or a tool result.
func printMessageOutput(agentName string, mv *adk.TypedMessageVariant[*schema.Message]) {
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
