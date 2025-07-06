# mcp-agents-go

A Go library for building intelligent agents that can interact with Multiple Model Context Protocol (MCP) servers using Large Language Models (LLMs).

## Overview

mcp-agents-go provides a framework for creating AI agents that can:
- Connect to multiple MCP servers to access various tools and resources
- Use different LLM providers (currently supports Azure OpenAI)
- Execute tool calls based on natural language prompts
- Manage multiple agents with different capabilities and tool access

This project uses:
- [langchain-go](https://github.com/tmc/langchaingo) for LLM integration
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) for MCP client and server functionality

## Features

- **Multi-provider LLM support**: Currently supports Azure OpenAI with extensible architecture
- **Flexible MCP server connections**: Supports both stdio and SSE transport types
- **Agent-based architecture**: Create multiple agents with different tool access permissions
- **Configuration-driven setup**: YAML-based configuration for easy deployment
- **Tool access control**: Fine-grained control over which tools each agent can use
- **Streaming responses**: Real-time response streaming with `GenerateContentAsStreaming`
- **Enhanced conversation flow**: Support for complex multi-turn conversations with tool interactions

## Installation

```bash
go get github.com/carlossantin/mcp-agents-go
```

## Quick Start

### 1. Configuration File

Create a `config.yaml` file with your providers, servers, and agents:

```yaml
providers:
  - name: my-azure-provider
    type: AZURE
    token: <YOUR_AZURE_TOKEN>
    baseUrl: <YOUR_AZURE_BASE_URL>
    model: gpt-4o-mini
    version: 2025-01-01-preview

servers:
  - name: my-mcp-server
    type: sse
    url: http://localhost:8080/mcp/events
    # For stdio servers:
    # type: stdio
    # command: 
    #   - /path/to/your/mcp-server

agents:
  - name: my-agent
    servers:
      - name: my-mcp-server
        allowed_tools:
          - tool001
          - tool002
    provider: my-azure-provider
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/carlossantin/mcp-agents-go/config"
    "github.com/tmc/langchaingo/llms"
)

func main() {
    ctx := context.Background()
    
    // Setup from configuration file
    err := config.SetupFromFile(ctx, "config.yaml")
    if err != nil {
        panic(err)
    }
    
    // Get an agent and generate content
    agent, ok := config.SysConfig.Agents["my-agent"]
    if !ok {
        panic("Agent not found")
    }
    
    // Create message content
    msgs := []llms.MessageContent{
        {Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "What tools are available?"}}},
    }
    
    response, _ := agent.GenerateContent(ctx, msgs, false)
    fmt.Println(response)
}
```

### 2.1. Streaming Usage

For real-time streaming responses:

```go
package main

import (
    "context"
    "fmt"
    "github.com/carlossantin/mcp-agents-go/config"
    "github.com/tmc/langchaingo/llms"
)

func main() {
    ctx := context.Background()
    
    // Setup from configuration file
    err := config.SetupFromFile(ctx, "config.yaml")
    if err != nil {
        panic(err)
    }
    
    // Get an agent
    agent, ok := config.SysConfig.Agents["my-agent"]
    if !ok {
        panic("Agent not found")
    }
    
    // Create message content
    msgs := []llms.MessageContent{
        {Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "Give me the current dollar to real exchange rate in BRL."}}},
    }
    
    // Stream responses
    var textResp <-chan string
    var msgsResp <-chan llms.MessageContent
    textResp, msgsResp = agent.GenerateContentAsStreaming(ctx, msgs, true)

    // Process both channels concurrently
    go func() {
        for resp := range msgsResp {
            msgs = append(msgs, resp)
        }
    }()

    for resp := range textResp {
        fmt.Print(resp)
    }
}
```

### 3. Programmatic Setup

Instead of using a configuration file, you can set up the system programmatically:

```go
package main

import (
    "context"
    "github.com/carlossantin/mcp-agents-go/config"
    "github.com/carlossantin/mcp-agents-go/agent"
    "github.com/tmc/langchaingo/llms"
)

func main() {
    ctx := context.Background()
    
    providers := []config.LLMProvider{
        {
            Name:    "my-provider",
            Type:    "AZURE",
            Token:   "your-token",
            BaseURL: "your-base-url",
            Model:   "gpt-4o-mini",
            Version: "2025-01-01-preview",
        },
    }
    
    servers := []config.MCPServer{
        {
            Name: "my-server",
            Type: "sse",
            URL:  "http://localhost:8080/mcp/events",
        },
    }
    
    agents := []config.MCPAgent{
        {
            Name: "my-agent",
            MCPAgentServers: []agent.MCPAgentServer{
                {
                    Name:         "my-server",
                    AllowedTools: []string{"tool1", "tool2"},
                },
            },
            Provider: "my-provider",
        },
    }
    
    err := config.Setup(ctx, providers, servers, agents)
    if err != nil {
        panic(err)
    }
}
```

## Configuration Reference

### LLM Providers

```yaml
providers:
  - name: string          # Unique identifier for the provider
    type: string          # Currently supports "AZURE"
    token: string         # API token/key
    baseUrl: string       # Base URL for the API
    model: string         # Model name (e.g., "gpt-4o-mini")
    version: string       # API version (for Azure)
```

### MCP Servers

```yaml
servers:
  - name: string          # Unique identifier for the server
    type: string          # "stdio" or "sse"
    # For stdio servers:
    command: []string     # Command to start the server
    # For SSE servers:
    url: string           # Server URL
    headers: []string     # Optional HTTP headers
```

### Agents

```yaml
agents:
  - name: string          # Unique identifier for the agent
    servers:              # List of MCP servers this agent can use
      - name: string      # Server name (must match a server definition)
        allowed_tools:    # Optional: restrict which tools can be used
          - string
    provider: string      # Provider name (must match a provider definition)
```

## Architecture

The library consists of several main components:

- **Config**: Manages system configuration and initialization
- **Server**: Handles MCP server connections (stdio and SSE)
- **Agent**: Implements the agent logic with LLM integration
- **Examples**: Demonstrates usage patterns

### Agent Workflow

1. Agent receives a natural language prompt as `MessageContent`
2. LLM analyzes the prompt and determines if tools are needed
3. If tools are required, the agent executes them via MCP servers
4. Tool responses are fed back to the LLM for final response generation
5. For streaming mode, responses are delivered in real-time as they're generated

## API Reference

### Agent Methods

#### `GenerateContent(ctx context.Context, msgs []llms.MessageContent, addNotFinalResponses bool) (string, []llms.MessageContent)`

Generates content synchronously from a sequence of messages.

**Parameters:**
- `ctx`: Context for the request
- `msgs`: Array of message content representing the conversation
- `addNotFinalResponses`: Whether to include intermediate tool execution details in the response

**Returns:**
- `string`: The generated response text
- `[]llms.MessageContent`: The complete conversation context including the new response

#### `GenerateContentAsStreaming(ctx context.Context, msgs []llms.MessageContent, addNotFinalResponses bool) (chan string, chan llms.MessageContent)`

Generates content with real-time streaming responses.

**Parameters:**
- `ctx`: Context for the request
- `msgs`: Array of message content representing the conversation
- `addNotFinalResponses`: Whether to include intermediate tool execution details in the stream

**Returns:**
- `chan string`: Channel for streaming response chunks
- `chan llms.MessageContent`: Channel for complete message contexts

### Message Content Structure

Messages use the `llms.MessageContent` structure:

```go
type MessageContent struct {
    Role  ChatMessageType  // Human, AI, Tool, etc.
    Parts []ContentPart    // Text, images, tool calls, etc.
}
```

Example usage:
```go
msgs := []llms.MessageContent{
    {
        Role: llms.ChatMessageTypeHuman, 
        Parts: []llms.ContentPart{
            llms.TextContent{Text: "Your question here"}
        }
    },
}
```

## Environment Variables

You can use environment variables in your configuration file:

```yaml
servers:
  - name: my-server
    type: sse
    url: ${MY_SERVER_URL|http://localhost:8080/mcp/events}
```

## Advanced Features

### Tool Execution Tracking

When `addNotFinalResponses` is set to `true`, the agent provides detailed information about tool execution:

- `[tool_usage] tool_name`: Indicates which tool is being executed
- `[tool_response] tool_name: response`: Shows the tool's response (truncated if longer than 1000 characters)

This is particularly useful for debugging and understanding the agent's decision-making process.

### Conversation Context Management

Both `GenerateContent` and `GenerateContentAsStreaming` methods return complete conversation contexts, allowing you to:

- Maintain conversation history across multiple interactions
- Implement conversation persistence
- Build complex multi-turn dialogues

Example:
```go
// Initial conversation
msgs := []llms.MessageContent{
    {Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "Hello!"}}},
}

response, conversationContext := agent.GenerateContent(ctx, msgs, false)

// Continue conversation with context
conversationContext = append(conversationContext, llms.MessageContent{
    Role: llms.ChatMessageTypeHuman, 
    Parts: []llms.ContentPart{llms.TextContent{Text: "What was my previous question?"}},
})

response2, updatedContext := agent.GenerateContent(ctx, conversationContext, false)
```

### Real-time Streaming

The streaming feature allows for real-time response delivery:

```go
var textResp <-chan string
var msgsResp <-chan llms.MessageContent
textResp, msgsResp = agent.GenerateContentAsStreaming(ctx, msgs, true)

// Process both channels concurrently
go func() {
    for msg := range msgsResp {
        // Handle conversation context updates
        msgs = append(msgs, msg)
    }
}()

for chunk := range textResp {
    fmt.Print(chunk) // Print each chunk as it arrives
}
```

## Error Handling

The library includes comprehensive error handling:
- Server connection failures are reported during initialization
- Tool execution errors are passed to the LLM for appropriate handling
- Configuration validation ensures all required fields are present
- Streaming operations include error propagation through channels

### Best Practices

1. **Always check for agent existence** before using:
   ```go
   agent, ok := config.SysConfig.Agents["my-agent"]
   if !ok {
       return fmt.Errorf("agent not found")
   }
   ```

2. **Handle streaming channels properly**:
   ```go
   var textResp <-chan string
   var msgsResp <-chan llms.MessageContent
   textResp, msgsResp = agent.GenerateContentAsStreaming(ctx, msgs, true)
   
   // Handle both channels concurrently
   go func() {
       for msg := range msgsResp {
           // Process conversation context
       }
   }()
   
   for chunk := range textResp {
       fmt.Print(chunk)
   }
   ```

3. **Use context for cancellation**:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   
   response, _ := agent.GenerateContent(ctx, msgs, false)
   ```

4. **Manage conversation context** for multi-turn dialogues:
   ```go
   var conversationHistory []llms.MessageContent
   
   // Add user message
   conversationHistory = append(conversationHistory, llms.MessageContent{
       Role: llms.ChatMessageTypeHuman,
       Parts: []llms.ContentPart{llms.TextContent{Text: userInput}},
   })
   
   // Get response and update context
   response, updatedContext := agent.GenerateContent(ctx, conversationHistory, false)
   conversationHistory = updatedContext
   ```

## Dependencies

This project uses several key dependencies:

- **[tmc/langchaingo](https://github.com/tmc/langchaingo)**: LLM integration and conversation management
- **[mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)**: MCP client and server functionality
- **[bytedance/sonic](https://github.com/bytedance/sonic)**: High-performance JSON serialization
- **[gookit/config](https://github.com/gookit/config)**: Configuration management with environment variable support
- **[life4/genesis](https://github.com/life4/genesis)**: Utility functions for slices and collections

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

When contributing, please ensure:
- Your code follows Go best practices
- Include tests for new functionality
- Update documentation for API changes
- Handle errors appropriately
- Consider backwards compatibility

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

For questions and support, please [create an issue](https://github.com/carlossantin/mcp-agents-go/issues) on GitHub.

### Common Issues

1. **Agent not found**: Ensure your `config.yaml` file is properly formatted and the agent name matches
2. **Tool execution failures**: Check that your MCP server is running and accessible
3. **Streaming issues**: Ensure you're properly handling both channels returned by `GenerateContentAsStreaming`
4. **Configuration errors**: Verify all required fields are present and environment variables are set