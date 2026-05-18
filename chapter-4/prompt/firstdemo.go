package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 创建模板：System 消息里有 {role} 变量，User 消息里有 {question} 变量
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{role}，请用专业且易懂的语言回答问题。"),
		schema.UserMessage("{question}"),
		schema.MessagesPlaceholder("chathistory", false),
	)
	history := []*schema.Message{
		schema.UserMessage("我叫张三"),
		schema.AssistantMessage("你好张三！很高兴认识你。", nil),
		schema.UserMessage("我最喜欢的编程语言是Go"),
		schema.AssistantMessage("Go语言确实很棒，简洁高效！", nil),
	}

	// 准备变量
	variables := map[string]any{
		"role":        "Go语言技术顾问",
		"question":    "goroutine 和线程有什么区别？",
		"chathistory": history,
	}

	// 格式化模板，生成消息列表
	messages, err := template.Format(ctx, variables)
	if err != nil {
		log.Fatal(err)
	}

	// 查看生成的消息
	for _, msg := range messages {
		fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
	}
}
