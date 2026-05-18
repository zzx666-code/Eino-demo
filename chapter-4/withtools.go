package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 定义工具描述信息
	weatherTool := &schema.ToolInfo{
		Name: "get_weather",
		Desc: "查询指定城市的当前天气",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"city": {
				Type:     "string",
				Desc:     "城市名称，如：北京、上海",
				Required: true,
			},
		}),
	}

	// 用 WithTools 绑定工具，返回新实例
	cmWithTools, err := cm.WithTools([]*schema.ToolInfo{weatherTool})
	if err != nil {
		log.Fatal(err)
	}

	// 用绑定了工具的实例来调用
	messages := []*schema.Message{
		schema.SystemMessage("你是一个天气助手。你必须通过调用工具来查询天气，如果没有可用的工具，请直接告诉用户你无法查询实时天气信息，不要编造任何天气数据。"),
		schema.UserMessage("北京今天天气怎么样？"),
	}

	resp, err := cmWithTools.Generate(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// 检查模型是否发起了工具调用
	if len(resp.ToolCalls) > 0 {
		fmt.Println("模型请求调用工具：")
		for _, tc := range resp.ToolCalls {
			println("模型请求调用工ID", tc.ID)
			fmt.Printf("  工具名: %s\n", tc.Function.Name)
			fmt.Printf("  参数: %s\n", tc.Function.Arguments)
		}
	} else {
		fmt.Println("模型直接回复：", resp.Content)
	}

	// 原始的 cm 没有绑定工具，不受影响
	// 使用不同的 messages，明确告知模型它没有任何工具可用
	messagesNoTool := []*schema.Message{
		schema.SystemMessage("你是一个天气助手，但你没有任何工具可以使用。当用户询问天气时，你如果不知道，就要回答不知道，绝对不能编造天气信息。"),
		schema.UserMessage("北京今天天气怎么样？"),
	}
	resp2, _ := cm.Generate(ctx, messagesNoTool)
	fmt.Println("\n未绑定工具的实例回复：", resp2.Content)
	fmt.Println(resp2.ResponseMeta.Usage.TotalTokens)
}
