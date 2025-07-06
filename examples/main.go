package main

import (
	"context"
	"fmt"

	"github.com/carlossantin/mcp-agents-go/config"
)

func main() {
	ctx := context.Background()

	config.SetupFromFile(ctx, "config.yaml")

	ag, ok := config.SysConfig.Agents["my-agent"]
	if ok {
		// fmt.Println(ag.GenerateContent(ctx, "Give me the current dollar to real exchange rate in BRL.", true))
		chanResp := ag.GenerateContentAsStreaming(ctx, "Give me the current dollar to real exchange rate in BRL.", true)
		for resp := range chanResp {
			fmt.Print(resp)
		}
	} else {
		panic("Agent my-agent not found in configuration")
	}
}
