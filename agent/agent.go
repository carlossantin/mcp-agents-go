package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/carlossantin/mcp-agents-go/server"
	"github.com/life4/genesis/slices"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tmc/langchaingo/llms"
)

type MCPAgent struct {
	Name           string                `json:"name"`
	MCPServerNames []MCPAgentServer      `json:"mcp_servers"` // List of MCP servers used by this agent
	MCPServerTools map[string][]mcp.Tool `json:"-"`           // Map of MCP server names to their allowed tools
	mcpServers     []*server.MCPServer   `json:"-"`
	Provider       string                `json:"provider"` // Name of the LLM provider to use
	LLMModel       llms.Model            `json:"-"`
}

func NewMCPAgent(ctx context.Context, name string, mcpServersForAgent []MCPAgentServer, servers []server.MCPServer, provider string, llmModel llms.Model) *MCPAgent {
	ag := &MCPAgent{
		Name:           name,
		MCPServerNames: mcpServersForAgent,
		MCPServerTools: make(map[string][]mcp.Tool),
		mcpServers:     []*server.MCPServer{},
		Provider:       provider,
		LLMModel:       llmModel,
	}

	for _, srv := range mcpServersForAgent {
		mcpServer, err := slices.Find(servers, func(s server.MCPServer) bool {
			return s.Name == srv.Name
		})
		if err == nil {
			ag.mcpServers = append(ag.mcpServers, &mcpServer)
			toolsRes, err := mcpServer.Client.ListTools(ctx, mcp.ListToolsRequest{})
			if err == nil && toolsRes != nil {
				allowedTools := slices.Filter(toolsRes.Tools, func(tool mcp.Tool) bool {
					return len(srv.AllowedTools) == 0 || slices.Contains(srv.AllowedTools, tool.Name)
				})
				if _, ok := ag.MCPServerTools[srv.Name]; !ok {
					ag.MCPServerTools[srv.Name] = []mcp.Tool{}
				}
				ag.MCPServerTools[srv.Name] = append(ag.MCPServerTools[srv.Name], allowedTools...)
				allowedToolNames := slices.Map(allowedTools, func(tool mcp.Tool) string {
					return tool.Name
				})
				fmt.Printf("Agent %s is allowed to use tools: %s on server %s\n", ag.Name, strings.Join(allowedToolNames, ", "), srv.Name)
			}
		}
	}

	return ag
}

type MCPAgentServer struct {
	Name         string   `json:"name" mapstructure:"name"`                   // Name of the MCP server
	AllowedTools []string `json:"allowed_tools" mapstructure:"allowed_tools"` // List of tools allowed for this agent server
}

// InvokableRun executes the tool by mapping back to the original name and server
func (m *MCPAgent) ExecuteTool(ctx context.Context, toolName, argumentsInJSON string) (string, error) {
	// Handle empty or invalid JSON arguments
	var arguments any
	if argumentsInJSON == "" || argumentsInJSON == "{}" {
		arguments = nil
	} else {
		// Validate that argumentsInJSON is valid JSON before using it
		var temp any
		if err := json.Unmarshal([]byte(argumentsInJSON), &temp); err != nil {
			return "", fmt.Errorf("invalid JSON arguments: %w", err)
		}
		arguments = json.RawMessage(argumentsInJSON)
	}

	serverName := toolName[:strings.Index(toolName, "__")]
	toolName = toolName[strings.Index(toolName, "__")+2:] // Remove the server prefix
	toolServer, err := slices.Find(m.mcpServers, func(srv *server.MCPServer) bool {
		return srv.Name == serverName
	})
	if err != nil {
		return "", err
	}

	result, err := toolServer.Client.CallTool(ctx, mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: struct {
			Name      string    `json:"name"`
			Arguments any       `json:"arguments,omitempty"`
			Meta      *mcp.Meta `json:"_meta,omitempty"`
		}{
			Name:      toolName, // Use original name, not prefixed
			Arguments: arguments,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call mcp tool: %w", err)
	}

	marshaledResult, err := sonic.MarshalString(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mcp tool result: %w", err)
	}

	// If the MCP server returned an error, we still return the error content as the response
	// to the LLM so it can see what went wrong. The error will be shown to the user via
	// the UI callbacks, but the LLM needs to see the actual error details to continue
	// the conversation appropriately.
	return marshaledResult, nil
}

func (m *MCPAgent) GenerateContent(ctx context.Context, prompt string) string {
	msgs := []llms.MessageContent{
		{Role: "human", Parts: []llms.ContentPart{llms.TextContent{Text: prompt}}},
	}

	tools := m.ExtractToolsFromAgent()
	resp, err := m.LLMModel.GenerateContent(ctx, msgs, llms.WithTools(tools))
	if err != nil {
		panic(err)
	}

	toolCalls := resp.Choices[0].ToolCalls

	response := ""

	if len(toolCalls) > 0 {
		for _, suggestedTool := range toolCalls {
			fmt.Printf("Suggested to use tool %s\n", suggestedTool.FunctionCall.Name)
			toolRes, err := m.ExecuteTool(ctx, suggestedTool.FunctionCall.Name, suggestedTool.FunctionCall.Arguments)
			if err != nil {
				panic(err)
			}

			msgs = append(msgs, llms.MessageContent{
				Role: "ai",
				Parts: []llms.ContentPart{
					suggestedTool,
				},
			})

			msgs = append(msgs, llms.MessageContent{
				Role: "tool",
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: suggestedTool.ID,
						Content:    toolRes,
					},
				},
			})
		}

		resp, err := m.LLMModel.GenerateContent(ctx, msgs, llms.WithTools(tools))
		if err != nil {
			panic(err)
		}

		response = resp.Choices[0].Content
	} else {
		response = resp.Choices[0].Content
	}

	return response
}

func (m *MCPAgent) ExtractToolsFromAgent() []llms.Tool {
	result := []llms.Tool{}
	for serverName, agTs := range m.MCPServerTools {
		for _, agT := range agTs {
			params := map[string]interface{}{
				"type":       agT.InputSchema.Type,
				"properties": agT.InputSchema.Properties,
			}

			t := llms.Tool{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        serverName + "__" + agT.Name,
					Description: agT.Description,
					Parameters:  params,
				},
			}

			result = append(result, t)
		}
	}

	return result
}
