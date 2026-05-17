package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		Model:   "qwen3.6-plus",
	})
	if err != nil {
		panic(err)
	}
	history := []*schema.Message{
		schema.SystemMessage("你是一个Go语言专家，擅长深入浅出地讲解技术概念。"),
	}
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "quit" {
			fmt.Println("再见！")
			break
		}
		if input == "" {
			continue
		}
		history = append(history, schema.UserMessage(input))
		stream, err := chatModel.Stream(ctx, history)
		if err != nil {
			log.Printf("failed")
			continue
		}
		fmt.Print("助手: ")
		var reply strings.Builder

		for {
			chunk, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("读取失败: %v", err)
				break
			}
			fmt.Print(chunk.Content)
			reply.WriteString(chunk.Content)
		}
		stream.Close()
		fmt.Println()
		history = append(history, &schema.Message{
			Role:    schema.Assistant,
			Content: reply.String(),
		})
		fmt.Println(history)
	}
}
