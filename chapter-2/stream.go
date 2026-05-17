package main

import (
	"context"
	"errors"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"io"
	"log"
)

func main() {
	cfg := openai.DefaultConfig("sk-95eec8f256f34cb188a42e51c8e0e200")
	cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	client := openai.NewClientWithConfig(cfg)

	// 用 CreateChatCompletionStream 替代 CreateChatCompletion
	stream, err := client.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "qwen3.6-plus",
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "你是一位Go语言专家，回答详细且有条理。"},
				{Role: openai.ChatMessageRoleUser, Content: "请介绍一下Go语言的goroutine调度模型。"},
			},
		},
	)
	if err != nil {
		log.Fatalf("创建流失败: %v", err)
	}
	defer stream.Close()

	fmt.Print("助手: ")
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// 流结束
			fmt.Println()
			break
		}
		if err != nil {
			log.Fatalf("接收流数据出错: %v", err)
		}
		// 每收到一个 Token 就立刻打印
		fmt.Print(response.Choices[0].Delta.Content)
	}
}
