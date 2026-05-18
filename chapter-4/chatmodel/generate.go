package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
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
	start := time.Now()
	messages := []*schema.Message{
		schema.SystemMessage("你是一个技术博主，文风活泼。"),
		schema.UserMessage("用100字介绍 Go 的错误处理机制"),
	}

	// Generate 适合需要完整输出的场景，比如解析 JSON
	resp, err := cm.Generate(ctx, messages, model.WithTemperature(0))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("模型输出的 JSON：")
	fmt.Println(resp.Content)

	// resp.ResponseMeta 包含 Token 用量等元信息
	if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
		fmt.Printf("\nToken 用量 - 输入: %d, 输出: %d, 总计: %d\n",
			resp.ResponseMeta.Usage.PromptTokens,
			resp.ResponseMeta.Usage.CompletionTokens,
			resp.ResponseMeta.Usage.TotalTokens)
	}
	fmt.Println("总耗时：", time.Since(start))
}
