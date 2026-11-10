// Command sub_agent shows how a parent agent delegates to specialist
// sub-agents using the AgentAsTool pattern: each sub-agent is wrapped as a
// tool the parent can call by name.
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

	// 1. Build two specialist sub-agents. Each is a normal ChatModelAgent with
	// its own name, description (used by the parent to decide whom to call) and
	// instruction. The name and description are what make delegation work: the
	// parent agent sees them as tool metadata.
	chef, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "chef",
		Description: "Suggests recipes you can cook from a list of ingredients.",
		Instruction: "You are a chef. Given available ingredients, suggest one simple dish and how to cook it in 3 short steps.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent(chef) failed, err=%v", err)
	}

	guide, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "local_guide",
		Description: "Plans day trips and itineraries for visiting a city.",
		Instruction: "You are a local tour guide. Plan a compact one-day itinerary with 3 highlights.",
		Model:       chatModel,
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent(guide) failed, err=%v", err)
	}

	// 2. Wrap each sub-agent as a tool so the parent can call it.
	// 3. Build the parent agent that owns the sub-agent tools. It decides which
	// specialist to delegate to based on their name/description.
	parentAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "concierge",
		Description: "Helps with travel and food questions by delegating to specialists.",
		Instruction: "You are a hotel concierge. Delegate food questions to the chef tool and sightseeing questions to the local_guide tool. Summarize each specialist's answer in one line, prefixed with their name.",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					adk.NewAgentTool(ctx, chef),
					adk.NewAgentTool(ctx, guide),
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent(parent) failed, err=%v", err)
	}

	// 4. Run the parent; it decides when to delegate to each sub-agent.
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           parentAgent,
		EnableStreaming: true,
	})

	// The question spans both specialties, so the parent should call both tools
	// and merge their answers.
	iter := runner.Query(ctx, "I'm visiting Lisbon for one day and I have eggs, potatoes and chorizo. What should I see, and what can I cook for dinner?")
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
