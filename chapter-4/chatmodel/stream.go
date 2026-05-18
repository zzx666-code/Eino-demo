package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	temp := float32(0.1)
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:      os.Getenv("DASHSCOPE_API_KEY"),
		Model:       "qwen-plus",
		Temperature: &temp,
	})
	if err != nil {
		log.Fatal(err)
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一个技术博主，文风活泼。"),
		schema.UserMessage("用100字介绍 Go 的错误处理机制"),
	}

	start := time.Now()

	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	firstChunk := true
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if firstChunk {
			fmt.Printf("[首 Token 延迟: %dms]\n", time.Since(start).Milliseconds())
			firstChunk = false
		}
		fmt.Print(chunk.Content)
	}

	fmt.Printf("\n[总耗时: %dms]\n", time.Since(start).Milliseconds())
}

func ptr[T any](v T) *T { return &v }
