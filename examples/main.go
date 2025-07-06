package main

import (
	"context"
	"fmt"

	"github.com/carlossantin/mcp-agents-go/config"
	"github.com/tmc/langchaingo/llms"
)

func main() {
	ctx := context.Background()

	config.SetupFromFile(ctx, "config.yaml")

	ag, ok := config.SysConfig.Agents["my-agent"]
	if ok {
		msgs := []llms.MessageContent{
			{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "Give me the current dollar to real exchange rate in BRL."}}},
		}

		// resp, _ := ag.GenerateContent(ctx, msgs, true)
		// fmt.Println(resp)
		chanResp, _ := ag.GenerateContentAsStreaming(ctx, msgs, true)
		for resp := range chanResp {
			fmt.Print(resp)
		}
	} else {
		panic("Agent my-agent not found in configuration")
	}
}
