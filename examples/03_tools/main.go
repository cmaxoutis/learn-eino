// Command tools shows how to define custom tools and bind them to an Eino ADK
// agent so the model can call them during a ReAct loop.
package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cmaxoutis/learn-eino/examples/internal/agentutil"
	"github.com/cmaxoutis/learn-eino/examples/internal/ollamahelper"
)

// weatherInput is the schema the model sees for the tool; utils.InferTool
// derives the parameter JSON-schema from this struct.
type weatherInput struct {
	City string `json:"city" jsonschema:"description=City name"`
}

// getWeather is a plain Go function wired as a tool by InferTool.
func getWeather(_ context.Context, in *weatherInput) (string, error) {
	return "4C and cloudy", nil
}

func main() {
	ctx := context.Background()

	chatModel, err := ollamahelper.NewChatModel(ctx)
	if err != nil {
		log.Fatalf("newChatModel failed, err=%v", err)
	}

	// 1. Create the tool from a plain function.
	weatherTool, err := utils.InferTool("get_weather", "Get the weather for a city", getWeather)
	if err != nil {
		log.Fatalf("InferTool failed, err=%v", err)
	}

	// 2. Bind the tool to the agent.
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "assistant",
		Description: "A helpful assistant with a weather tool.",
		Instruction: "You are a helpful assistant. Use the get_weather tool when asked about weather.",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{weatherTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent failed, err=%v", err)
	}

	// 3. Run the agent; the model decides when to call the tool.
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	iter := runner.Query(ctx, "What's the weather in Tokyo?")
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatalf("event error, err=%v", event.Err)
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			agentutil.PrintMessageOutput(event.AgentName, event.Output.MessageOutput)
		}
	}
}
