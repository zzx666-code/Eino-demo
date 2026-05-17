package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// $env:DASHSCOPE_API_KEY="sk-your-api-key-here"
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

	messages := []*schema.Message{
		schema.SystemMessage("你是一个Go语言专家，擅长深入浅出地讲解技术概念。"),
		schema.UserMessage("请用200字左右解释 Go 语言的 channel 是什么，以及它在并发编程中的作用。"),
	}

	// 调用 Stream 方法，获取流式读取器
	stream, err := chatModel.Stream(ctx, messages)
	if err != nil {
		log.Fatalf("调用流式接口失败: %v", err)
	}
	defer stream.Close()

	fmt.Println("模型回复（流式）：")

	// 循环读取流式数据块
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// 流结束
			break
		}
		if err != nil {
			log.Fatalf("读取流数据失败: %v", err)
		}
		// 每收到一块就立即输出，不换行
		fmt.Print(chunk.Content)
	}

	fmt.Println() // 最后换行
}
