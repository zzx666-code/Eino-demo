package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	cfg := openai.DefaultConfig("sk-95eec8f256f34cb188a42e51c8e0e200")
	cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	client := openai.NewClientWithConfig(cfg)

	// 用一个 slice 维护完整的对话历史
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一位友好的Go语言助手，回答简洁明了。",
		},
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("开始对话（输入 quit 退出）：")

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

		// 把用户的新消息追加到历史中
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		})

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    "qwen3.6-plus",
				Messages: messages, // 每次都带上完整历史
			},
		)
		if err != nil {
			log.Printf("API调用失败: %v", err)
			continue
		}
		fmt.Println(messages)
		reply := resp.Choices[0].Message.Content
		fmt.Printf("助手: %s\n", reply)

		// 把模型的回复也追加到历史中，这样下一轮对话就有了完整的上下文
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: reply,
		})
	}
}
