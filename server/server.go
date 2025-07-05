package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type MCPServer struct {
	Name    string              `json:"name"`
	Type    string              `json:"type"`              // e.g., local, sse, stdio
	Command []string            `json:"command,omitempty"` // Command to start the server (for stdio type)
	URL     string              `json:"url,omitempty"`     // URL for the server connection (for sse type)
	Headers []string            `json:"headers,omitempty"` // Headers for the server connection (for sse type)
	Client  mcpclient.MCPClient `json:"client"`            // The MCP client used to communicate with this server
}

// NewMCPServer creates a new MCPServer instance and initializes the MCP client.
// It supports different server types: stdio, sse
// The command is used for stdio type servers, while URL and headers are used for sse type servers.
func NewMCPServer(ctx context.Context, name, serverType string, command []string, url string, headers []string) (*MCPServer, error) {
	s := &MCPServer{
		Name:    name,
		Type:    serverType,
		Command: command,
		URL:     url,
		Headers: headers,
	}

	cli, err := s.createMCPClient(ctx)
	if err != nil {
		return nil, err
	}

	s.Client = cli

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-agents-go",
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = s.Client.Initialize(ctx, initRequest)
	if err != nil {
		fmt.Printf("Failed to initialize MCP client for server %s: %+v\n", name, err)
		return nil, fmt.Errorf("failed to initialize MCP client: %v", err)
	}

	return s, nil
}

func (s *MCPServer) GetTransportType() string {
	switch s.Type {
	case "local", "stdio":
		return "stdio"
	case "sse":
		return "sse"
	default:
		return "stdio"
	}
}

func (m *MCPServer) createMCPClient(ctx context.Context) (mcpclient.MCPClient, error) {
	transportType := m.GetTransportType()

	switch transportType {
	case "stdio":
		// STDIO client
		var env []string
		var command string
		var args []string

		// Handle command and environment
		if len(m.Command) > 0 {
			command = m.Command[0]
			if len(m.Command) > 1 {
				args = m.Command[1:]
			}
		}

		stdioClient, err := mcpclient.NewStdioMCPClient(command, env, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdio client: %v", err)
		}

		// Add a brief delay to allow the process to start and potentially fail
		time.Sleep(100 * time.Millisecond)

		// TODO: Add process health check here if the mcp-go library exposes process info
		// For now, we rely on the timeout in initializeClient to catch dead processes

		return stdioClient, nil
	case "sse":
		// SSE client
		var options []transport.ClientOption

		// Add headers if specified
		if len(m.Headers) > 0 {
			headers := make(map[string]string)
			for _, header := range m.Headers {
				parts := strings.SplitN(header, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					headers[key] = value
				}
			}
			if len(headers) > 0 {
				options = append(options, transport.WithHeaders(headers))
			}
		}

		sseClient, err := client.NewSSEMCPClient(m.URL, options...)
		if err != nil {
			return nil, err
		}

		// Start the SSE client
		if err := sseClient.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start SSE client: %v", err)
		}

		return sseClient, nil

	default:
		return nil, fmt.Errorf("unsupported transport type '%s' for server %s", transportType, m.Name)
	}
}
