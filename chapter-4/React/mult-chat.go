package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type WeatherRequest1 struct {
	City string `json:"city"`
}

type WeatherResponse1 struct {
	City    string `json:"city"`
	Temp    string `json:"temp"`
	Weather string `json:"weather"`
}

func getWeather1(ctx context.Context, req *WeatherRequest1) (*WeatherResponse1, error) {
	mockData := map[string]WeatherResponse1{
		"北京": {City: "北京", Temp: "22°C", Weather: "晴"},
		"上海": {City: "上海", Temp: "26°C", Weather: "多云"},
	}
	if data, ok := mockData[req.City]; ok {
		return &data, nil
	}
	return &WeatherResponse1{City: req.City, Temp: "未知", Weather: "未知"}, nil
}

func main() {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatalf("创建 ChatModel 失败: %v", err)
	}

	weatherTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "get_weather",
			Desc: "查询指定城市的实时天气信息",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"city": {Type: schema.String, Desc: "城市名称", Required: true},
			}),
		},
		getWeather1,
	)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{weatherTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			messages := make([]*schema.Message, 0, len(input)+1)
			messages = append(messages, schema.SystemMessage("你是一个天气助手，回答简洁。"))
			messages = append(messages, input...)
			return messages
		},
	})
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}

	// 维护对话历史
	history := make([]*schema.Message, 0)

	// 第一轮对话
	history = append(history, schema.UserMessage("北京天气怎么样？"))
	answer1, err := agent.Generate(ctx, history)
	if err != nil {
		log.Fatalf("执行失败: %v", err)
	}
	fmt.Println("第一轮回答:", answer1.Content)
	history = append(history, answer1) // 把 Agent 的回答也加入历史

	// 第二轮对话——追问，不需要重复说城市
	history = append(history, schema.UserMessage("那上海呢？"))
	answer2, err := agent.Generate(ctx, history)
	if err != nil {
		log.Fatalf("执行失败: %v", err)
	}
	fmt.Println("第二轮回答:", answer2.Content)
	history = append(history, answer2)

	// 第三轮对话——基于前两轮的对比问题
	history = append(history, schema.UserMessage("哪个城市更热？"))
	answer3, err := agent.Generate(ctx, history)
	if err != nil {
		log.Fatalf("执行失败: %v", err)
	}
	fmt.Println("第三轮回答:", answer3.Content)
}
