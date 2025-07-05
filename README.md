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
    
    response := agent.GenerateContent(ctx, "What tools are available?")
    fmt.Println(response)
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

1. Agent receives a natural language prompt
2. LLM analyzes the prompt and determines if tools are needed
3. If tools are required, the agent executes them via MCP servers
4. Results are fed back to the LLM for final response generation

## Environment Variables

You can use environment variables in your configuration file:

```yaml
servers:
  - name: my-server
    type: sse
    url: ${MY_SERVER_URL|http://localhost:8080/mcp/events}
```

## Error Handling

The library includes comprehensive error handling:
- Server connection failures are reported during initialization
- Tool execution errors are passed to the LLM for appropriate handling
- Configuration validation ensures all required fields are present

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

For questions and support, please [create an issue](https://github.com/carlossantin/mcp-agents-go/issues) on GitHub.