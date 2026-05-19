package main

//
//import (
//	"context"
//	"fmt"
//	"github.com/cloudwego/eino-ext/components/model/openai"
//	"github.com/cloudwego/eino/components/tool"
//	"github.com/cloudwego/eino/components/tool/utils"
//	"github.com/cloudwego/eino/compose"
//	"github.com/cloudwego/eino/flow/agent/react"
//	"github.com/cloudwego/eino/schema"
//	"log"
//)
//
//// 天气查询的入参
//type WeatherRequest struct {
//	City string `json:"city"`
//}
//
//// 天气查询的返回
//type WeatherResponse struct {
//	City    string `json:"city"`
//	Temp    string `json:"temp"`
//	Weather string `json:"weather"`
//}
//
//func getWeather(ctx context.Context, req *WeatherRequest) (*WeatherResponse, error) {
//	// 模拟天气数据，实际项目中替换为真实 API 调用
//	mockData := map[string]WeatherResponse{
//		"北京": {City: "北京", Temp: "22°C", Weather: "晴"},
//		"上海": {City: "上海", Temp: "26°C", Weather: "多云"},
//		"深圳": {City: "深圳", Temp: "30°C", Weather: "阵雨"},
//	}
//	if data, ok := mockData[req.City]; ok {
//		return &data, nil
//	}
//	return &WeatherResponse{City: req.City, Temp: "未知", Weather: "未知"}, nil
//}
//
//func main() {
//	ctx := context.Background()
//
//	// 1. 创建 ChatModel（接入通义千问）
//	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
//		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
//		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
//		Model:   "qwen-plus",
//	})
//	if err != nil {
//		log.Fatalf("创建 ChatModel 失败: %v", err)
//	}
//
//	// 2. 创建天气查询工具
//	weatherTool := utils.NewTool(
//		&schema.ToolInfo{
//			Name: "get_weather",
//			Desc: "查询指定城市的实时天气信息，包括温度和天气状况",
//			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
//				"city": {
//					Type:     schema.String,
//					Desc:     "要查询天气的城市名称，如：北京、上海、深圳",
//					Required: true,
//				},
//			}),
//		},
//		getWeather,
//	)
//
//	// 3. 创建 ReAct Agent
//	agent, err := react.NewAgent(ctx, &react.AgentConfig{
//		ToolCallingModel: chatModel,
//		ToolsConfig: compose.ToolsNodeConfig{
//			Tools: []tool.BaseTool{weatherTool},
//		},
//		MaxStep: 3,
//		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
//			messages := make([]*schema.Message, 0, len(input)+1)
//			messages = append(messages, schema.SystemMessage(
//				"你是一个热情的天气助手，回答简洁但有温度。天气好就鼓励用户出去玩，天气不好就贴心提醒带伞或注意防晒。",
//			))
//			messages = append(messages, input...)
//			return messages
//		},
//	})
//	if err != nil {
//		log.Fatalf("创建 Agent 失败: %v", err)
//	}
//
//	// 4. 向 Agent 提问
//	answer, err := agent.Generate(ctx, []*schema.Message{
//		schema.UserMessage("北京今天天气怎么样？"),
//		schema.SystemMessage("你是一个可爱的天气助手，请根据用户的提问，调用工具查询天气信息。"),
//	})
//
//	if err != nil {
//		log.Fatalf("Agent 执行失败: %v", err)
//	}
//
//	fmt.Println("Agent 回答:", answer.Content)
//}
