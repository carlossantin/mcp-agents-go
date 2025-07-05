package config

import (
	"context"
	"fmt"

	"github.com/carlossantin/mcp-agents-go/agent"
	"github.com/carlossantin/mcp-agents-go/server"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/life4/genesis/slices"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var SysConfig = &SystemConfig{
	LLMModels: map[string]llms.Model{},
	Servers:   map[string]server.MCPServer{},
	Agents:    map[string]agent.MCPAgent{},
}

type SystemConfig struct {
	LLMModels map[string]llms.Model
	Servers   map[string]server.MCPServer
	Agents    map[string]agent.MCPAgent
}

type LLMProvider struct {
	Name    string `mapstructure:"name"`
	Type    string `mapstructure:"type"` // e.g., AZURE, OPENAI
	Token   string `mapstructure:"token"`
	BaseURL string `mapstructure:"baseUrl"`
	Model   string `mapstructure:"model"`   // e.g., gpt-4o-mini
	Version string `mapstructure:"version"` // e.g., 2025-01-01-preview
}

type MCPServer struct {
	Name    string   `mapstructure:"name"`
	Type    string   `mapstructure:"type"`    // e.g., stdio, sse
	Command []string `mapstructure:"command"` // Command to start the server (stdio)
	URL     string   `mapstructure:"url"`     // URL for the server connection (sse)
	Headers []string `mapstructure:"headers"` // Headers for the server connection (sse)
}

type MCPAgent struct {
	Name            string                 `mapstructure:"name"`
	MCPAgentServers []agent.MCPAgentServer `mapstructure:"servers"`  // List of MCP servers used by this agent
	Provider        string                 `mapstructure:"provider"` // Name of the LLM provider to use
}

func Setup(ctx context.Context, providers []LLMProvider, servers []MCPServer, agents []MCPAgent) error {
	providersLLMs, err := initLLMProviders(providers)
	if err != nil {
		return err
	}
	SysConfig.LLMModels = providersLLMs

	serversMap, err := initMCPServers(ctx, servers)
	if err != nil {
		return err
	}
	SysConfig.Servers = serversMap

	mapAgents, err := initAgents(ctx, agents)
	if err != nil {
		return err
	}

	SysConfig.Agents = mapAgents
	return nil
}

func initLLMProviders(providers []LLMProvider) (map[string]llms.Model, error) {
	providersLLMs := map[string]llms.Model{}

	for _, provider := range providers {
		switch provider.Type {
		case "AZURE":
			llmModel, err := openai.New(
				openai.WithToken(provider.Token),
				openai.WithBaseURL(provider.BaseURL),
				openai.WithModel(provider.Model),
				openai.WithAPIType(openai.APITypeAzure),
				openai.WithAPIVersion(provider.Version),
			)
			if err != nil {
				fmt.Printf("Error creating LLM model for provider %s: %+v", provider.Name, err)
				return nil, err
			} else {
				providersLLMs[provider.Name] = llmModel
				fmt.Printf("LLM model for provider %q created successfully\n", provider.Name)
			}
		}
	}

	return providersLLMs, nil
}

func initMCPServers(ctx context.Context, servers []MCPServer) (map[string]server.MCPServer, error) {
	serversMap := make(map[string]server.MCPServer)

	for _, sv := range servers {
		mcpServer, err := server.NewMCPServer(ctx, sv.Name, sv.Type, sv.Command, sv.URL, sv.Headers)
		if err != nil {
			fmt.Printf("Error creating MCP server %s: %+v\n", sv.Name, err)
			return nil, err
		}
		serversMap[sv.Name] = *mcpServer
		fmt.Printf("MCP server %q created successfully\n", sv.Name)
	}

	return serversMap, nil
}

func initAgents(ctx context.Context, agents []MCPAgent) (map[string]agent.MCPAgent, error) {
	mapAgents := make(map[string]agent.MCPAgent)

	for _, ag := range agents {
		agentServers := slices.Map(ag.MCPAgentServers, func(srv agent.MCPAgentServer) agent.MCPAgentServer {
			return agent.MCPAgentServer{
				Name:         srv.Name,
				AllowedTools: srv.AllowedTools,
			}
		})

		agentMCPServers := slices.Map(ag.MCPAgentServers, func(srv agent.MCPAgentServer) server.MCPServer {
			mcpServer, ok := SysConfig.Servers[srv.Name]
			if !ok {
				fmt.Printf("Error finding MCP server %s for agent %s\n", srv.Name, ag.Name)
				return server.MCPServer{} // Return an empty server if not found
			}
			return mcpServer
		})

		// Filter out any empty servers
		agentMCPServers = slices.Filter(agentMCPServers, func(srv server.MCPServer) bool {
			return srv.Name != ""
		})

		mcpAgent := agent.NewMCPAgent(
			ctx,
			ag.Name,
			agentServers,
			agentMCPServers,
			ag.Provider,
			SysConfig.LLMModels[ag.Provider],
		)

		mapAgents[ag.Name] = *mcpAgent
		fmt.Printf("MCP Agent %q created successfully with servers: %q using provider %q\n", ag.Name, ag.MCPAgentServers, ag.Provider)
	}

	return mapAgents, nil
}

func SetupFromFile(ctx context.Context, configFileName string) error {
	var providers []LLMProvider
	config.WithOptions(config.ParseEnv)
	config.AddDriver(yaml.Driver)
	err := config.LoadFiles(configFileName)
	if err != nil {
		fmt.Printf("Error loading config file %s: %+v\n", configFileName, err)
		return err
	}
	config.BindStruct("providers", &providers)

	var servers []MCPServer
	config.BindStruct("servers", &servers)

	var agents []MCPAgent
	config.BindStruct("agents", &agents)

	return Setup(ctx, providers, servers, agents)
}
