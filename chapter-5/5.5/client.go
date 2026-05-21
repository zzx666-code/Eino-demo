package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
)

func main() {
	ctx := context.Background()

	// 连接到 MCP Server
	cli, err := client.NewSSEMCPClient("http://localhost:8080/sse")
	if err != nil {
		log.Fatal(err)
	}
	if err := cli.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// 初始化握手
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "eino-bridge-demo",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		log.Fatal(err)
	}

	// 关键一步：用 Eino 桥接组件把 MCP 工具转换为 Eino Tool
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{Cli: cli})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("成功桥接 %d 个 MCP 工具为 Eino Tool:\n\n", len(tools))
	for i, t := range tools {
		info, _ := t.Info(ctx)
		fmt.Printf("[%d] 名称: %s\n    描述: %s\n    参数Schema: %v\n\n",
			i+1, info.Name, info.Desc, info.ParamsOneOf)
	}
}
