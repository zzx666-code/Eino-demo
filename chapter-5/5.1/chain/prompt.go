package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	// 创建模型
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
		Model:   "qwen-plus",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 创建 Prompt 模板
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{role}，请用简洁专业的语言回答问题。"),
		schema.UserMessage("{question}"),
	)

	// 构建 Chain：模板 → 模型
	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.
		AppendChatTemplate(tpl).
		AppendChatModel(model)

	// 编译
	runner, err := chain.Compile(ctx)
	if err != nil {
		log.Fatal("编译失败:", err)
	}

	// 运行
	result, err := runner.Invoke(ctx, map[string]any{
		"role":     "Go语言专家",
		"question": "Go的channel和mutex各自适合什么场景？",
	})
	if err != nil {
		log.Fatal("运行失败:", err)
	}

	fmt.Println(result.Content)
}
