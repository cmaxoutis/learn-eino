# learn-eino

Learn [Eino](https://github.com/cloudwego/eino) by examples. Each example is a standalone program — just run it with `go run`.

## Examples

| # | Example | Description |
|---|---------|-------------|
| 01 | [Ollama inference](examples/01_ollama_inference/) | Single-shot and streaming inference against a local Ollama server. |
| 02 | [Sub-agent](examples/02_sub_agent/) | ADK sub-agent via the `AgentAsTool` pattern: wrap a sub-agent as a tool the parent agent can delegate to. |

## Prerequisites

- Go
- A local [Ollama](https://ollama.com) server with a pulled model.

## Usage

```bash
# Optional configuration (defaults shown)
export OLLAMA_BASE_URL=http://localhost:11434
export OLLAMA_MODEL=llama3.1

go run examples/01_ollama_inference/main.go
go run examples/02_sub_agent/main.go
```
