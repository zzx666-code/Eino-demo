package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// ========== 天气查询工具 ==========

type WeatherRequest struct {
	City string `json:"city" jsonschema:"city"`
}

type WeatherResponse struct {
	City    string  `json:"city"`
	Temp    float64 `json:"temp"`
	Weather string  `json:"weather"`
}

func getWeather(ctx context.Context, req *WeatherRequest) (*WeatherResponse, error) {
	mockData := map[string]WeatherResponse{
		"北京":   {City: "北京", Temp: 22, Weather: "晴"},
		"上海":   {City: "上海", Temp: 26, Weather: "多云"},
		"深圳":   {City: "深圳", Temp: 30, Weather: "阵雨"},
		"哈尔滨": {City: "哈尔滨", Temp: 8, Weather: "小雪"},
	}
	if data, ok := mockData[req.City]; ok {
		return &data, nil
	}
	return &WeatherResponse{City: req.City, Temp: 0, Weather: "暂无数据"}, nil
}

// ========== 数学计算工具 ==========

type CalcRequest struct {
	A  float64 `json:"a"`
	B  float64 `json:"b"`
	Op string  `json:"op"`
}

type CalcResponse struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
}

func calculate(ctx context.Context, req *CalcRequest) (*CalcResponse, error) {
	var result float64
	switch req.Op {
	case "add":
		result = req.A + req.B
	case "subtract":
		result = req.A - req.B
	case "multiply":
		result = req.A * req.B
	case "divide":
		if req.B == 0 {
			return nil, fmt.Errorf("除数不能为零")
		}
		result = req.A / req.B
	default:
		return nil, fmt.Errorf("不支持的运算: %s", req.Op)
	}
	return &CalcResponse{
		Expression: fmt.Sprintf("%.1f %s %.1f", req.A, req.Op, req.B),
		Result:     math.Round(result*100) / 100,
	}, nil
}

// ========== 时间查询工具 ==========

type TimeRequest struct {
	Timezone string `json:"timezone"`
}

type TimeResponse struct {
	Timezone    string `json:"timezone"`
	CurrentTime string `json:"current_time"`
	Date        string `json:"date"`
}

func getCurrentTime(ctx context.Context, req *TimeRequest) (*TimeResponse, error) {
	tz := req.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := time.Now().In(loc)
	return &TimeResponse{
		Timezone:    tz,
		CurrentTime: now.Format("15:04:05"),
		Date:        now.Format("2006-01-02"),
	}, nil
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

	// 创建三个工具
	weatherTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "get_weather",
			Desc: "查询指定城市的实时天气信息，返回温度（数值）和天气状况",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"city": {
					Type:     schema.String,
					Desc:     "城市名称，如：北京、上海、深圳、哈尔滨",
					Required: true,
				},
			}),
		},
		getWeather,
	)

	calcTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "calculator",
			Desc: "对两个数字执行四则运算，返回计算结果",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"a":  {Type: schema.Number, Desc: "第一个数字", Required: true},
				"b":  {Type: schema.Number, Desc: "第二个数字", Required: true},
				"op": {Type: schema.String, Desc: "运算类型", Required: true, Enum: []string{"add", "subtract", "multiply", "divide"}},
			}),
		},
		calculate,
	)

	timeTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "get_current_time",
			Desc: "获取当前时间和日期",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"timezone": {
					Type: schema.String,
					Desc: "时区，如 Asia/Shanghai、America/New_York，默认为中国时区",
				},
			}),
		},
		getCurrentTime,
	)

	// 组装 Agent，注册三个工具
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{weatherTool, calcTool, timeTool},
		},
		MaxStep: 10,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			messages := make([]*schema.Message, 0, len(input)+1)
			messages = append(messages, schema.SystemMessage(
				"你是一个多功能助手，可以查天气、做计算、查时间。回答要准确简洁。",
			))
			messages = append(messages, input...)
			return messages
		},
	})
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}

	// 测试1：需要多次工具调用的问题
	fmt.Println("===== 测试1：多城市天气对比 =====")
	answer1, err := agent.Generate(ctx, []*schema.Message{
		schema.UserMessage("北京和哈尔滨今天的温差是多少度？"),
	})
	if err != nil {
		log.Fatalf("执行失败: %v", err)
	}
	fmt.Println("回答:", answer1.Content)

	// 测试2：需要组合使用工具
	fmt.Println("\n===== 测试2：综合查询 =====")
	answer2, err := agent.Generate(ctx, []*schema.Message{
		schema.UserMessage("现在几点了？另外帮我查一下上海的天气。"),
	})
	if err != nil {
		log.Fatalf("执行失败: %v", err)
	}
	fmt.Println("回答:", answer2.Content)
}
