package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type CityTimeRequest struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称"`
}

type CityTimeResponse struct {
	City string `json:"city"`
	Time string `json:"time"`
	Zone string `json:"zone"`
}

func getCityTime(ctx context.Context, req *CityTimeRequest) (*CityTimeResponse, error) {
	zones := map[string]string{
		"北京": "Asia/Shanghai (UTC+8)",
		"东京": "Asia/Tokyo (UTC+9)",
		"伦敦": "Europe/London (UTC+0)",
		"纽约": "America/New_York (UTC-5)",
	}
	zone := zones[req.City]
	if zone == "" {
		zone = "未知时区"
	}
	return &CityTimeResponse{
		City: req.City,
		Time: "2025-06-01 14:30:00",
		Zone: zone,
	}, nil
}

func main() {
	ctx := context.Background()

	// 创建工具
	timeTool, _ := utils.InferTool("get_city_time", "查询指定城市的当前时间和时区信息", getCityTime)

	// 创建 ToolsNode
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{timeTool},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 模拟模型返回了一条带有工具调用请求的消息
	modelOutput := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{
			{
				ID: "call_001",
				Function: schema.FunctionCall{
					Name:      "get_city_time",
					Arguments: `{"city": "东京"}`,
				},
			},
		},
	}

	// ToolsNode 执行工具调用
	results, err := toolsNode.Invoke(ctx, modelOutput)
	if err != nil {
		log.Fatal(err)
	}

	for _, msg := range results {
		fmt.Printf("角色: %s\n", msg.Role)
		fmt.Printf("工具调用ID: %s\n", msg.ToolCallID)
		fmt.Printf("结果: %s\n", msg.Content)
	}
}
