// Command mcp_servers shows how to connect to an MCP server, pull its tools,
// and bind them to an Eino ADK agent. It also starts a tiny in-process MCP
// server so the example is self-contained.
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cmaxoutis/learn-eino/examples/internal/agentutil"
	"github.com/cmaxoutis/learn-eino/examples/internal/ollamahelper"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Start a tiny in-process MCP server first so the example is self-contained.
	startMCPServer()
	time.Sleep(time.Second)
	ctx := context.Background()

	// 1. Connect to the MCP server and fetch its tools.
	mcpTools, err := getMCPTool(ctx)
	if err != nil {
		log.Fatalf("getMCPTool failed, err=%v", err)
	}

	chatModel, err := ollamahelper.NewChatModel(ctx)
	if err != nil {
		log.Fatalf("newChatModel failed, err=%v", err)
	}

	// 2. Bind the MCP tools to the agent like any other tool.
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "mcp-assistant",
		Description: "An assistant that can use MCP-provided tools.",
		Instruction: "You are a helpful assistant. Use the get_time and get_weather tools when needed.",
		Model:       chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: mcpTools,
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent failed, err=%v", err)
	}

	// 3. Run the agent; it can call the MCP "get_time" and "get_weather" tools.
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// The clock and the live weather are things the model cannot know without
	// the MCP tools, which makes this a real end-to-end MCP test.
	iter := runner.Query(ctx, "What's the current time, and what's the weather in London?")
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

// getMCPTool connects to the SSE MCP server and returns its tools.
func getMCPTool(ctx context.Context) ([]tool.BaseTool, error) {
	cli, err := client.NewSSEMCPClient("http://localhost:12345/sse")
	if err != nil {
		return nil, err
	}
	if startErr := cli.Start(ctx); startErr != nil {
		return nil, startErr
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}
	if _, initErr := cli.Initialize(ctx, initRequest); initErr != nil {
		return nil, initErr
	}

	return mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
}

// startMCPServer runs a minimal in-process MCP server exposing a "get_time"
// and a "get_weather" tool over SSE on localhost:12345. Time is unknown to the
// model, and get_weather hits a live external API (wttr.in, no key needed) —
// neither can be faked from training data.
func startMCPServer() {
	svr := server.NewMCPServer("demo", mcp.LATEST_PROTOCOL_VERSION)

	svr.AddTool(mcp.NewTool(
		"get_time",
		mcp.WithDescription("Get the current local time in RFC 3339 format"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(time.Now().Format(time.RFC3339)), nil
	})

	svr.AddTool(mcp.NewTool(
		"get_weather",
		mcp.WithDescription("Get the current weather for a city"),
		mcp.WithString("city", mcp.Required(), mcp.Description("City name")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultText("invalid arguments"), nil
		}
		city, _ := args["city"].(string)
		// wttr.in returns a one-line report for a plain HTTP GET; an allowlisted
		// URL keeps the tool safe (the model can only ask about weather).
		url := "https://wttr.in/" + url.PathEscape(city) + "?format=3"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return mcp.NewToolResultText("error: " + err.Error()), nil
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return mcp.NewToolResultText("error: " + err.Error()), nil
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
		if err != nil {
			return mcp.NewToolResultText("error: " + err.Error()), nil
		}
		return mcp.NewToolResultText(strings.TrimSpace(string(body))), nil
	})

	go func() {
		if err := server.NewSSEServer(svr, server.WithBaseURL("http://localhost:12345")).Start("localhost:12345"); err != nil {
			log.Printf("MCP server stopped, err=%v", err)
		}
	}()
}
