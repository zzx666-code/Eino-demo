package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"log"
)

// ========== 工具定义 ==========

type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"城市名称，如北京、上海"`
}

type WeatherOutput struct {
	City    string `json:"city"`
	Temp    int    `json:"temp"`
	Weather string `json:"weather"`
	Wind    string `json:"wind"`
}

func getWeather(ctx context.Context, input *WeatherInput) (*WeatherOutput, error) {
	data := map[string]WeatherOutput{
		"北京": {City: "北京", Temp: 22, Weather: "晴", Wind: "北风3级"},
		"上海": {City: "上海", Temp: 26, Weather: "多云", Wind: "东南风2级"},
		"成都": {City: "成都", Temp: 28, Weather: "阴", Wind: "微风"},
	}
	if w, ok := data[input.City]; ok {
		return &w, nil
	}
	return &WeatherOutput{City: input.City, Temp: 0, Weather: "暂无数据", Wind: "暂无数据"}, nil
}

type TranslateInput struct {
	Text       string `json:"text" jsonschema:"required" jsonschema_description:"要翻译的文本"`
	TargetLang string `json:"target_lang" jsonschema:"required,enum=en,enum=ja,enum=ko" jsonschema_description:"目标语言：en英语，ja日语，ko韩语"`
}

type TranslateOutput struct {
	Original   string `json:"original"`
	Translated string `json:"translated"`
	Lang       string `json:"lang"`
}

func translateText(ctx context.Context, input *TranslateInput) (*TranslateOutput, error) {
	// 模拟翻译
	translations := map[string]string{
		"en": "Hello, today's weather is great!",
		"ja": "こんにちは、今日の天気はいいですね！",
		"ko": "안녕하세요, 오늘 날씨가 좋네요!",
	}
	result := translations[input.TargetLang]
	if result == "" {
		result = "[Translation not available]"
	}
	return &TranslateOutput{
		Original:   input.Text,
		Translated: result,
		Lang:       input.TargetLang,
	}, nil
}

func main() {
	ctx := context.Background()

	// 1. 创建 ChatModel
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  "sk-95eec8f256f34cb188a42e51c8e0e200",
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 创建工具
	weatherTool, _ := utils.InferTool("get_weather", "查询指定城市的实时天气，返回温度、天气状况和风力信息", getWeather)
	translateTool, _ := utils.InferTool("translate", "将文本翻译成指定的目标语言", translateText)

	// 3. 获取工具信息，绑定到模型
	weatherInfo, _ := weatherTool.Info(ctx)
	translateInfo, _ := translateTool.Info(ctx)
	toolInfos := []*schema.ToolInfo{weatherInfo, translateInfo}
	// 告诉，模型有哪些工具可以使用
	chatModel, err := cm.WithTools(toolInfos)
	if err != nil {
		log.Fatal(err)
	}

	// 4. 创建 ToolsNode， 工具执行器
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{weatherTool, translateTool},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 5. 构建对话消息
	messages := []*schema.Message{
		schema.SystemMessage("你是一个多功能助手。请根据用户需求选择合适的工具。"),
		schema.UserMessage("帮我查一下北京今天的天气"),
	}

	fmt.Println("用户: 帮我查一下北京今天的天气")
	fmt.Println()

	// 6. 第一轮：模型决定调用工具
	resp, err := chatModel.Generate(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}
	if len(resp.ToolCalls) > 0 {
		fmt.Printf("模型决定调用工具: %s\n", resp.ToolCalls[0].Function.Name)
		fmt.Printf("调用参数: %s\n\n", resp.ToolCalls[0].Function.Arguments)

		// 7. ToolsNode 执行工具
		toolResults, err := toolsNode.Invoke(ctx, resp)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("工具返回: %s\n\n", toolResults[0].Content)

		// 8. 把模型的工具调用请求和工具结果追加到消息历史
		messages = append(messages, resp)           // 模型的工具调用消息
		messages = append(messages, toolResults...) // 工具执行结果

		// 9. 第二轮：模型根据工具结果生成最终回答
		finalResp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("助手: %s\n", finalResp.Content)
	} else {
		fmt.Printf("助手: %s\n", resp.Content)
	}
}
